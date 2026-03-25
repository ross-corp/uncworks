package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/protobuf/types/known/timestamppb"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/internal/eventbus"
)

// SSEHandler serves Server-Sent Event endpoints for real-time graph and trace updates.
type SSEHandler struct {
	k8sClient runtimeclient.Client
	eventBus  eventbus.EventBus
	namespace string
}

// NewSSEHandler creates a new SSEHandler.
func NewSSEHandler(k8sClient runtimeclient.Client, bus eventbus.EventBus, namespace string) *SSEHandler {
	return &SSEHandler{
		k8sClient: k8sClient,
		eventBus:  bus,
		namespace: namespace,
	}
}

// RegisterSSEHandlers registers the SSE REST endpoints on the given mux.
func (s *SSEHandler) RegisterSSEHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/specs/{id}/graph", s.handleGetGraph)
	mux.HandleFunc("GET /api/v1/specs/{id}/graph/watch", s.handleWatchGraph)
	mux.HandleFunc("GET /api/v1/runs/{id}/traces/watch", s.handleWatchTraces)
}

// graphResponse is the JSON structure returned by the graph endpoint.
type graphResponse struct {
	Nodes []graphNodeJSON `json:"nodes"`
	Edges []graphEdgeJSON `json:"edges"`
}

type graphNodeJSON struct {
	RunID       string `json:"runId"`
	ParentRunID string `json:"parentRunId,omitempty"`
	AgentType   string `json:"agentType"`
	Phase       string `json:"phase"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

type graphEdgeJSON struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
}

// handleGetGraph returns the run graph as JSON, matching the shape the frontend expects.
func (s *SSEHandler) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	specRunID := r.PathValue("id")

	// Query all runs in this spec execution
	var list aotv1alpha1.AgentRunList
	if err := s.k8sClient.List(r.Context(), &list,
		runtimeclient.InNamespace(s.namespace),
		runtimeclient.MatchingLabels{"aot.uncworks.io/spec-run-id": specRunID},
	); err != nil {
		// Try to find a single run with this ID instead
		crd := &aotv1alpha1.AgentRun{}
		if getErr := s.k8sClient.Get(r.Context(), runtimeclient.ObjectKey{
			Namespace: s.namespace,
			Name:      specRunID,
		}, crd); getErr != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("spec run %q not found", specRunID)})
			return
		}
		list.Items = []aotv1alpha1.AgentRun{*crd}
	}

	if len(list.Items) == 0 {
		// Try single run lookup
		crd := &aotv1alpha1.AgentRun{}
		if err := s.k8sClient.Get(r.Context(), runtimeclient.ObjectKey{
			Namespace: s.namespace,
			Name:      specRunID,
		}, crd); err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("spec run %q not found", specRunID)})
			return
		}
		list.Items = []aotv1alpha1.AgentRun{*crd}
	}

	resp := graphResponse{
		Nodes: make([]graphNodeJSON, 0, len(list.Items)),
		Edges: make([]graphEdgeJSON, 0),
	}

	for _, item := range list.Items {
		role := "single"
		if item.Labels != nil {
			if r, ok := item.Labels["aot.uncworks.io/run-role"]; ok {
				role = r
			}
		}

		node := graphNodeJSON{
			RunID:     item.Name,
			AgentType: role,
			Phase:     phaseToString(crdPhaseToProto(item.Status.Phase)),
		}

		if item.Spec.ParentRunID != "" {
			node.ParentRunID = item.Spec.ParentRunID
		}
		if item.Status.StartedAt != nil {
			node.StartedAt = item.Status.StartedAt.Format("2006-01-02T15:04:05Z")
		}
		if item.Status.CompletedAt != nil {
			node.CompletedAt = item.Status.CompletedAt.Format("2006-01-02T15:04:05Z")
		}

		resp.Nodes = append(resp.Nodes, node)

		if item.Spec.ParentRunID != "" {
			resp.Edges = append(resp.Edges, graphEdgeJSON{
				Parent: item.Spec.ParentRunID,
				Child:  item.Name,
			})
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// graphSSEEvent is the JSON payload sent over the SSE stream for graph updates.
type graphSSEEvent struct {
	Type            string `json:"type"` // NODE_ADDED, NODE_STATUS_CHANGED, NODE_PROGRESS
	RunID           string `json:"runId"`
	ParentRunID     string `json:"parentRunId,omitempty"`
	AgentType       string `json:"agentType,omitempty"`
	Phase           string `json:"phase,omitempty"`
	Message         string `json:"message,omitempty"`
	CurrentActivity string `json:"currentActivity,omitempty"`
}

// handleWatchGraph sends SSE events for graph updates on a spec run.
func (s *SSEHandler) handleWatchGraph(w http.ResponseWriter, r *http.Request) {
	specRunID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if s.eventBus == nil {
		// No event bus — send a heartbeat and wait for client disconnect
		_, _ = fmt.Fprintf(w, ": connected\n\n")
		flusher.Flush()
		<-r.Context().Done()
		return
	}

	// Subscribe to events for this spec run ID
	ch, subID := s.eventBus.Subscribe(specRunID)
	defer s.eventBus.Unsubscribe(specRunID, subID)

	// Send initial connection comment
	_, _ = fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			sseEvent := agentRunEventToGraphSSE(event)
			data, err := json.Marshal(sseEvent)
			if err != nil {
				slog.Error("SSE: failed to marshal graph event", "err", err)
				continue
			}

			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleWatchTraces sends SSE events for trace span updates on a run.
func (s *SSEHandler) handleWatchTraces(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if s.eventBus == nil {
		_, _ = fmt.Fprintf(w, ": connected\n\n")
		flusher.Flush()
		<-r.Context().Done()
		return
	}

	// Subscribe to events for this run
	ch, subID := s.eventBus.Subscribe(runID)
	defer s.eventBus.Unsubscribe(runID, subID)

	// Send initial connection comment
	_, _ = fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			// Convert AgentRunEvent to a trace-span-style SSE event
			spanEvent := agentRunEventToTraceSSE(event)
			if spanEvent == nil {
				continue
			}

			data, err := json.Marshal(spanEvent)
			if err != nil {
				slog.Error("SSE: failed to marshal trace event", "err", err)
				continue
			}

			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// agentRunEventToGraphSSE converts an AgentRunEvent to a graph SSE event.
func agentRunEventToGraphSSE(event *apiv1.AgentRunEvent) graphSSEEvent {
	switch event.Type {
	case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
		return graphSSEEvent{
			Type:    "NODE_STATUS_CHANGED",
			RunID:   event.AgentRunId,
			Phase:   event.Payload,
			Message: event.Payload,
		}
	case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
		return graphSSEEvent{
			Type:            "NODE_PROGRESS",
			RunID:           event.AgentRunId,
			CurrentActivity: event.Payload,
		}
	case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED:
		return graphSSEEvent{
			Type:    "NODE_STATUS_CHANGED",
			RunID:   event.AgentRunId,
			Phase:   "succeeded",
			Message: event.Payload,
		}
	default:
		return graphSSEEvent{
			Type:    "NODE_STATUS_CHANGED",
			RunID:   event.AgentRunId,
			Phase:   event.Payload,
			Message: event.Payload,
		}
	}
}

// traceSSESpan is a minimal trace span event for SSE delivery.
type traceSSESpan struct {
	ID        string `json:"id"`
	ParentID  string `json:"parentId,omitempty"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// agentRunEventToTraceSSE converts an AgentRunEvent to a trace span SSE event.
// Returns nil for events that don't map to trace spans.
func agentRunEventToTraceSSE(event *apiv1.AgentRunEvent) *traceSSESpan {
	switch event.Type {
	case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED:
		now := timestamppb.Now().AsTime().Format("2006-01-02T15:04:05Z")
		return &traceSSESpan{
			ID:        fmt.Sprintf("%s-phase-%s", event.AgentRunId, event.Payload),
			Name:      fmt.Sprintf("Phase: %s", event.Payload),
			Type:      "phase",
			StartTime: now,
			EndTime:   now,
		}
	case apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG:
		now := timestamppb.Now().AsTime().Format("2006-01-02T15:04:05Z")
		return &traceSSESpan{
			ID:        fmt.Sprintf("%s-log-%d", event.AgentRunId, timestamppb.Now().GetSeconds()),
			Name:      event.Payload,
			Type:      "log",
			StartTime: now,
			EndTime:   now,
		}
	default:
		return nil
	}
}

// phaseToString converts a proto AgentRunPhase to a lowercase string for JSON.
func phaseToString(phase apiv1.AgentRunPhase) string {
	switch phase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING:
		return "pending"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING:
		return "running"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
		return "waiting_for_input"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		return "succeeded"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		return "failed"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}
