// Package server implements the AOT ConnectRPC API server.
package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	temporalclient "go.temporal.io/sdk/client"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// AOTServiceHandler implements the AOTService ConnectRPC handler.
type AOTServiceHandler struct {
	apiv1connect.UnimplementedAOTServiceHandler

	K8sClient      client.Client
	TemporalClient temporalclient.Client
	EventBus       eventbus.EventBus
	Namespace      string
}

// NewAOTServiceHandler creates a new AOTService handler.
func NewAOTServiceHandler(k8sClient client.Client, bus eventbus.EventBus, namespace string) *AOTServiceHandler {
	return &AOTServiceHandler{
		K8sClient: k8sClient,
		EventBus:  bus,
		Namespace: namespace,
	}
}

func (s *AOTServiceHandler) CreateAgentRun(ctx context.Context, req *connect.Request[apiv1.CreateAgentRunRequest]) (*connect.Response[apiv1.CreateAgentRunResponse], error) {
	if req.Msg.Spec == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("spec is required"))
	}

	name, err := generateRunName()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generate name: %w", err))
	}

	crd := &aotv1alpha1.AgentRun{}
	crd.Name = name
	crd.Namespace = s.Namespace
	crd.Spec = specProtoToCRD(req.Msg.Spec)
	crd.Status.Phase = aotv1alpha1.AgentRunPhasePending
	crd.Status.Message = "Queued"

	if err := s.K8sClient.Create(ctx, crd); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create agentrun CRD: %w", err))
	}

	return connect.NewResponse(&apiv1.CreateAgentRunResponse{
		AgentRun: crdToProto(crd),
	}), nil
}

func (s *AOTServiceHandler) GetAgentRun(ctx context.Context, req *connect.Request[apiv1.GetAgentRunRequest]) (*connect.Response[apiv1.AgentRun], error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, crd); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	run := crdToProto(crd)

	// Enrich with real-time Temporal state
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.Id)
		resp, err := s.TemporalClient.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err == nil {
			var state aottemporal.WorkflowState
			if resp.Get(&state) == nil {
				run.Status = mapWorkflowStateToProto(state)
			}
		}
	}

	return connect.NewResponse(run), nil
}

func (s *AOTServiceHandler) ListAgentRuns(ctx context.Context, req *connect.Request[apiv1.ListAgentRunsRequest]) (*connect.Response[apiv1.ListAgentRunsResponse], error) {
	var list aotv1alpha1.AgentRunList
	if err := s.K8sClient.List(ctx, &list, client.InNamespace(s.Namespace)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list agentruns: %w", err))
	}

	// Sort by creation time (newest first)
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[j].CreationTimestamp.Before(&list.Items[i].CreationTimestamp)
	})

	var runs []*apiv1.AgentRun
	for i := range list.Items {
		crd := &list.Items[i]
		run := crdToProto(crd)

		// Apply phase filter
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
	crd := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, crd); err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	run := crdToProto(crd)

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
		return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("event streaming not configured"))
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
			if event.Type == apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED {
				return nil
			}
		}
	}
}

func (s *AOTServiceHandler) CancelAgentRun(ctx context.Context, req *connect.Request[apiv1.CancelAgentRunRequest]) (*connect.Response[apiv1.CancelAgentRunResponse], error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, crd); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	// Cancel via Temporal workflow
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.Id)
		if err := s.TemporalClient.CancelWorkflow(ctx, workflowID, ""); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("cancel workflow: %w", err))
		}
	}

	// Re-read to get latest state after cancellation signal
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, crd); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("re-read agentrun: %w", err))
	}

	return connect.NewResponse(&apiv1.CancelAgentRunResponse{AgentRun: crdToProto(crd)}), nil
}

func (s *AOTServiceHandler) SendHumanInput(ctx context.Context, req *connect.Request[apiv1.SendHumanInputRequest]) (*connect.Response[apiv1.SendHumanInputResponse], error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.AgentRunId,
	}, crd); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.AgentRunId))
	}

	if crd.Status.Phase != aotv1alpha1.AgentRunPhaseWaitingForInput {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent is not waiting for input"))
	}

	if s.TemporalClient == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("temporal not configured"))
	}

	workflowID := fmt.Sprintf("agentrun-%s", req.Msg.AgentRunId)
	signal := aottemporal.HumanInputSignal{Input: req.Msg.Input}
	if err := s.TemporalClient.SignalWorkflow(ctx, workflowID, "", aottemporal.SignalHumanInput, signal); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("signal workflow: %w", err))
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

// generateRunName creates a random name like "ar-a1b2c3".
func generateRunName() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 6)
	for i := range suffix {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		suffix[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("ar-%s", string(suffix)), nil
}

// specProtoToCRD converts a proto AgentRunSpec to a CRD AgentRunSpec.
func specProtoToCRD(spec *apiv1.AgentRunSpec) aotv1alpha1.AgentRunSpec {
	var repos []aotv1alpha1.Repository
	for _, r := range spec.Repos {
		repos = append(repos, aotv1alpha1.Repository{
			URL:    r.Url,
			Branch: r.Branch,
			Path:   r.Path,
		})
	}
	crdSpec := aotv1alpha1.AgentRunSpec{
		Backend:      aotv1alpha1.BackendPod,
		Repos:        repos,
		Prompt:       spec.Prompt,
		DevboxConfig: spec.DevboxConfig,
		TTLSeconds:   spec.TtlSeconds,
		EnvVars:      spec.EnvVars,
		ModelTier:    spec.ModelTier,
		Image:        spec.Image,
	}
	return crdSpec
}

// crdToProto converts a CRD AgentRun to a proto AgentRun.
func crdToProto(crd *aotv1alpha1.AgentRun) *apiv1.AgentRun {
	var protoRepos []*apiv1.Repository
	for _, r := range crd.Spec.Repos {
		protoRepos = append(protoRepos, &apiv1.Repository{
			Url:    r.URL,
			Branch: r.Branch,
			Path:   r.Path,
		})
	}
	run := &apiv1.AgentRun{
		Id:   crd.Name,
		Name: crd.Name,
		Spec: &apiv1.AgentRunSpec{
			Repos:        protoRepos,
			Prompt:       crd.Spec.Prompt,
			DevboxConfig: crd.Spec.DevboxConfig,
			TtlSeconds:   crd.Spec.TTLSeconds,
			EnvVars:      crd.Spec.EnvVars,
			ModelTier:    crd.Spec.ModelTier,
			Image:        crd.Spec.Image,
		},
		Status: &apiv1.AgentRunStatus{
			Phase:        crdPhaseToProto(crd.Status.Phase),
			Message:      crd.Status.Message,
			PodName:      crd.Status.PodName,
			TraceId:      crd.Status.TraceID,
			WorktreePath: crd.Status.WorktreePath,
		},
		CreatedAt: timestamppb.New(crd.CreationTimestamp.Time),
	}

	if crd.Status.StartedAt != nil {
		run.Status.StartedAt = timestamppb.New(crd.Status.StartedAt.Time)
	}
	if crd.Status.CompletedAt != nil {
		run.Status.CompletedAt = timestamppb.New(crd.Status.CompletedAt.Time)
	}

	return run
}

// crdPhaseToProto maps CRD phase strings to proto enum values.
func crdPhaseToProto(phase aotv1alpha1.AgentRunPhase) apiv1.AgentRunPhase {
	switch phase {
	case aotv1alpha1.AgentRunPhasePending:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
	case aotv1alpha1.AgentRunPhaseRunning:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
	case aotv1alpha1.AgentRunPhaseWaitingForInput:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT
	case aotv1alpha1.AgentRunPhaseSucceeded:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED
	case aotv1alpha1.AgentRunPhaseFailed:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED
	case aotv1alpha1.AgentRunPhaseCancelled:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED
	default:
		return apiv1.AgentRunPhase_AGENT_RUN_PHASE_UNSPECIFIED
	}
}

// mapWorkflowStateToProto converts a Temporal workflow state to a proto status.
func mapWorkflowStateToProto(state aottemporal.WorkflowState) *apiv1.AgentRunStatus {
	phase := apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
	switch state.Phase {
	case "Creating":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING
	case "Hydrating":
		phase = apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING
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
