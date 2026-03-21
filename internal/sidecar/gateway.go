// Package sidecar implements the RPC Gateway sidecar that bridges ConnectRPC and the agent harness process.
package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// Gateway is the RPC Gateway sidecar server.
type Gateway struct {
	agentv1connect.UnimplementedAgentSidecarServiceHandler
	agentv1connect.UnimplementedAgentNotificationServiceHandler
	port      int
	debugMode bool

	mu      sync.RWMutex
	process *AgentProcess
	server  *http.Server
}

// agentLogDir is the directory for agent log files on the PVC.
const agentLogDir = "/workspace/.aot/logs"

// agentLogPath is the full path to the human-readable agent log file on the PVC.
const agentLogPath = agentLogDir + "/agent.log"

// agentJSONLPath is the full path to the raw JSON lines log for machine parsing.
const agentJSONLPath = agentLogDir + "/agent.jsonl"

// agentInputDir is the directory for HITL input files on the PVC.
const agentInputDir = "/workspace/.aot/input"

// agentInputResponsePath is the file where SendInput writes the human's answer.
const agentInputResponsePath = agentInputDir + "/response.txt"

// traceDir is the directory for trace span files on the PVC.
const traceDir = "/workspace/.aot/traces"

// traceSpansPath is the JSONL file for trace spans.
const traceSpansPath = traceDir + "/spans.jsonl"

// TraceSpan represents a single trace span recorded during an agent run.
type TraceSpan struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"traceId,omitempty"`
	ParentID  string                 `json:"parentId,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
	Status    string                 `json:"status,omitempty"` // "ok", "error", "unset"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	HasDiff   bool                   `json:"hasDiff"`
	Diff      *SpanDiff              `json:"diff,omitempty"`
}

// SpanDiff holds the git diff captured for a trace span.
type SpanDiff struct {
	Files []FileDiff `json:"files"`
}

// FileDiff represents a single file's patch within a span diff.
type FileDiff struct {
	Path  string `json:"path"`
	Patch string `json:"patch"`
}

// AgentProcess wraps the agent harness process.
type AgentProcess struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	logFile   *os.File
	jsonlFile *os.File
	state     agentv1.AgentProcessState
	exitError string
	startedAt time.Time
	outputs   []chan *agentv1.AgentOutput
	mu        sync.Mutex
	readerWg  sync.WaitGroup
	// textBuf accumulates text_delta fragments for the human-readable log.
	textBuf strings.Builder
	// pendingQuestion stores the HITL question payload when state is WAITING_FOR_INPUT.
	pendingQuestion string
}

// NewGateway creates a new RPC Gateway sidecar.
func NewGateway(port int) *Gateway {
	// Ensure log directory exists at sidecar startup (5.2)
	if err := os.MkdirAll(agentLogDir, 0o755); err != nil {
		log.Printf("WARNING: failed to create log dir %s: %v", agentLogDir, err)
	}

	// Ensure trace directory exists at sidecar startup (10.4)
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		log.Printf("WARNING: failed to create trace dir %s: %v", traceDir, err)
	}

	debugMode := os.Getenv("AOT_DEBUG_MODE") == "true"
	if debugMode {
		log.Printf("Debug mode — waiting for connections")
	}

	return &Gateway{port: port, debugMode: debugMode}
}

// Start begins listening for ConnectRPC connections from the Control Plane.
func (g *Gateway) Start() error {
	mux := http.NewServeMux()

	path, handler := agentv1connect.NewAgentSidecarServiceHandler(g)
	mux.Handle(path, handler)

	nPath, nHandler := agentv1connect.NewAgentNotificationServiceHandler(g)
	mux.Handle(nPath, nHandler)

	g.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.Printf("RPC Gateway listening on :%d", g.port)
	return g.server.ListenAndServe()
}

// Stop gracefully stops the gateway.
func (g *Gateway) Stop() {
	if g.server != nil {
		_ = g.server.Close()
	}
}

func (g *Gateway) StartAgent(_ context.Context, req *connect.Request[agentv1.StartAgentRequest]) (*connect.Response[agentv1.StartAgentResponse], error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Debug mode: don't launch the agent, just report started (5.3)
	if g.debugMode {
		log.Printf("Debug mode active — skipping agent launch for run %s", req.Msg.AgentRunId)
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: true}), nil
	}

	// If a previous agent is still running, stop it before starting a new one.
	// This handles pipeline stage transitions and retry attempts where the
	// previous agent may not have fully exited yet.
	if g.process != nil && g.process.state == agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING {
		log.Printf("Stopping previous agent before starting new one for run %s", req.Msg.AgentRunId)
		if g.process.cmd.Process != nil {
			_ = g.process.cmd.Process.Signal(os.Interrupt)
			// Give it a moment to clean up, but don't block long.
			done := make(chan struct{})
			go func() {
				_ = g.process.cmd.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = g.process.cmd.Process.Kill()
			}
		}
		g.process = nil
	}

	proc, err := startAgentProcess(req.Msg)
	if err != nil {
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: false, Error: err.Error()}), nil
	}

	g.process = proc

	// Track current stage + workdir for span naming and diff capture.
	resolvedDir := resolveWorkDir(req.Msg.RepoPath)
	setCurrentStage(req.Msg.Stage, resolvedDir)

	// Store parent span ID and trace ID for child span linking
	setCurrentParentSpan(req.Msg.ParentSpanId, req.Msg.TraceId)

	// Store model name for span metadata
	currentModel = os.Getenv("PI_MODEL")
	if currentModel == "" {
		currentModel = "default"
	}

	// Configure git for checkpoint commits
	gitConfigCmd := exec.Command("git", "config", "user.name", "aot-agent")
	gitConfigCmd.Dir = resolvedDir
	_ = gitConfigCmd.Run()
	gitEmailCmd := exec.Command("git", "config", "user.email", "agent@aot.uncworks.io")
	gitEmailCmd.Dir = resolvedDir
	_ = gitEmailCmd.Run()

	// Reset checkpoint state for new stage
	checkpointMu.Lock()
	lastCheckpointSHA = ""
	checkpointMu.Unlock()

	// Write an initial trace span so the traces tab has data immediately.
	stageName := spanPrefix() + ".started"
	appendTraceSpan(TraceSpan{
		ID:        uuid.New().String(),
		TraceID:   getTraceID(),
		ParentID:  getParentSpanID(),
		Name:      stageName,
		Type:      "input",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Metadata:  map[string]interface{}{"stage": req.Msg.Stage, "agentRunId": req.Msg.AgentRunId},
	})

	// CRITICAL: Start pipe readers NOW, before releasing the mutex.
	// monitorProcess uses RLock which would block until this Lock is released,
	// but pi may finish in < 5s, closing stdout before the scanner starts.
	// Solution: start scanning goroutines here under the write lock, then
	// launch the wait/cleanup goroutine separately.
	proc.startReaders()
	go g.waitForProcess(req.Msg.AgentRunId)

	return connect.NewResponse(&agentv1.StartAgentResponse{Started: true}), nil
}

func startAgentProcess(req *agentv1.StartAgentRequest) (*AgentProcess, error) {
	// Use --mode json so pi streams ALL events (tool calls, tool results,
	// text responses) as JSONL to stdout. The old "-p" (print) mode only
	// printed the final text and swallowed tool execution output, which
	// meant commands like "ls" ran but their results never appeared in logs.
	// --no-session avoids persisting session state in the ephemeral container.
	// -p = non-interactive (process and exit), --mode json = stream JSON events
	args := []string{"-p", "--mode", "json", "--no-session"}

	// Load AOT determinism extension for policy enforcement
	const aotExtensionPath = "/opt/aot/extensions/aot-determinism.ts"
	if _, err := os.Stat(aotExtensionPath); err == nil {
		args = append(args, "--extension", aotExtensionPath)
	}

	// Stage-specific system prompt for spec-driven pipeline.
	if sp := stageSystemPrompt(req.GetStage()); sp != "" {
		args = append(args, "--system-prompt", sp)
	}

	args = append(args, req.Prompt)

	// Use model from env if configured
	if model := os.Getenv("PI_MODEL"); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.Command("pi", args...)
	cmd.Dir = resolveWorkDir(req.RepoPath)

	// Inherit current environment and add request-specific vars on top.
	// PI_LOG_LEVEL=debug: emit detailed tool call and LLM response info.
	// PI_ACCEPT_TOS=1: skip interactive TOS prompt in headless mode.
	cmd.Env = append(os.Environ(), "PI_LOG_LEVEL=debug", "PI_ACCEPT_TOS=1")
	if stage := req.GetStage(); stage != "" {
		cmd.Env = append(cmd.Env, "PI_STAGE="+stage)
	}
	// Set PI_ROLE based on pipeline stage so the determinism extension can
	// enforce role-based tool policies (manage vs implement).
	cmd.Env = append(cmd.Env, "PI_ROLE="+stageToRole(req.GetStage()))
	for k, v := range req.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Redirect stdin from /dev/null so pi-coding-agent's readPipedStdin()
	// gets immediate EOF instead of blocking forever on an open pipe.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("open /dev/null: %w", err)
	}
	cmd.Stdin = devNull

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// Open log file for tee-ing agent output to PVC (5.1)
	if err := os.MkdirAll(agentLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logFile, err := os.OpenFile(agentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open agent log: %w", err)
	}
	jsonlFile, err := os.OpenFile(agentJSONLPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("open agent jsonl: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = jsonlFile.Close()
		return nil, fmt.Errorf("start agent: %w", err)
	}
	_ = devNull.Close()

	return &AgentProcess{
		cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
		logFile:   logFile,
		jsonlFile: jsonlFile,
		state:     agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING,
		startedAt: time.Now(),
	}, nil
}

// stageSystemPrompt returns a stage-specific system prompt for spec-driven pipelines.
// Returns empty string for default/single mode (uses pi's built-in prompt).
func stageSystemPrompt(stage string) string {
	switch stage {
	case "plan":
		return `You are a manage agent (unc). Create an OpenSpec change following the instructions in your prompt.

Key rules:
- The OpenSpec workspace is at /workspace — run ALL openspec commands from /workspace (cd /workspace && openspec ...)
- The repo source code is in /workspace/<reponame>/ — read code there to understand the codebase
- Use openspec CLI to get templates: openspec instructions proposal/specs/tasks --change <name>
- Write spec files to the paths specified in your prompt (under /workspace/openspec/changes/)
- After writing, run openspec validate to check your work and fix any errors
- Each requirement MUST use SHALL or MUST. Each MUST have WHEN/THEN scenarios.
- Do NOT implement any code. Only create spec artifacts.
- Keep task count proportional to scope. A simple change: 3-8 tasks. Complex: 10-20.
- Be thorough in acceptance criteria — they will be used to verify the implementation.`

	case "execute":
		return `You are an implement agent (neph). Implement a spec-driven change.

Use the openspec CLI to understand your work:
  cd /workspace && openspec instructions tasks --change <change-name>   # see your task checklist
  cd /workspace && openspec show <change-name>                          # see full change details
  cd /workspace && openspec list --json                                 # check task completion status

Steps:
1. Run: ls /workspace/ to find the repo and the openspec change name
2. Run: cd /workspace && openspec show <change-name> to understand the full spec
3. Read the specs at /workspace/openspec/changes/<change-name>/specs/*/spec.md for WHEN/THEN acceptance criteria
4. Read /workspace/openspec/changes/<change-name>/tasks.md for your implementation checklist
5. Implement each task in the repo code
6. After completing each task, mark it [x] in tasks.md (openspec tracks completion via these checkboxes)
7. Run: cd /workspace && openspec list --json to verify all tasks show as complete
8. Run any test/build commands referenced in the specs

The repo is at /workspace/<reponame>/. OpenSpec artifacts are at /workspace/openspec/.
Verification checks openspec list for task completion — tasks not marked [x] will fail verification.`

	case "verify":
		return `You are a manage agent (unc) performing verification.

1. Read specs at /workspace/openspec/changes/*/specs/*/spec.md
2. For each WHEN/THEN scenario, check the implementation
3. Run any test/build commands referenced in scenarios
4. Check /workspace/openspec/changes/*/tasks.md — all tasks must be [x]
5. Run: cd /workspace && openspec validate --json
6. Run: cd /workspace && openspec list --json

Output JSON: {"pass": true/false, "criteria": [{"scenario": "...", "pass": true/false, "explanation": "..."}]}`

	default:
		return ""
	}
}

// stageToRole maps a pipeline stage to the PI_ROLE value ("manage" or "implement").
func stageToRole(stage string) string {
	switch stage {
	case "plan", "verify":
		return "manage"
	case "execute":
		return "implement"
	default:
		return "implement"
	}
}

// maxRepeatedToolCalls is the number of identical consecutive tool calls before
// the agent is killed to prevent infinite loops (e.g., rewriting the same file).
const maxRepeatedToolCalls = 5

// MaxRepeatedToolCalls is the exported value of maxRepeatedToolCalls for testing.
const MaxRepeatedToolCalls = maxRepeatedToolCalls

// startReaders begins scanning stdout/stderr pipes immediately.
// MUST be called while the pipe is still open (before the process exits).
func (proc *AgentProcess) startReaders() {
	proc.readerWg.Add(2)

	streamPipe := func(reader io.ReadCloser, outputType agentv1.OutputType) {
		defer proc.readerWg.Done()
		defer func() { _ = reader.Close() }()
		scanner := bufio.NewScanner(reader)
		// Allow up to 256KB per line for verbose JSON output from pi.
		scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

		// Loop detection state
		var lastToolSig string
		var repeatCount int

		for scanner.Scan() {
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())

			// For stdout: write raw JSON to agent.jsonl, formatted text to agent.log
			if outputType == agentv1.OutputType_OUTPUT_TYPE_STDOUT {
				// Loop detection: kill agent if it repeats the same tool call too many times
				if sig := extractToolCallSignature(string(line)); sig != "" {
					if sig == lastToolSig {
						repeatCount++
						if repeatCount >= maxRepeatedToolCalls {
							log.Printf("Loop detected: tool call %q repeated %d times — killing agent", sig, repeatCount)
							_ = proc.cmd.Process.Kill()
							return
						}
					} else {
						lastToolSig = sig
						repeatCount = 1
					}
				}

				// Always write raw line to JSONL file for machine parsing
				if proc.jsonlFile != nil {
					proc.mu.Lock()
					_, _ = proc.jsonlFile.Write(append(line, '\n'))
					proc.mu.Unlock()
				}

				// Format for human-readable log
				formatted := proc.formatPiEvent(string(line))
				if formatted != "" && proc.logFile != nil {
					proc.mu.Lock()
					_, _ = proc.logFile.Write([]byte(formatted))
					proc.mu.Unlock()
				}

				// Detect tool call lines and record trace spans
				maybeCaptureStdoutSpan(string(line))
			} else if proc.logFile != nil {
				// Stderr: write as-is to log file with timestamp
				ts := time.Now().Format("15:04:05")
				entry := fmt.Sprintf("[%s] STDERR: %s\n", ts, string(line))
				proc.mu.Lock()
				_, _ = proc.logFile.Write([]byte(entry))
				proc.mu.Unlock()
			}

			output := &agentv1.AgentOutput{
				Type:      outputType,
				Data:      line,
				Timestamp: timestamppb.Now(),
			}
			proc.mu.Lock()
			for _, ch := range proc.outputs {
				select {
				case ch <- output:
				default:
					log.Printf("WARNING: dropped %s output (subscriber buffer full)", outputType)
				}
			}
			proc.mu.Unlock()
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Scanner error on %s: %v", outputType, err)
		}
	}

	go streamPipe(proc.stdout, agentv1.OutputType_OUTPUT_TYPE_STDOUT)
	go streamPipe(proc.stderr, agentv1.OutputType_OUTPUT_TYPE_STDERR)
}

// maxRateLimitRetries is the number of times to retry the agent process on rate limit errors.
const maxRateLimitRetries = 3

// rateLimitRetryDelay is the delay before retrying after a rate limit error.
const rateLimitRetryDelay = 10 * time.Second

// isRateLimitError checks if the process stderr output indicates a rate limit error.
func isRateLimitError(stderrOutput string) bool {
	lower := strings.ToLower(stderrOutput)
	return strings.Contains(lower, "429") ||
		strings.Contains(lower, "ratelimiterror") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "rate_limit")
}

func (g *Gateway) waitForProcess(agentRunID string) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return
	}

	err := g.waitForSingleProcess(proc)

	// Check if this is a rate limit error and retry if so.
	// We read the log file to check for rate limit indicators in stderr output.
	if err != nil && isRateLimitError(proc.exitError+err.Error()) {
		for attempt := 1; attempt <= maxRateLimitRetries; attempt++ {
			log.Printf("Agent process hit rate limit (attempt %d/%d), retrying in %v: %s",
				attempt, maxRateLimitRetries, rateLimitRetryDelay, agentRunID)

			time.Sleep(rateLimitRetryDelay)

			// Re-read the original request args from the process to rebuild the command.
			newProc, startErr := restartAgentProcess(proc.cmd)
			if startErr != nil {
				log.Printf("Failed to restart agent process (attempt %d): %v", attempt, startErr)
				continue
			}

			g.mu.Lock()
			g.process = newProc
			g.mu.Unlock()

			newProc.startReaders()
			err = g.waitForSingleProcess(newProc)
			proc = newProc

			if err == nil || !isRateLimitError(newProc.exitError+err.Error()) {
				break
			}
		}

		// If all retries exhausted and still failing, set a clear message.
		if err != nil {
			g.mu.Lock()
			proc.exitError = fmt.Sprintf("Rate limited after %d retries: %s", maxRateLimitRetries, proc.exitError)
			g.mu.Unlock()
		}
	}

	g.mu.Lock()
	if err != nil {
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_FAILED
		if proc.exitError == "" {
			proc.exitError = err.Error()
		}
	} else {
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_COMPLETED
	}
	// Close all output channels
	proc.mu.Lock()
	for _, ch := range proc.outputs {
		close(ch)
	}
	proc.outputs = nil
	proc.mu.Unlock()
	g.mu.Unlock()

	log.Printf("Agent process finished: %s (state=%v)", agentRunID, proc.state)
}

// waitForSingleProcess waits for a single agent process to complete, drains its
// pipes, and closes its log files. Returns the process exit error (nil on success).
func (g *Gateway) waitForSingleProcess(proc *AgentProcess) error {
	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	var err error
	select {
	case err = <-done:
	case <-time.After(24 * time.Hour):
		log.Printf("Agent process timed out after 24h, killing")
		_ = proc.cmd.Process.Kill()
		err = <-done
	}

	// Wait for readers to drain all remaining pipe data
	proc.readerWg.Wait()

	// Close log files after streams are drained
	if proc.logFile != nil {
		_ = proc.logFile.Close()
	}
	if proc.jsonlFile != nil {
		_ = proc.jsonlFile.Close()
	}

	if err != nil {
		proc.exitError = err.Error()
	}

	return err
}

// restartAgentProcess creates a new agent process using the same command arguments
// as the original process.
func restartAgentProcess(origCmd *exec.Cmd) (*AgentProcess, error) {
	cmd := exec.Command(origCmd.Path, origCmd.Args[1:]...)
	cmd.Dir = origCmd.Dir
	cmd.Env = origCmd.Env

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("open /dev/null: %w", err)
	}
	cmd.Stdin = devNull

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := os.MkdirAll(agentLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logFile, err := os.OpenFile(agentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open agent log: %w", err)
	}
	jsonlFile, err := os.OpenFile(agentJSONLPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("open agent jsonl: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = jsonlFile.Close()
		return nil, fmt.Errorf("start agent: %w", err)
	}
	_ = devNull.Close()

	return &AgentProcess{
		cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
		logFile:   logFile,
		jsonlFile: jsonlFile,
		state:     agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING,
		startedAt: time.Now(),
	}, nil
}

func (g *Gateway) StreamOutput(_ context.Context, _ *connect.Request[agentv1.StreamOutputRequest], stream *connect.ServerStream[agentv1.AgentOutput]) error {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no agent process running"))
	}

	ch := make(chan *agentv1.AgentOutput, 100)
	proc.mu.Lock()
	proc.outputs = append(proc.outputs, ch)
	proc.mu.Unlock()

	for output := range ch {
		if err := stream.Send(output); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateway) SendInput(_ context.Context, req *connect.Request[agentv1.SendInputRequest]) (*connect.Response[agentv1.SendInputResponse], error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no agent process running"))
	}

	// Write the human's answer to the response file.
	// The aot-determinism extension polls for this file and resolves the waiting promise.
	if err := os.MkdirAll(filepath.Dir(agentInputResponsePath), 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create input dir: %w", err))
	}
	if err := os.WriteFile(agentInputResponsePath, req.Msg.Data, 0o644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write response file: %w", err))
	}

	// Transition the agent state back to RUNNING.
	proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING
	proc.pendingQuestion = ""
	log.Printf("SendInput: wrote response file, state → RUNNING")

	return connect.NewResponse(&agentv1.SendInputResponse{Accepted: true}), nil
}

func (g *Gateway) GetStatus(_ context.Context, _ *connect.Request[agentv1.GetStatusRequest]) (*connect.Response[agentv1.AgentStatus], error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return connect.NewResponse(&agentv1.AgentStatus{
			State: agentv1.AgentProcessState_AGENT_PROCESS_STATE_UNSPECIFIED,
		}), nil
	}

	s := &agentv1.AgentStatus{
		State:     proc.state,
		StartedAt: timestamppb.New(proc.startedAt),
		Error:     proc.exitError,
	}
	// When waiting for input, surface the pending question in the error field
	// so callers can display it. The proto has no dedicated message field.
	if proc.state == agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT && proc.pendingQuestion != "" {
		s.Error = proc.pendingQuestion
	}
	if proc.cmd.Process != nil {
		s.Pid = int32(proc.cmd.Process.Pid)
	}
	return connect.NewResponse(s), nil
}

func (g *Gateway) NotifyEvent(_ context.Context, req *connect.Request[agentv1.NotifyEventRequest]) (*connect.Response[agentv1.NotifyEventResponse], error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no agent process running"))
	}

	now := time.Now()

	switch req.Msg.EventType {
	case agentv1.EventType_EVENT_TYPE_WAITING_FOR_INPUT:
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT
		proc.pendingQuestion = req.Msg.Payload
		log.Printf("Agent entered WAITING_FOR_INPUT: %s", req.Msg.Payload)

	case agentv1.EventType_EVENT_TYPE_STARTED:
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING
		log.Printf("Agent resumed RUNNING")
		appendTraceSpan(TraceSpan{
			ID:        uuid.New().String(),
			Name:      "agent_resumed",
			Type:      "input",
			StartTime: now,
			EndTime:   now,
		})

	case agentv1.EventType_EVENT_TYPE_TOOL_CALL:
		// 10.1 + 10.2: Record tool call span with git checkpoint
		log.Printf("Agent tool call: %s", req.Msg.Payload)
		metadata := parsePayloadMetadata(req.Msg.Payload)
		spanName := "tool_call"
		if n, ok := metadata["name"].(string); ok && n != "" {
			spanName = n
		}

		span := TraceSpan{
			ID:        uuid.New().String(),
			Name:      spanName,
			Type:      "tool",
			StartTime: now,
			EndTime:   now,
			Metadata:  metadata,
		}

		// Create git checkpoint and capture diff
		sha, diff := createGitCheckpoint(resolvedWorkDir(), spanName)
		if diff != nil && len(diff.Files) > 0 {
			span.HasDiff = true
			span.Diff = diff
		}
		if sha != "" {
			span.Metadata["checkpointSHA"] = sha
		}

		appendTraceSpan(span)

	case agentv1.EventType_EVENT_TYPE_LOG:
		// 10.3: Check for LLM response markers in log events
		payload := req.Msg.Payload
		if isLLMResponseLog(payload) {
			metadata := parsePayloadMetadata(payload)
			span := TraceSpan{
				ID:        uuid.New().String(),
				Name:      "llm_response",
				Type:      "llm",
				StartTime: now,
				EndTime:   now,
				Metadata:  metadata,
			}
			appendTraceSpan(span)
		}
		log.Printf("NotifyEvent LOG: %s", req.Msg.Payload)

	default:
		log.Printf("NotifyEvent: %s payload=%s", req.Msg.EventType, req.Msg.Payload)
	}

	return connect.NewResponse(&agentv1.NotifyEventResponse{Acknowledged: true}), nil
}

func (g *Gateway) StopAgent(_ context.Context, req *connect.Request[agentv1.StopAgentRequest]) (*connect.Response[agentv1.StopAgentResponse], error) {
	g.mu.Lock()
	proc := g.process
	g.mu.Unlock()

	if proc == nil || proc.cmd.Process == nil {
		return connect.NewResponse(&agentv1.StopAgentResponse{Stopped: true}), nil
	}

	if req.Msg.Force {
		_ = proc.cmd.Process.Kill()
	} else {
		_ = proc.cmd.Process.Signal(os.Interrupt)
	}

	return connect.NewResponse(&agentv1.StopAgentResponse{Stopped: true}), nil
}

// ExecCommand runs a bash command in the workspace and returns stdout/stderr/exit code.
// This is a lightweight alternative to StartAgent for running CLI tools like openspec,
// test suites, and file checks.
func (g *Gateway) ExecCommand(ctx context.Context, req *connect.Request[agentv1.ExecCommandRequest]) (*connect.Response[agentv1.ExecCommandResponse], error) {
	// ExecCommand uses the EXACT working dir from the request — no resolveWorkDir.
	// This is intentional: PlanRun/VerifyRun pass specDir=/workspace for openspec commands
	// and workDir for repo commands. resolveWorkDir is only for startAgentProcess (pi).
	workDir := req.Msg.WorkingDir
	if workDir == "" {
		workDir = "/workspace"
	}
	// Fall back to current directory if the specified directory doesn't exist.
	if _, err := os.Stat(workDir); err != nil {
		workDir = "."
	}

	timeout := time.Duration(req.Msg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", req.Msg.Command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return connect.NewResponse(&agentv1.ExecCommandResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: int32(exitCode),
	}), nil
}

// --- Pi JSON event formatting for human-readable logs ---

// piEvent represents the nested structure of pi --mode json output.
type piEvent struct {
	Type              string          `json:"type"`
	Message           json.RawMessage `json:"message,omitempty"`
	AssistantMsgEvent json.RawMessage `json:"assistantMessageEvent,omitempty"`
	ToolResults       json.RawMessage `json:"toolResults,omitempty"`
}

// piAssistantEvent is the inner event within assistantMessageEvent.
type piAssistantEvent struct {
	Type    string          `json:"type"`
	Delta   string          `json:"delta,omitempty"`
	Content string          `json:"content,omitempty"`
	Tool    json.RawMessage `json:"tool,omitempty"`
	// Partial is present in toolcall_start events — contains content blocks with tool info.
	Partial json.RawMessage `json:"partial,omitempty"`
}

// piToolCallBlock is a content block of type "toolCall" within a toolcall_start partial.
type piToolCallBlock struct {
	Type      string                 `json:"type"`
	Name      string                 `json:"name,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// piToolInfo represents tool call information from pi events.
type piToolInfo struct {
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// formatPiEvent takes a raw stdout line from pi and returns a formatted
// human-readable string for the agent.log file. Returns empty string to skip.
func (proc *AgentProcess) formatPiEvent(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}

	// If not JSON, write as-is with timestamp
	if len(trimmed) == 0 || trimmed[0] != '{' {
		ts := time.Now().Format("15:04:05")
		return fmt.Sprintf("[%s] %s\n", ts, trimmed)
	}

	var evt piEvent
	if err := json.Unmarshal([]byte(trimmed), &evt); err != nil {
		// Not valid JSON — write as-is with timestamp
		ts := time.Now().Format("15:04:05")
		return fmt.Sprintf("[%s] %s\n", ts, trimmed)
	}

	ts := time.Now().Format("15:04:05")

	switch evt.Type {
	case "message_start":
		// Reset text accumulator for new message
		proc.textBuf.Reset()
		return ""

	case "message_update":
		if len(evt.AssistantMsgEvent) == 0 {
			return ""
		}
		var ame piAssistantEvent
		if err := json.Unmarshal(evt.AssistantMsgEvent, &ame); err != nil {
			return ""
		}

		switch ame.Type {
		case "text_delta":
			// Accumulate text fragments; don't write yet
			proc.textBuf.WriteString(ame.Delta)
			return ""

		case "text_end":
			// Flush accumulated text (text_end has the full content too)
			text := proc.textBuf.String()
			if text == "" && ame.Content != "" {
				text = ame.Content
			}
			proc.textBuf.Reset()
			if text == "" {
				return ""
			}
			return fmt.Sprintf("[%s] \U0001F916 %s\n", ts, text)

		case "tool_use":
			if len(ame.Tool) == 0 {
				return fmt.Sprintf("[%s] \U0001F527 tool_use\n", ts)
			}
			var tool piToolInfo
			if err := json.Unmarshal(ame.Tool, &tool); err != nil {
				return fmt.Sprintf("[%s] \U0001F527 tool_use\n", ts)
			}
			summary := summarizeToolInput(tool.Input)
			return fmt.Sprintf("[%s] \U0001F527 %s: %s\n", ts, tool.Name, summary)

		default:
			return ""
		}

	case "message_end":
		// Flush any remaining accumulated text
		text := proc.textBuf.String()
		proc.textBuf.Reset()
		if text != "" {
			return fmt.Sprintf("[%s] \U0001F916 %s\n", ts, text)
		}
		return ""

	case "turn_end":
		// Format tool results if present
		if len(evt.ToolResults) == 0 {
			return fmt.Sprintf("[%s] ────────────────────\n", ts)
		}
		var results []map[string]interface{}
		if err := json.Unmarshal(evt.ToolResults, &results); err != nil {
			return fmt.Sprintf("[%s] ────────────────────\n", ts)
		}
		var sb strings.Builder
		for _, r := range results {
			output, _ := r["output"].(string)
			if output == "" {
				continue
			}
			// Indent tool output lines with vertical bar
			for _, ol := range strings.Split(output, "\n") {
				fmt.Fprintf(&sb, "  \u2502 %s\n", ol)
			}
		}
		fmt.Fprintf(&sb, "[%s] \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n", ts)
		return sb.String()

	case "agent_end":
		return fmt.Sprintf("[%s] Agent finished\n", ts)

	default:
		// Skip unrecognized event types
		return ""
	}
}

// summarizeToolInput creates a brief summary of tool input for logging.
func summarizeToolInput(input map[string]interface{}) string {
	if input == nil {
		return ""
	}
	// Common case: bash tool with "command" field
	if cmd, ok := input["command"].(string); ok {
		if len(cmd) > 120 {
			return cmd[:120] + "..."
		}
		return cmd
	}
	// File tools: show file_path
	if fp, ok := input["file_path"].(string); ok {
		return fp
	}
	// Generic: marshal keys
	data, err := json.Marshal(input)
	if err != nil {
		return "<input>"
	}
	s := string(data)
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}

// --- Stdout-based span detection ---

// stdoutToolCallPrefixes are line prefixes that indicate pi-coding-agent is
// invoking a tool. Different versions of pi emit these in slightly different
// formats, so we check several common patterns.
var stdoutToolCallPrefixes = []string{
	"⚡ Running tool:",  // common in pi verbose output
	"> Running tool:",  // alternate format
	"Tool call:",       // generic
	"[tool_call]",      // structured log format
	"Running command:", // shell/bash tool
}

// maybeCaptureStdoutSpan inspects a single stdout line and, if it looks like
// a tool invocation or a pi JSON-mode event, records a trace span.
func maybeCaptureStdoutSpan(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	// Parse as pi streaming event (message_start, tool_execution_start, etc.)
	// This is the primary format pi emits with --mode json.
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var streamEvt piEvent
		if err := json.Unmarshal([]byte(trimmed), &streamEvt); err == nil && streamEvt.Type != "" {
			maybeCaptureStreamEvent(&streamEvt, trimmed)
			return
		}
	}

	// Fall back to plain-text prefix matching (verbose/print mode output).
	for _, prefix := range stdoutToolCallPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			toolInfo := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			now := time.Now()
			appendTraceSpan(TraceSpan{
				ID:        uuid.New().String(),
				Name:      toolInfo,
				Type:      "tool",
				StartTime: now,
				EndTime:   now,
				Metadata:  map[string]interface{}{"source": "stdout", "raw": trimmed},
			})
			return
		}
	}

	// Also detect LLM response lines from stdout
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "[llm]") || strings.HasPrefix(lower, "llm response:") {
		now := time.Now()
		appendTraceSpan(TraceSpan{
			ID:        uuid.New().String(),
			Name:      "llm_response",
			Type:      "llm",
			StartTime: now,
			EndTime:   now,
			Metadata:  map[string]interface{}{"source": "stdout", "raw": trimmed},
		})
	}
}

// --- Span state for streaming events ---
var (
	thinkingSpanMu    sync.Mutex
	thinkingSpanID    string
	thinkingSpanStart time.Time
	inThinkingBlock   bool

	// Current agent context — set by StartAgent, used by span creators
	currentStageMu sync.RWMutex
	currentStage   string // "plan", "execute", "verify", ""
	currentWorkDir string // resolved workspace path for git diffs

	// Git checkpoint state — tracks the last checkpoint commit SHA
	checkpointMu      sync.Mutex
	lastCheckpointSHA string

	// Parent span linking — set by StartAgent from workflow-provided IDs
	parentSpanMu        sync.RWMutex
	currentParentSpanID string
	currentTraceID      string

	// Model tracking — set by StartAgent from PI_MODEL env var
	currentModel string

	// Text accumulator for thought spans — collects text_delta events
	thoughtTextMu      sync.Mutex
	thoughtTextBuilder strings.Builder
)

// modelContextWindows maps model names to their context window sizes (in tokens).
var modelContextWindows = map[string]int{
	"deepseek-v3.1": 32768,
	"deepseek-v3.2": 163840,
	"qwen3-coder":   262144,
	"qwen3:8b":      32768,
	"default":       32768,
}

func setCurrentParentSpan(parentID, traceID string) {
	parentSpanMu.Lock()
	defer parentSpanMu.Unlock()
	currentParentSpanID = parentID
	currentTraceID = traceID
}

func getParentSpanID() string {
	parentSpanMu.RLock()
	defer parentSpanMu.RUnlock()
	return currentParentSpanID
}

func getTraceID() string {
	parentSpanMu.RLock()
	defer parentSpanMu.RUnlock()
	return currentTraceID
}

// setCurrentStage updates the package-level stage and workdir for span naming.
func setCurrentStage(stage, workDir string) {
	currentStageMu.Lock()
	defer currentStageMu.Unlock()
	currentStage = stage
	currentWorkDir = workDir
}

// spanPrefix returns "manage" for plan/verify stages, "implement" for execute/single.
func spanPrefix() string {
	currentStageMu.RLock()
	defer currentStageMu.RUnlock()
	if currentStage == "plan" || currentStage == "verify" {
		return "manage"
	}
	return "implement"
}

// resolvedWorkDir returns the current resolved workspace for git diffs.
func resolvedWorkDir() string {
	currentStageMu.RLock()
	defer currentStageMu.RUnlock()
	if currentWorkDir != "" {
		return currentWorkDir
	}
	return "/workspace"
}

// Active span tracking for open-close matching
var (
	activeToolSpanMu       sync.Mutex
	activeToolSpanID       string
	activeToolSpanStart    time.Time
	activeToolSpanName     string
	activeToolInputSummary string

	activeLLMSpanMu    sync.Mutex
	activeLLMSpanID    string
	activeLLMSpanStart time.Time
)

// maybeCaptureStreamEvent processes pi streaming events and creates trace spans
// for tool executions, LLM turns, and thinking blocks.
func maybeCaptureStreamEvent(evt *piEvent, raw string) {
	now := time.Now()

	switch evt.Type {
	case "tool_execution_start":
		activeToolSpanMu.Lock()
		// Tool name was already set by the preceding message_update tool_use event.
		// Only default to "tool" if it wasn't set.
		if activeToolSpanName == "" {
			activeToolSpanName = "tool"
		}
		activeToolSpanID = uuid.New().String()
		activeToolSpanStart = now
		activeToolSpanMu.Unlock()

	case "tool_execution_end":
		activeToolSpanMu.Lock()
		if activeToolSpanID != "" {
			prefix := spanPrefix()
			span := TraceSpan{
				ID:        activeToolSpanID,
				TraceID:   getTraceID(),
				ParentID:  getParentSpanID(),
				Name:      prefix + "." + activeToolSpanName,
				Type:      "tool",
				StartTime: activeToolSpanStart,
				EndTime:   now,
				Metadata: map[string]interface{}{
					"tool":      activeToolSpanName,
					"toolInput": activeToolInputSummary,
					"stage":     currentStage,
					"role":      prefix,
				},
			}
			// Create git checkpoint and capture diff
			sha, diff := createGitCheckpoint(resolvedWorkDir(), activeToolSpanName)
			if diff != nil && len(diff.Files) > 0 {
				span.HasDiff = true
				span.Diff = diff
			}
			if sha != "" {
				span.Metadata["checkpointSHA"] = sha
				checkpointMu.Lock()
				if lastCheckpointSHA != sha {
					span.Metadata["prevCheckpointSHA"] = lastCheckpointSHA
				}
				checkpointMu.Unlock()
			}
			appendTraceSpan(span)
			activeToolSpanID = ""
			activeToolInputSummary = ""
		}
		activeToolSpanMu.Unlock()

	case "message_start":
		// Check if this is an assistant message (LLM turn)
		if len(evt.Message) > 0 {
			var msg struct {
				Role string `json:"role"`
			}
			if json.Unmarshal(evt.Message, &msg) == nil && msg.Role == "assistant" {
				activeLLMSpanMu.Lock()
				activeLLMSpanID = uuid.New().String()
				activeLLMSpanStart = now
				activeLLMSpanMu.Unlock()
			}
		}

	case "message_update":
		if len(evt.AssistantMsgEvent) == 0 {
			return
		}
		var ame piAssistantEvent
		if err := json.Unmarshal(evt.AssistantMsgEvent, &ame); err != nil {
			return
		}

		switch ame.Type {
		case "thinking_delta":
			thinkingSpanMu.Lock()
			if !inThinkingBlock {
				inThinkingBlock = true
				thinkingSpanID = uuid.New().String()
				thinkingSpanStart = now
			}
			thinkingSpanMu.Unlock()

		case "text_delta":
			closeThinkingSpan()
			// Accumulate text for the thought span content
			if ame.Delta != "" {
				thoughtTextMu.Lock()
				thoughtTextBuilder.WriteString(ame.Delta)
				thoughtTextMu.Unlock()
			}

		case "text_end":
			closeThinkingSpan()

		case "tool_use", "toolcall_start", "toolcall_end":
			closeThinkingSpan()
			// Extract tool name + input from the event.
			toolName, toolInputJSON := extractToolFromEvent(&ame)
			if toolName != "" {
				activeToolSpanMu.Lock()
				activeToolSpanName = toolName
				if len(toolInputJSON) > 200 {
					activeToolInputSummary = toolInputJSON[:200] + "..."
				} else {
					activeToolInputSummary = toolInputJSON
				}
				activeToolSpanMu.Unlock()
			}
		}

	case "message_end":
		closeThinkingSpan()
		// Close LLM span
		activeLLMSpanMu.Lock()
		if activeLLMSpanID != "" {
			prefix := spanPrefix()
			durationMs := now.Sub(activeLLMSpanStart).Milliseconds()
			span := TraceSpan{
				ID:        activeLLMSpanID,
				TraceID:   getTraceID(),
				ParentID:  getParentSpanID(),
				Name:      prefix + ".thought",
				Type:      "llm",
				StartTime: activeLLMSpanStart,
				EndTime:   now,
				Metadata: map[string]interface{}{
					"durationMs":           durationMs,
					"stage":                currentStage,
					"role":                 prefix,
					"gen_ai.request.model": currentModel,
				},
			}

			// Attach accumulated thought text (truncated to 2000 chars for storage)
			thoughtTextMu.Lock()
			thoughtText := thoughtTextBuilder.String()
			thoughtTextBuilder.Reset()
			thoughtTextMu.Unlock()
			if len(thoughtText) > 2000 {
				span.Metadata["content"] = thoughtText[:2000] + "..."
			} else if thoughtText != "" {
				span.Metadata["content"] = thoughtText
			}

			// Extract token usage from message_end event.
			// Pi sends usage as: {"input": N, "output": N, "cacheRead": N, "totalTokens": N}
			if len(evt.Message) > 0 {
				var msg struct {
					Usage *struct {
						Input       int `json:"input"`
						Output      int `json:"output"`
						CacheRead   int `json:"cacheRead"`
						CacheWrite  int `json:"cacheWrite"`
						TotalTokens int `json:"totalTokens"`
					} `json:"usage"`
				}
				if json.Unmarshal(evt.Message, &msg) == nil && msg.Usage != nil {
					span.Metadata["gen_ai.usage.input_tokens"] = msg.Usage.Input
					span.Metadata["gen_ai.usage.output_tokens"] = msg.Usage.Output
					span.Metadata["gen_ai.usage.total_tokens"] = msg.Usage.TotalTokens
					if msg.Usage.CacheRead > 0 {
						span.Metadata["gen_ai.usage.cache_read_tokens"] = msg.Usage.CacheRead
					}
					// Context utilization
					if msg.Usage.Input > 0 {
						windowSize := modelContextWindows[currentModel]
						if windowSize == 0 {
							windowSize = 32768
						}
						span.Metadata["gen_ai.context.window_size"] = windowSize
						span.Metadata["gen_ai.context.utilization_pct"] = int(float64(msg.Usage.Input) / float64(windowSize) * 100)
					}
				}
			}

			appendTraceSpan(span)
			activeLLMSpanID = ""
		}
		activeLLMSpanMu.Unlock()
	}
}

// extractToolFromEvent extracts the tool name and input JSON from an assistantMessageEvent.
// Handles both pi's "toolcall_start" format (partial.content[].type=="toolCall")
// and legacy "tool_use" format (ame.Tool).
func extractToolFromEvent(ame *piAssistantEvent) (name string, inputJSON string) {
	// Try pi's toolcall_start format: partial.content[] has toolCall blocks
	if len(ame.Partial) > 0 {
		var partial struct {
			Content []piToolCallBlock `json:"content"`
		}
		if json.Unmarshal(ame.Partial, &partial) == nil {
			for _, block := range partial.Content {
				if block.Type == "toolCall" && block.Name != "" {
					name = block.Name
					if block.Arguments != nil {
						b, _ := json.Marshal(block.Arguments)
						inputJSON = string(b)
					}
					return
				}
			}
		}
	}

	// Try legacy format: ame.Tool
	if len(ame.Tool) > 0 {
		var tool piToolInfo
		if json.Unmarshal(ame.Tool, &tool) == nil && tool.Name != "" {
			name = tool.Name
			if tool.Input != nil {
				b, _ := json.Marshal(tool.Input)
				inputJSON = string(b)
			}
			return
		}
	}

	return "", ""
}

// closeThinkingSpan closes an active thinking span if one is open.
func closeThinkingSpan() {
	thinkingSpanMu.Lock()
	defer thinkingSpanMu.Unlock()
	if !inThinkingBlock {
		return
	}
	inThinkingBlock = false
	appendTraceSpan(TraceSpan{
		ID:        thinkingSpanID,
		TraceID:   getTraceID(),
		ParentID:  getParentSpanID(),
		Name:      "thinking",
		Type:      "thought",
		StartTime: thinkingSpanStart,
		EndTime:   time.Now(),
	})
	thinkingSpanID = ""
}

// --- Trace helpers (Section 10) ---

// parsePayloadMetadata attempts to parse a JSON payload string into a metadata map.
// If the payload is not valid JSON, it returns a map with the raw payload as "raw".
func parsePayloadMetadata(payload string) map[string]interface{} {
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &metadata); err != nil {
		return map[string]interface{}{"raw": payload}
	}
	return metadata
}

// isLLMResponseLog checks if a log event payload contains LLM response markers.
func isLLMResponseLog(payload string) bool {
	markers := []string{"llm_response", "model", "completion", "tokens", "usage"}
	lower := strings.ToLower(payload)
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// createGitCheckpoint stages all changes, commits them as a checkpoint, and returns
// the commit SHA + diff against the previous checkpoint. Returns ("", nil) if no changes.
func createGitCheckpoint(workDir, toolName string) (string, *SpanDiff) {
	// 1. git add -A (stage everything including untracked)
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = workDir
	if err := addCmd.Run(); err != nil {
		log.Printf("git checkpoint: add failed in %s: %v", workDir, err)
		return "", nil
	}

	// 2. Check if anything to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = workDir
	var statusOut bytes.Buffer
	statusCmd.Stdout = &statusOut
	_ = statusCmd.Run()
	if strings.TrimSpace(statusOut.String()) == "" {
		return "", nil // No changes
	}

	// 3. Commit with checkpoint message
	msg := fmt.Sprintf("aot-checkpoint: %s", toolName)
	commitCmd := exec.Command("git", "commit", "--no-verify", "--allow-empty", "-m", msg)
	commitCmd.Dir = workDir
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aot-agent",
		"GIT_AUTHOR_EMAIL=agent@aot.uncworks.io",
		"GIT_COMMITTER_NAME=aot-agent",
		"GIT_COMMITTER_EMAIL=agent@aot.uncworks.io",
	)
	if err := commitCmd.Run(); err != nil {
		log.Printf("git checkpoint: commit failed in %s: %v", workDir, err)
		return "", nil
	}

	// 4. Get current SHA
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaCmd.Dir = workDir
	var shaOut bytes.Buffer
	shaCmd.Stdout = &shaOut
	if err := shaCmd.Run(); err != nil {
		return "", nil
	}
	currentSHA := strings.TrimSpace(shaOut.String())

	// 5. Compute diff against previous checkpoint
	checkpointMu.Lock()
	prevSHA := lastCheckpointSHA
	lastCheckpointSHA = currentSHA
	checkpointMu.Unlock()

	var diffCmd *exec.Cmd
	if prevSHA != "" {
		diffCmd = exec.Command("git", "diff", prevSHA+".."+currentSHA)
	} else {
		diffCmd = exec.Command("git", "diff", "HEAD~1..HEAD")
	}
	diffCmd.Dir = workDir
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	if err := diffCmd.Run(); err != nil {
		// diff command failed (e.g., no HEAD~1 for first commit)
		return currentSHA, nil
	}

	diffText := diffOut.String()
	if strings.TrimSpace(diffText) == "" {
		return currentSHA, nil
	}

	return currentSHA, parseDiffOutput(diffText)
}

// parseDiffOutput splits unified diff output into per-file FileDiff entries.
func parseDiffOutput(output string) *SpanDiff {
	var files []FileDiff
	sections := strings.Split(output, "diff --git ")

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// Extract file path from the "a/path b/path" header
		lines := strings.SplitN(section, "\n", 2)
		header := lines[0]
		parts := strings.Fields(header)
		filePath := ""
		if len(parts) >= 2 {
			filePath = strings.TrimPrefix(parts[1], "b/")
		}

		files = append(files, FileDiff{
			Path:  filePath,
			Patch: "diff --git " + section,
		})
	}

	if len(files) == 0 {
		return nil
	}
	return &SpanDiff{Files: files}
}

// appendTraceSpan appends a TraceSpan as a JSON line to the trace spans JSONL file.
func appendTraceSpan(span TraceSpan) {
	data, err := json.Marshal(span)
	if err != nil {
		log.Printf("WARNING: failed to marshal trace span: %v", err)
		return
	}

	f, err := os.OpenFile(traceSpansPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("WARNING: failed to open trace spans file: %v", err)
		return
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("WARNING: failed to write trace span: %v", err)
	}
}

// resolveWorkDir determines the actual working directory for agent processes.
// Repos are cloned as worktrees into /workspace/<repoName>/.
// Single-repo runs have one subdir, multi-repo runs have multiple.
func resolveWorkDir(repoPath string) string {
	if repoPath == "" {
		repoPath = "/workspace"
	}
	return ResolveWorkDirAt(repoPath, "/workspace")
}

// ResolveWorkDirAt resolves the working directory for the agent, relative to
// the given defaultBase. If repoPath differs from defaultBase, it is returned
// unchanged. Otherwise it scans for a git repo at defaultBase or in its
// immediate subdirectories (skipping .bare, .aot, .devcontainer, openspec,
// spec).
func ResolveWorkDirAt(repoPath, defaultBase string) string {
	if repoPath != defaultBase {
		return repoPath
	}
	// Check if this is already a repo (single-repo clone into root)
	if _, err := os.Stat(filepath.Join(defaultBase, ".git")); err == nil {
		return defaultBase
	}
	// Check for repo subdirs in defaultBase/ (worktrees at defaultBase/<repoName>/)
	entries, err := os.ReadDir(defaultBase)
	if err != nil {
		return repoPath
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != ".bare" && e.Name() != ".aot" && e.Name() != ".devcontainer" && e.Name() != "openspec" && e.Name() != "spec" {
			gitPath := filepath.Join(defaultBase, e.Name(), ".git")
			if _, err := os.Stat(gitPath); err == nil {
				resolved := filepath.Join(defaultBase, e.Name())
				log.Printf("resolveWorkDir: %s → %s", defaultBase, resolved)
				return resolved
			}
		}
	}
	return defaultBase
}

// extractToolCallSignature returns a short signature for a tool call JSONL event,
// used for loop detection. Returns "" for non-tool-call events.
func extractToolCallSignature(line string) string {
	var event struct {
		Type    string `json:"type"`
		Message *struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal([]byte(line), &event) != nil || event.Type != "message_end" {
		return ""
	}
	if event.Message == nil {
		return ""
	}

	var blocks []struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if json.Unmarshal(event.Message.Content, &blocks) != nil {
		return ""
	}
	for _, b := range blocks {
		if b.Type == "tool_use" {
			return fmt.Sprintf("%s:%d", b.Name, len(b.Input))
		}
	}
	return ""
}

// ExtractToolCallSignature is the exported wrapper of extractToolCallSignature
// for integration testing.
func ExtractToolCallSignature(line string) string {
	return extractToolCallSignature(line)
}
