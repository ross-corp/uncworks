// Package server implements the AOT ConnectRPC API server.
package server

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
)

// AOTServiceHandler implements the AOTService ConnectRPC handler.
type AOTServiceHandler struct {
	apiv1connect.UnimplementedAOTServiceHandler

	mu       sync.RWMutex
	runs     map[string]*apiv1.AgentRun
	watchers map[string][]chan *apiv1.AgentRunEvent
}

// NewAOTServiceHandler creates a new AOTService handler.
func NewAOTServiceHandler() *AOTServiceHandler {
	return &AOTServiceHandler{
		runs:     make(map[string]*apiv1.AgentRun),
		watchers: make(map[string][]chan *apiv1.AgentRunEvent),
	}
}

func (s *AOTServiceHandler) CreateAgentRun(_ context.Context, req *connect.Request[apiv1.CreateAgentRunRequest]) (*connect.Response[apiv1.CreateAgentRunResponse], error) {
	if req.Msg.Spec == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("spec is required"))
	}

	id := fmt.Sprintf("ar-%d", len(s.runs)+1)
	now := timestamppb.Now()

	run := &apiv1.AgentRun{
		Id:   id,
		Name: id,
		Spec: req.Msg.Spec,
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

	return connect.NewResponse(&apiv1.CreateAgentRunResponse{AgentRun: run}), nil
}

func (s *AOTServiceHandler) GetAgentRun(_ context.Context, req *connect.Request[apiv1.GetAgentRunRequest]) (*connect.Response[apiv1.AgentRun], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, ok := s.runs[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}
	return connect.NewResponse(run), nil
}

func (s *AOTServiceHandler) ListAgentRuns(_ context.Context, req *connect.Request[apiv1.ListAgentRunsRequest]) (*connect.Response[apiv1.ListAgentRunsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*apiv1.AgentRun
	for _, run := range s.runs {
		if req.Msg.PhaseFilter != apiv1.AgentRunPhase_AGENT_RUN_PHASE_UNSPECIFIED &&
			run.Status.Phase != req.Msg.PhaseFilter {
			continue
		}
		runs = append(runs, run)
	}

	limit := int(req.Msg.Limit)
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}

	return connect.NewResponse(&apiv1.ListAgentRunsResponse{AgentRuns: runs}), nil
}

func (s *AOTServiceHandler) WatchAgentRun(_ context.Context, req *connect.Request[apiv1.WatchAgentRunRequest], stream *connect.ServerStream[apiv1.AgentRunEvent]) error {
	s.mu.RLock()
	_, ok := s.runs[req.Msg.Id]
	s.mu.RUnlock()
	if !ok {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	ch := make(chan *apiv1.AgentRunEvent, 100)
	s.mu.Lock()
	s.watchers[req.Msg.Id] = append(s.watchers[req.Msg.Id], ch)
	s.mu.Unlock()

	for event := range ch {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *AOTServiceHandler) CancelAgentRun(_ context.Context, req *connect.Request[apiv1.CancelAgentRunRequest]) (*connect.Response[apiv1.CancelAgentRunResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	run.Status.Phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	run.Status.Message = "Cancelled by user"
	run.UpdatedAt = timestamppb.Now()

	return connect.NewResponse(&apiv1.CancelAgentRunResponse{AgentRun: run}), nil
}

func (s *AOTServiceHandler) SendHumanInput(_ context.Context, req *connect.Request[apiv1.SendHumanInputRequest]) (*connect.Response[apiv1.SendHumanInputResponse], error) {
	s.mu.RLock()
	run, ok := s.runs[req.Msg.AgentRunId]
	s.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.AgentRunId))
	}

	if run.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent is not waiting for input"))
	}

	return connect.NewResponse(&apiv1.SendHumanInputResponse{Accepted: true}), nil
}

// EmitEvent sends an event to all watchers of the given agent run.
func (s *AOTServiceHandler) EmitEvent(agentRunID string, event *apiv1.AgentRunEvent) {
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
