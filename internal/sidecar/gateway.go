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

// agentLogPath is the full path to the agent log file on the PVC.
const agentLogPath = agentLogDir + "/agent.log"

// traceDir is the directory for trace span files on the PVC.
const traceDir = "/workspace/.aot/traces"

// traceSpansPath is the JSONL file for trace spans.
const traceSpansPath = traceDir + "/spans.jsonl"

// TraceSpan represents a single trace span recorded during an agent run.
type TraceSpan struct {
	ID        string                 `json:"id"`
	ParentID  string                 `json:"parentId,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
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
	state     agentv1.AgentProcessState
	exitError string
	startedAt time.Time
	outputs   []chan *agentv1.AgentOutput
	mu        sync.Mutex
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

	if g.process != nil && g.process.state == agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING {
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: false, Error: "agent already running"}), nil
	}

	proc, err := startAgentProcess(req.Msg)
	if err != nil {
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: false, Error: err.Error()}), nil
	}

	g.process = proc

	// Write an initial trace span so the traces tab has data immediately.
	appendTraceSpan(TraceSpan{
		ID:        uuid.New().String(),
		Name:      "agent_started",
		Type:      "lifecycle",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Metadata:  map[string]interface{}{"prompt": req.Msg.Prompt, "agentRunId": req.Msg.AgentRunId},
	})

	// Monitor process in background
	go g.monitorProcess(req.Msg.AgentRunId)

	return connect.NewResponse(&agentv1.StartAgentResponse{Started: true}), nil
}

func startAgentProcess(req *agentv1.StartAgentRequest) (*AgentProcess, error) {
	args := []string{"-p", req.Prompt, "--verbose"}
	// Use model from env if configured
	if model := os.Getenv("PI_MODEL"); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.Command("pi", args...)
	cmd.Dir = req.RepoPath
	if cmd.Dir == "" {
		cmd.Dir = "/workspace"
	}

	// Inherit current environment and add request-specific vars on top.
	// Set PI_LOG_LEVEL=debug so pi-coding-agent emits tool call and
	// LLM response details to stdout where the sidecar can capture them.
	cmd.Env = append(os.Environ(), "PI_LOG_LEVEL=debug")
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

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start agent: %w", err)
	}
	devNull.Close()

	return &AgentProcess{
		cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
		logFile:   logFile,
		state:     agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING,
		startedAt: time.Now(),
	}, nil
}

func (g *Gateway) monitorProcess(agentRunID string) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return
	}

	// Stream stdout to watchers
	var wg sync.WaitGroup
	wg.Add(2)

	streamPipe := func(reader io.ReadCloser, outputType agentv1.OutputType) {
		defer wg.Done()
		defer func() { _ = reader.Close() }()
		scanner := bufio.NewScanner(reader)
		// Allow up to 256KB per line for verbose tool output.
		scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
		for scanner.Scan() {
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())

			// Tee output to PVC log file (5.1)
			if proc.logFile != nil {
				proc.mu.Lock()
				_, _ = proc.logFile.Write(append(line, '\n'))
				proc.mu.Unlock()
			}

			// Heuristic: detect tool call lines from pi-coding-agent stdout
			// and record trace spans even if the extension doesn't notify us.
			if outputType == agentv1.OutputType_OUTPUT_TYPE_STDOUT {
				maybeCaptureStdoutSpan(string(line))
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
	}

	go streamPipe(proc.stdout, agentv1.OutputType_OUTPUT_TYPE_STDOUT)
	go streamPipe(proc.stderr, agentv1.OutputType_OUTPUT_TYPE_STDERR)

	// Wait for readers to drain before waiting on process
	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	// Wait for process with timeout
	var err error
	select {
	case err = <-done:
	case <-time.After(24 * time.Hour):
		log.Printf("Agent process timed out after 24h, killing: %s", agentRunID)
		_ = proc.cmd.Process.Kill()
		err = <-done
	}

	wg.Wait()

	// Close the log file after streams are drained
	if proc.logFile != nil {
		_ = proc.logFile.Close()
	}

	g.mu.Lock()
	if err != nil {
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_FAILED
		proc.exitError = err.Error()
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

func (g *Gateway) SendInput(_ context.Context, _ *connect.Request[agentv1.SendInputRequest]) (*connect.Response[agentv1.SendInputResponse], error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no agent process running"))
	}

	// pi-coding-agent in -p (print) mode doesn't accept stdin after startup.
	// HITL will need a different mechanism (e.g., RPC mode or session continuation).
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("stdin-based input not supported in print mode"))
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
		log.Printf("Agent entered WAITING_FOR_INPUT: %s", req.Msg.Payload)

	case agentv1.EventType_EVENT_TYPE_STARTED:
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING
		log.Printf("Agent resumed RUNNING")
		appendTraceSpan(TraceSpan{
			ID:        uuid.New().String(),
			Name:      "agent_resumed",
			Type:      "lifecycle",
			StartTime: now,
			EndTime:   now,
		})

	case agentv1.EventType_EVENT_TYPE_TOOL_CALL:
		// 10.1 + 10.2: Record tool call span with git diff
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

		// 10.6: Capture git diff HEAD in the workspace
		if diff := captureGitDiff("/workspace"); diff != nil && len(diff.Files) > 0 {
			span.HasDiff = true
			span.Diff = diff
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
// a tool invocation, records a trace span. This is a fallback for when the
// extension doesn't send NotifyEvent TOOL_CALL events.
func maybeCaptureStdoutSpan(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

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

// captureGitDiff runs `git diff HEAD` in the given directory and parses the output
// into a SpanDiff with per-file patches.
func captureGitDiff(workDir string) *SpanDiff {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("WARNING: git diff failed in %s: %v (stderr: %s)", workDir, err, stderr.String())
		return nil
	}

	output := stdout.String()
	if strings.TrimSpace(output) == "" {
		return nil
	}

	return parseDiffOutput(output)
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
