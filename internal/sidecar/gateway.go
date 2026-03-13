// Package sidecar implements the RPC Gateway sidecar that bridges ConnectRPC and the agent harness process.
package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"connectrpc.com/connect"
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
	port int

	mu      sync.RWMutex
	process *AgentProcess
	server  *http.Server
}

// AgentProcess wraps the agent harness process.
type AgentProcess struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	state     agentv1.AgentProcessState
	exitError string
	startedAt time.Time
	outputs   []chan *agentv1.AgentOutput
	mu        sync.Mutex
}

// NewGateway creates a new RPC Gateway sidecar.
func NewGateway(port int) *Gateway {
	return &Gateway{port: port}
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

	if g.process != nil && g.process.state == agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING {
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: false, Error: "agent already running"}), nil
	}

	proc, err := startAgentProcess(req.Msg)
	if err != nil {
		return connect.NewResponse(&agentv1.StartAgentResponse{Started: false, Error: err.Error()}), nil
	}

	g.process = proc

	// Monitor process in background
	go g.monitorProcess(req.Msg.AgentRunId)

	return connect.NewResponse(&agentv1.StartAgentResponse{Started: true}), nil
}

func startAgentProcess(req *agentv1.StartAgentRequest) (*AgentProcess, error) {
	args := []string{"-p", req.Prompt}
	// Use model from env if configured
	if model := os.Getenv("PI_MODEL"); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.Command("pi", args...)
	cmd.Dir = req.RepoPath
	if cmd.Dir == "" {
		cmd.Dir = "/workspace"
	}

	// Inherit current environment and add request-specific vars on top
	cmd.Env = os.Environ()
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

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}
	devNull.Close()

	return &AgentProcess{
		cmd:       cmd,
		stdout:    stdout,
		stderr:    stderr,
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
		for scanner.Scan() {
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())
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

func (g *Gateway) StreamOutput(_ context.Context, req *connect.Request[agentv1.StreamOutputRequest], stream *connect.ServerStream[agentv1.AgentOutput]) error {
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

	switch req.Msg.EventType {
	case agentv1.EventType_EVENT_TYPE_WAITING_FOR_INPUT:
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT
		log.Printf("Agent entered WAITING_FOR_INPUT: %s", req.Msg.Payload)
	case agentv1.EventType_EVENT_TYPE_STARTED:
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING
		log.Printf("Agent resumed RUNNING")
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
