// Package sidecar implements the RPC Gateway sidecar that bridges gRPC and the agent harness process.
package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
)

// Gateway is the RPC Gateway sidecar server.
type Gateway struct {
	agentv1.UnimplementedAgentSidecarServiceServer
	port int

	mu         sync.RWMutex
	process    *AgentProcess
	grpcServer *grpc.Server
	notifier   agentv1.AgentNotificationServiceClient
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

// Start begins listening for gRPC connections from the Control Plane.
func (g *Gateway) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", g.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", g.port, err)
	}

	g.grpcServer = grpc.NewServer()
	agentv1.RegisterAgentSidecarServiceServer(g.grpcServer, g)

	log.Printf("RPC Gateway listening on :%d", g.port)
	return g.grpcServer.Serve(lis)
}

// Stop gracefully stops the gateway.
func (g *Gateway) Stop() {
	if g.grpcServer != nil {
		g.grpcServer.GracefulStop()
	}
}

func (g *Gateway) StartAgent(_ context.Context, req *agentv1.StartAgentRequest) (*agentv1.StartAgentResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.process != nil && g.process.state == agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING {
		return &agentv1.StartAgentResponse{Started: false, Error: "agent already running"}, nil
	}

	proc, err := startAgentProcess(req)
	if err != nil {
		return &agentv1.StartAgentResponse{Started: false, Error: err.Error()}, nil
	}

	g.process = proc

	// Monitor process in background
	go g.monitorProcess(req.AgentRunId)

	return &agentv1.StartAgentResponse{Started: true}, nil
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
			output := &agentv1.AgentOutput{
				Type:      agentv1.OutputType_OUTPUT_TYPE_STDOUT,
				Data:      scanner.Bytes(),
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

func (g *Gateway) StreamOutput(_ *agentv1.StreamOutputRequest, stream grpc.ServerStreamingServer[agentv1.AgentOutput]) error {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return status.Error(codes.FailedPrecondition, "no agent process running")
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

func (g *Gateway) SendInput(_ context.Context, req *agentv1.SendInputRequest) (*agentv1.SendInputResponse, error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return nil, status.Error(codes.FailedPrecondition, "no agent process running")
	}

	if proc.state != agentv1.AgentProcessState_AGENT_PROCESS_STATE_RUNNING &&
		proc.state != agentv1.AgentProcessState_AGENT_PROCESS_STATE_WAITING_FOR_INPUT {
		return nil, status.Error(codes.FailedPrecondition, "agent not accepting input")
	}

	if _, err := proc.stdin.Write(append(req.Data, '\n')); err != nil {
		return nil, status.Errorf(codes.Internal, "write to stdin: %v", err)
	}

	return &agentv1.SendInputResponse{Accepted: true}, nil
}

func (g *Gateway) GetStatus(_ context.Context, _ *agentv1.GetStatusRequest) (*agentv1.AgentStatus, error) {
	g.mu.RLock()
	proc := g.process
	g.mu.RUnlock()

	if proc == nil {
		return &agentv1.AgentStatus{
			State: agentv1.AgentProcessState_AGENT_PROCESS_STATE_UNSPECIFIED,
		}, nil
	}

	s := &agentv1.AgentStatus{
		State:     proc.state,
		StartedAt: timestamppb.New(proc.startedAt),
	}
	if proc.cmd.Process != nil {
		s.Pid = int32(proc.cmd.Process.Pid)
	}
	return s, nil
}

func (g *Gateway) StopAgent(_ context.Context, req *agentv1.StopAgentRequest) (*agentv1.StopAgentResponse, error) {
	g.mu.Lock()
	proc := g.process
	g.mu.Unlock()

	if proc == nil || proc.cmd.Process == nil {
		return &agentv1.StopAgentResponse{Stopped: true}, nil
	}

	if req.Force {
		proc.cmd.Process.Kill()
	} else {
		proc.cmd.Process.Signal(os.Interrupt)
	}

	return &agentv1.StopAgentResponse{Stopped: true}, nil
}
