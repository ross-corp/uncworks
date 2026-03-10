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
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	state     agentv1.AgentProcessState
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
	cmd := exec.Command("devbox", "run", "--", "agent", "--prompt", req.Prompt)
	cmd.Dir = req.RepoPath

	for k, v := range req.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
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

	return &AgentProcess{
		cmd:       cmd,
		stdin:     stdin,
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
	go func() {
		scanner := bufio.NewScanner(proc.stdout)
		for scanner.Scan() {
			line := make([]byte, len(scanner.Bytes()))
			copy(line, scanner.Bytes())
			output := &agentv1.AgentOutput{
				Type:      agentv1.OutputType_OUTPUT_TYPE_STDOUT,
				Data:      line,
				Timestamp: timestamppb.Now(),
			}
			proc.mu.Lock()
			for _, ch := range proc.outputs {
				select {
				case ch <- output:
				default:
				}
			}
			proc.mu.Unlock()
		}
	}()

	// Wait for process to finish
	err := proc.cmd.Wait()

	g.mu.Lock()
	if err != nil {
		proc.state = agentv1.AgentProcessState_AGENT_PROCESS_STATE_FAILED
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

	if proc.state != agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING &&
		proc.state != agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent not accepting input"))
	}

	data := make([]byte, len(req.Msg.Data)+1)
	copy(data, req.Msg.Data)
	data[len(req.Msg.Data)] = '\n'
	if _, err := proc.stdin.Write(data); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write to stdin: %v", err))
	}

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
	}
	if proc.cmd.Process != nil {
		s.Pid = int32(proc.cmd.Process.Pid)
	}
	return connect.NewResponse(s), nil
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
