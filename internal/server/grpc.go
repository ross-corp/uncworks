// Package server implements the AOT ConnectRPC API server.
package server

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	temporalclient "go.temporal.io/sdk/client"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// AOTServiceHandler implements the AOTService ConnectRPC handler.
type AOTServiceHandler struct {
	apiv1connect.UnimplementedAOTServiceHandler

	mu             sync.RWMutex
	runs           map[string]*apiv1.AgentRun
	TemporalClient temporalclient.Client
	EventBus       eventbus.EventBus
}

// NewAOTServiceHandler creates a new AOTService handler.
func NewAOTServiceHandler(bus eventbus.EventBus) *AOTServiceHandler {
	return &AOTServiceHandler{
		runs:     make(map[string]*apiv1.AgentRun),
		EventBus: bus,
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

func (s *AOTServiceHandler) GetAgentRun(ctx context.Context, req *connect.Request[apiv1.GetAgentRunRequest]) (*connect.Response[apiv1.AgentRun], error) {
	s.mu.RLock()
	run, ok := s.runs[req.Msg.Id]
	s.mu.RUnlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	// Optionally enrich with real-time Temporal state
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.Id)
		resp, err := s.TemporalClient.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err == nil {
			var state aottemporal.WorkflowState
			if resp.Get(&state) == nil {
				run = cloneRunWithStatus(run, mapWorkflowStateToProto(state))
			}
		}
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

func (s *AOTServiceHandler) WatchAgentRun(ctx context.Context, req *connect.Request[apiv1.WatchAgentRunRequest], stream *connect.ServerStream[apiv1.AgentRunEvent]) error {
	s.mu.RLock()
	run, ok := s.runs[req.Msg.Id]
	s.mu.RUnlock()
	if !ok {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	// Send current state as initial event
	initialEvent := &apiv1.AgentRunEvent{
		AgentRunId: run.Id,
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
		Payload:    run.Status.Phase.String(),
	}
	if err := stream.Send(initialEvent); err != nil {
		return err
	}

	// If already terminal, close immediately
	if isTerminalPhase(run.Status.Phase) {
		return nil
	}

	// Subscribe to event bus
	if s.EventBus == nil {
		return nil
	}
	ch, subID := s.EventBus.Subscribe(req.Msg.Id)
	defer s.EventBus.Unsubscribe(req.Msg.Id, subID)

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(event); err != nil {
				return err
			}
			// Close stream on terminal events
			if event.Type == apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED {
				return nil
			}
		}
	}
}

func (s *AOTServiceHandler) CancelAgentRun(ctx context.Context, req *connect.Request[apiv1.CancelAgentRunRequest]) (*connect.Response[apiv1.CancelAgentRunResponse], error) {
	s.mu.Lock()
	run, ok := s.runs[req.Msg.Id]
	s.mu.Unlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	// Cancel via Temporal workflow if client is configured (best-effort)
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.Id)
		_ = s.TemporalClient.CancelWorkflow(ctx, workflowID, "")
	}

	s.mu.Lock()
	run.Status.Phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	run.Status.Message = "Cancelled by user"
	run.UpdatedAt = timestamppb.Now()
	s.mu.Unlock()

	return connect.NewResponse(&apiv1.CancelAgentRunResponse{AgentRun: run}), nil
}

func (s *AOTServiceHandler) SendHumanInput(ctx context.Context, req *connect.Request[apiv1.SendHumanInputRequest]) (*connect.Response[apiv1.SendHumanInputResponse], error) {
	s.mu.RLock()
	run, ok := s.runs[req.Msg.AgentRunId]
	s.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.AgentRunId))
	}

	if run.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent is not waiting for input"))
	}

	// Send via Temporal signal if client is configured
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.AgentRunId)
		signal := aottemporal.HumanInputSignal{Input: req.Msg.Input}
		if err := s.TemporalClient.SignalWorkflow(ctx, workflowID, "", aottemporal.SignalHumanInput, signal); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("signal workflow: %w", err))
		}
	}

	return connect.NewResponse(&apiv1.SendHumanInputResponse{Accepted: true}), nil
}

// isTerminalPhase returns true for phases that indicate a completed run.
func isTerminalPhase(phase apiv1.AgentRunPhase) bool {
	switch phase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
		apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED,
		apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		return true
	}
	return false
}

// cloneRunWithStatus creates a copy of an AgentRun with a new status to avoid mutating the stored version.
func cloneRunWithStatus(run *apiv1.AgentRun, status *apiv1.AgentRunStatus) *apiv1.AgentRun {
	return &apiv1.AgentRun{
		Id:        run.Id,
		Name:      run.Name,
		Spec:      run.Spec,
		Status:    status,
		CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt,
	}
}

// mapWorkflowStateToProto converts a Temporal workflow state to a proto status.
func mapWorkflowStateToProto(state aottemporal.WorkflowState) *apiv1.AgentRunStatus {
	phase := apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
	switch state.Phase {
	case "Running":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
	case "WaitingForInput":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
	case "Succeeded":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
	case "Failed":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
	case "Cancelled", "Cancelling":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	}
	return &apiv1.AgentRunStatus{
		Phase:   phase,
		Message: state.Message,
		PodName: state.PodName,
	}
}
