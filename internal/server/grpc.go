// Package server implements the AOT gRPC and WebSocket API server.
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// GRPCServer wraps the gRPC server for the AOT API.
type GRPCServer struct {
	apiv1.UnimplementedAOTServiceServer
	server *grpc.Server
	port   int

	mu       sync.RWMutex
	runs     map[string]*apiv1.AgentRun
	watchers map[string][]chan *apiv1.AgentRunEvent
}

// NewGRPCServer creates a new gRPC server on the given port.
func NewGRPCServer(port int) *GRPCServer {
	s := &GRPCServer{
		port:     port,
		runs:     make(map[string]*apiv1.AgentRun),
		watchers: make(map[string][]chan *apiv1.AgentRunEvent),
	}
	s.server = grpc.NewServer()
	apiv1.RegisterAOTServiceServer(s.server, s)
	reflection.Register(s.server)
	return s
}

// Start begins listening for gRPC connections.
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}
	log.Printf("gRPC server listening on :%d", s.port)
	return s.server.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *GRPCServer) CreateAgentRun(_ context.Context, req *apiv1.CreateAgentRunRequest) (*apiv1.CreateAgentRunResponse, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	id := fmt.Sprintf("ar-%d", len(s.runs)+1)
	now := timestamppb.Now()

	run := &apiv1.AgentRun{
		Id:   id,
		Name: id,
		Spec: req.Spec,
		Status: &apiv1.AgentRunStatus{
			Phase:   apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
			Message: "Queued",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.runs[id] = run
	s.mu.Unlock()

	return &apiv1.CreateAgentRunResponse{AgentRun: run}, nil
}

func (s *GRPCServer) GetAgentRun(_ context.Context, req *apiv1.GetAgentRunRequest) (*apiv1.AgentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, ok := s.runs[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent run %q not found", req.Id)
	}
	return run, nil
}

func (s *GRPCServer) ListAgentRuns(_ context.Context, req *apiv1.ListAgentRunsRequest) (*apiv1.ListAgentRunsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*apiv1.AgentRun
	for _, run := range s.runs {
		if req.PhaseFilter != apiv1.AgentRunPhase_AGENT_RUN_PHASE_UNSPECIFIED &&
			run.Status.Phase != req.PhaseFilter {
			continue
		}
		runs = append(runs, run)
	}

	limit := int(req.Limit)
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}

	return &apiv1.ListAgentRunsResponse{AgentRuns: runs}, nil
}

func (s *GRPCServer) WatchAgentRun(req *apiv1.WatchAgentRunRequest, stream grpc.ServerStreamingServer[apiv1.AgentRunEvent]) error {
	s.mu.RLock()
	_, ok := s.runs[req.Id]
	s.mu.RUnlock()
	if !ok {
		return status.Errorf(codes.NotFound, "agent run %q not found", req.Id)
	}

	ch := make(chan *apiv1.AgentRunEvent, 100)
	s.mu.Lock()
	s.watchers[req.Id] = append(s.watchers[req.Id], ch)
	s.mu.Unlock()

	for event := range ch {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *GRPCServer) CancelAgentRun(_ context.Context, req *apiv1.CancelAgentRunRequest) (*apiv1.CancelAgentRunResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[req.Id]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent run %q not found", req.Id)
	}

	run.Status.Phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	run.Status.Message = "Cancelled by user"
	run.UpdatedAt = timestamppb.Now()

	return &apiv1.CancelAgentRunResponse{AgentRun: run}, nil
}

func (s *GRPCServer) SendHumanInput(_ context.Context, req *apiv1.SendHumanInputRequest) (*apiv1.SendHumanInputResponse, error) {
	s.mu.RLock()
	run, ok := s.runs[req.AgentRunId]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent run %q not found", req.AgentRunId)
	}

	if run.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
		return nil, status.Error(codes.FailedPrecondition, "agent is not waiting for input")
	}

	return &apiv1.SendHumanInputResponse{Accepted: true}, nil
}

// EmitEvent sends an event to all watchers of the given agent run.
func (s *GRPCServer) EmitEvent(agentRunID string, event *apiv1.AgentRunEvent) {
	s.mu.RLock()
	watchers := s.watchers[agentRunID]
	s.mu.RUnlock()

	for _, ch := range watchers {
		select {
		case ch <- event:
		default:
			// Drop if channel full
		}
	}
}
