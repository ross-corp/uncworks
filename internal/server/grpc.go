// Package server implements the AOT ConnectRPC API server.
package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	temporalclient "go.temporal.io/sdk/client"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/brain"
	cudgelclient "github.com/uncworks/aot/internal/cudgel"
	"github.com/uncworks/aot/internal/embeddings"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/repoutil"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// AOTServiceHandler implements the AOTService ConnectRPC handler.
type AOTServiceHandler struct {
	apiv1connect.UnimplementedAOTServiceHandler

	K8sClient      client.Client
	TemporalClient temporalclient.Client
	EventBus       eventbus.EventBus
	Namespace      string
	LiteLLMBaseURL string

	// Knowledge system (optional — nil means search is unavailable)
	BrainSearcher *brain.Searcher
	Embedder      *embeddings.Embedder
}

var runIDPattern = regexp.MustCompile(`^ar-[a-z0-9]{4,10}$`)

func validateRunID(id string) error {
    if !runIDPattern.MatchString(id) {
        return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run ID format: %q", id))
    }
    return nil
}

// NewAOTServiceHandler creates a new AOTService handler.
func NewAOTServiceHandler(k8sClient client.Client, bus eventbus.EventBus, namespace string) *AOTServiceHandler {
	litellmURL := os.Getenv("LITELLM_BASE_URL")
	if litellmURL == "" {
		litellmURL = "http://litellm:4000"
	}
	return &AOTServiceHandler{
		K8sClient:      k8sClient,
		EventBus:       bus,
		Namespace:      namespace,
		LiteLLMBaseURL: litellmURL,
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

	slog.Debug("apiserver: CreateAgentRun",
		"run_id", name,
		"project", req.Msg.Spec.Project,
		"feature", req.Msg.Spec.Feature,
		"orchestration_mode", req.Msg.Spec.OrchestrationMode,
	)

	// Generate a human-readable display name from the prompt via LLM.
	displayName := s.generateDisplayName(ctx, req.Msg.Spec.Prompt)

	crd := &aotv1alpha1.AgentRun{}
	crd.Name = name
	crd.Namespace = s.Namespace
	crd.Spec = specProtoToCRD(req.Msg.Spec)
	crd.Spec.DisplayName = displayName
	crd.Status.Phase = aotv1alpha1.AgentRunPhasePending
	crd.Status.Message = "Queued"

	// Auto-set labels for project, feature, tags, and repo
	labels := crd.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	if v := sanitizeLabelValue(crd.Spec.Project); v != "" {
		labels["aot.uncworks.io/project"] = v
	}
	if v := sanitizeLabelValue(crd.Spec.Feature); v != "" {
		labels["aot.uncworks.io/feature"] = v
	}
	// Tags stored as annotation (not label) because label values can't contain commas
	if len(crd.Spec.Tags) > 0 {
		if crd.Annotations == nil {
			crd.Annotations = make(map[string]string)
		}
		crd.Annotations["aot.uncworks.io/tags"] = strings.Join(crd.Spec.Tags, ",")
	}
	if len(crd.Spec.Repos) > 0 {
		if v := sanitizeLabelValue(repoNameFromURL(crd.Spec.Repos[0].URL)); v != "" {
			labels["aot.uncworks.io/repo"] = v
		}
	}
	crd.Labels = labels

	if err := s.K8sClient.Create(ctx, crd); err != nil {
		slog.Error("apiserver: failed to create AgentRun CRD", "run_id", name, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create agentrun CRD: %w", err))
	}

	slog.Info("apiserver: AgentRun created", "run_id", name, "display_name", displayName, "namespace", s.Namespace)
	return connect.NewResponse(&apiv1.CreateAgentRunResponse{
		AgentRun: crdToProto(crd),
	}), nil
}

func (s *AOTServiceHandler) GetAgentRun(ctx context.Context, req *connect.Request[apiv1.GetAgentRunRequest]) (*connect.Response[apiv1.AgentRun], error) {
	if err := validateRunID(req.Msg.Id); err != nil {
		return nil, err
	}
	crd := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, crd); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	run := crdToProto(crd)

	// Enrich with real-time Temporal state.
	// Only overwrite the fields that Temporal knows about (Phase, Message, PodName,
	// PrUrl); preserve the richer CRD-sourced fields (TraceId, LogOutput,
	// WorktreePath, Stage, RetryCount, VerificationResult, DebugActive,
	// StartedAt, CompletedAt, RetainUntil) that mapWorkflowStateToProto does not
	// populate.
	if s.TemporalClient != nil {
		workflowID := fmt.Sprintf("agentrun-%s", req.Msg.Id)
		resp, err := s.TemporalClient.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err == nil {
			var state aottemporal.WorkflowState
			if resp.Get(&state) == nil {
				enriched := mapWorkflowStateToProto(state)
				// Merge: only overwrite the fields Temporal provides.
				run.Status.Phase = enriched.Phase
				run.Status.Message = enriched.Message
				if enriched.PodName != "" {
					run.Status.PodName = enriched.PodName
				}
				if enriched.PrUrl != "" {
					run.Status.PrUrl = enriched.PrUrl
				}
			}
		}
	}

	// Populate children by querying runs with matching parentRunID
	var childList aotv1alpha1.AgentRunList
	if err := s.K8sClient.List(ctx, &childList,
		client.InNamespace(s.Namespace),
		client.MatchingLabels{"aot.uncworks.io/spec-run-id": req.Msg.Id},
	); err == nil {
		for _, child := range childList.Items {
			if child.Spec.ParentRunID == req.Msg.Id {
				run.Children = append(run.Children, child.Name)
			}
		}
	}

	return connect.NewResponse(run), nil
}

func (s *AOTServiceHandler) ListAgentRuns(ctx context.Context, req *connect.Request[apiv1.ListAgentRunsRequest]) (*connect.Response[apiv1.ListAgentRunsResponse], error) {
	listOpts := []client.ListOption{client.InNamespace(s.Namespace)}

	// Apply spec_run_id label filter if provided
	if req.Msg.SpecRunId != "" {
		listOpts = append(listOpts, client.MatchingLabels{
			"aot.uncworks.io/spec-run-id": req.Msg.SpecRunId,
		})
	}

	// Apply project label filter if provided
	if req.Msg.ProjectFilter != "" {
		listOpts = append(listOpts, client.MatchingLabels{
			"aot.uncworks.io/project": req.Msg.ProjectFilter,
		})
	}

	// Apply feature label filter if provided
	if req.Msg.FeatureFilter != "" {
		listOpts = append(listOpts, client.MatchingLabels{
			"aot.uncworks.io/feature": req.Msg.FeatureFilter,
		})
	}

	// Determine page size; clamp to [1, 100], default 50.
	pageSize := int64(req.Msg.Limit)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	// Use K8s server-side pagination via Limit + Continue token.
	k8sListOpts := append(listOpts,
		client.Limit(pageSize),
		client.Continue(req.Msg.Cursor),
	)

	var list aotv1alpha1.AgentRunList
	if err := s.K8sClient.List(ctx, &list, k8sListOpts...); err != nil {
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

		// Filter archived runs (unless includeArchived is set via header)
		if crd.Status.Archived && req.Header().Get("X-Include-Archived") != "true" {
			continue
		}

		// Apply phase filter
		if req.Msg.PhaseFilter != apiv1.AgentRunPhase_AGENT_RUN_PHASE_UNSPECIFIED &&
			run.Status.Phase != req.Msg.PhaseFilter {
			continue
		}

		// Apply parent_run_id filter
		if req.Msg.ParentRunId != "" && crd.Spec.ParentRunID != req.Msg.ParentRunId {
			continue
		}

		// Apply stage filter
		if req.Msg.StageFilter != "" && crd.Status.Stage != req.Msg.StageFilter {
			continue
		}

		// Apply tag filter (check if the requested tag is present in the CRD's tags)
		if req.Msg.TagFilter != "" {
			found := false
			for _, t := range crd.Spec.Tags {
				if t == req.Msg.TagFilter {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		runs = append(runs, run)
	}

	// Client-side truncation: K8s fake client ignores Limit, so we enforce it
	// here as a safety net. For a real API server the K8s Limit already applied.
	if int64(len(runs)) > pageSize {
		runs = runs[:pageSize]
	}

	return connect.NewResponse(&apiv1.ListAgentRunsResponse{
		AgentRuns:  runs,
		NextCursor: list.Continue,
	}), nil
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
	if err := validateRunID(req.Msg.Id); err != nil {
		return nil, err
	}
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

func (s *AOTServiceHandler) GetRunGraph(ctx context.Context, req *connect.Request[apiv1.GetRunGraphRequest]) (*connect.Response[apiv1.RunGraph], error) {
	// Get the root run
	rootCRD := &aotv1alpha1.AgentRun{}
	if err := s.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: s.Namespace,
		Name:      req.Msg.Id,
	}, rootCRD); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent run %q not found", req.Msg.Id))
	}

	// Determine the spec-run-id to query
	specRunID := req.Msg.Id
	if rootCRD.Labels != nil {
		if sid, ok := rootCRD.Labels["aot.uncworks.io/spec-run-id"]; ok && sid != "" {
			specRunID = sid
		}
	}

	// Query all runs in this spec execution
	var list aotv1alpha1.AgentRunList
	if err := s.K8sClient.List(ctx, &list,
		client.InNamespace(s.Namespace),
		client.MatchingLabels{"aot.uncworks.io/spec-run-id": specRunID},
	); err != nil {
		// If no orchestration labels, just return the single node
		list.Items = []aotv1alpha1.AgentRun{*rootCRD}
	}

	// If no matching runs found, return just the root
	if len(list.Items) == 0 {
		list.Items = []aotv1alpha1.AgentRun{*rootCRD}
	}

	graph := &apiv1.RunGraph{}
	for _, item := range list.Items {
		node := &apiv1.RunGraphNode{
			Name:  item.Name,
			Phase: crdPhaseToProto(item.Status.Phase),
			Role:  "single",
		}
		if item.Labels != nil {
			if role, ok := item.Labels["aot.uncworks.io/run-role"]; ok {
				node.Role = role
			}
		}
		if item.Status.StartedAt != nil {
			node.StartedAt = timestamppb.New(item.Status.StartedAt.Time)
		}
		if item.Status.CompletedAt != nil {
			node.CompletedAt = timestamppb.New(item.Status.CompletedAt.Time)
		}
		graph.Nodes = append(graph.Nodes, node)

		if item.Spec.ParentRunID != "" {
			graph.Edges = append(graph.Edges, &apiv1.RunGraphEdge{
				Parent: item.Spec.ParentRunID,
				Child:  item.Name,
			})
		}
	}

	return connect.NewResponse(graph), nil
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

// SearchPastWork searches the knowledge base for relevant past work using natural language.
func (s *AOTServiceHandler) SearchPastWork(ctx context.Context, req *connect.Request[apiv1.SearchPastWorkRequest]) (*connect.Response[apiv1.SearchPastWorkResponse], error) {
	if req.Msg.Query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("query is required"))
	}

	// Build limit before branching — both paths use it.
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// SOURCE_CODE: delegate to cudgel instead of the internal code_chunks table.
	// This path does not require BrainSearcher or Embedder.
	if req.Msg.SourceFilter == apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE {
		return s.searchSourceCode(ctx, req.Msg.Query, limit)
	}

	if s.BrainSearcher == nil || s.Embedder == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("knowledge system not configured"))
	}

	// Embed the query
	queryVec, err := s.Embedder.Embed(ctx, req.Msg.Query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("embed query: %w", err))
	}

	sourceFilter := ""
	switch req.Msg.SourceFilter {
	case apiv1.SourceFilter_SOURCE_FILTER_CODE:
		sourceFilter = "code"
	case apiv1.SourceFilter_SOURCE_FILTER_TRACE:
		sourceFilter = "trace"
	}

	var createdAfter, createdBefore *time.Time
	if req.Msg.CreatedAfter != nil {
		t := req.Msg.CreatedAfter.AsTime()
		createdAfter = &t
	}
	if req.Msg.CreatedBefore != nil {
		t := req.Msg.CreatedBefore.AsTime()
		createdBefore = &t
	}

	results, err := s.BrainSearcher.Search(ctx, brain.SearchQuery{
		QueryVec:      queryVec,
		RepoURL:       req.Msg.RepoUrl,
		SourceFilter:  sourceFilter,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Limit:         limit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("search: %w", err))
	}

	var protoResults []*apiv1.PastWorkResult
	for _, r := range results {
		pr := &apiv1.PastWorkResult{
			ChunkText:       r.ChunkText,
			SourceType:      r.SourceType,
			SimilarityScore: r.BoostedScore,
			RunId:           r.AgentRunID,
			FilePath:        r.FilePath,
			Language:        r.Language,
			NodeType:        r.NodeType,
			ChunkType:       r.ChunkType,
			Severity:        r.Severity,
			RepoUrl:         r.RepoURL,
			CreatedAt:       timestamppb.New(r.CreatedAt),
		}
		protoResults = append(protoResults, pr)
	}

	return connect.NewResponse(&apiv1.SearchPastWorkResponse{Results: protoResults}), nil
}

// searchSourceCode handles SearchPastWork with SOURCE_FILTER_SOURCE_CODE by
// delegating to the cudgel HTTP shim. Returns empty results (no error) when
// cudgel is unreachable or CUDGEL_ENDPOINT is unset.
func (s *AOTServiceHandler) searchSourceCode(ctx context.Context, query string, limit int) (*connect.Response[apiv1.SearchPastWorkResponse], error) {
	endpoint := os.Getenv("CUDGEL_ENDPOINT")
	if endpoint == "" {
		return connect.NewResponse(&apiv1.SearchPastWorkResponse{}), nil
	}

	client := cudgelclient.NewHTTPClient(endpoint)
	symbols, err := client.SemanticSearch(ctx, query, limit)
	if err != nil {
		slog.Warn("cudgel searchSourceCode failed, returning empty results", slog.Any("error", err))
		return connect.NewResponse(&apiv1.SearchPastWorkResponse{}), nil
	}

	protoResults := make([]*apiv1.PastWorkResult, 0, len(symbols))
	for _, s := range symbols {
		protoResults = append(protoResults, &apiv1.PastWorkResult{
			ChunkText:       s.Snippet,
			SourceType:      "source_code",
			SimilarityScore: s.Score,
			FilePath:        s.File,
			NodeType:        s.Kind,
		})
	}
	return connect.NewResponse(&apiv1.SearchPastWorkResponse{Results: protoResults}), nil
}

// repoNameFromURL derives a directory name from a git URL.
func repoNameFromURL(repoURL string) string {
	return repoutil.NameFromURL(repoURL)
}

// crdFieldOrLabel returns the spec field value if non-empty, otherwise falls back to the label.
func crdFieldOrLabel(crd *aotv1alpha1.AgentRun, field, labelKey string) string {
	if field != "" {
		return field
	}
	if crd.Labels != nil {
		return crd.Labels[labelKey]
	}
	return ""
}

// crdTagsOrLabel returns spec tags if non-empty, otherwise parses from annotation or label.
func crdTagsOrLabel(crd *aotv1alpha1.AgentRun) []string {
	if len(crd.Spec.Tags) > 0 {
		return crd.Spec.Tags
	}
	// Check annotation first (new format)
	if crd.Annotations != nil {
		if v := crd.Annotations["aot.uncworks.io/tags"]; v != "" {
			return strings.Split(v, ",")
		}
	}
	// Backwards compat: check label (old format)
	if crd.Labels != nil {
		if v := crd.Labels["aot.uncworks.io/tags"]; v != "" {
			return strings.Split(v, ",")
		}
	}
	return nil
}

// sanitizeLabelValue truncates and strips characters that are illegal in
// Kubernetes label values (must match [a-zA-Z0-9._-]{0,63}).
// An empty result means the value should not be applied as a label.
func sanitizeLabelValue(v string) string {
	v = regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(v, "-")
	v = regexp.MustCompile(`^[^a-zA-Z0-9]+`).ReplaceAllString(v, "")
	v = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(v, "")
	if len(v) > 63 {
		v = v[:63]
	}
	return v
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

// displayNameRegex validates generated display names.
var displayNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$`)

// generateDisplayName calls the LiteLLM proxy to generate a short kebab-case
// display name from the run's prompt. Returns an empty string on any failure.
func (s *AOTServiceHandler) generateDisplayName(ctx context.Context, prompt string) string {
	if s.LiteLLMBaseURL == "" || prompt == "" {
		return ""
	}

	// Truncate prompt to 200 characters.
	truncated := prompt
	if len(truncated) > 200 {
		truncated = truncated[:200]
	}

	reqBody := map[string]interface{}{
		"model": "default",
		"messages": []map[string]string{
			{"role": "system", "content": "Generate a short kebab-case name (3-5 words) for this coding task. Output ONLY the name, nothing else."},
			{"role": "user", "content": truncated},
		},
		"max_tokens": 20,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		slog.Warn("failed to marshal display name request", slog.Any("error", err))
		return deriveNameFromPrompt(prompt)
	}

	llmCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	llmBase := strings.TrimSuffix(strings.TrimRight(s.LiteLLMBaseURL, "/"), "/v1")
	url := llmBase + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(llmCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Warn("failed to create display name request", slog.Any("error", err))
		return deriveNameFromPrompt(prompt)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Warn("display name LLM call failed, using prompt fallback", slog.Any("error", err))
		return deriveNameFromPrompt(prompt)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("display name LLM returned non-200 status, using prompt fallback", "status", resp.StatusCode)
		return deriveNameFromPrompt(prompt)
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		slog.Warn("failed to read display name response", slog.Any("error", err))
		return deriveNameFromPrompt(prompt)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		slog.Warn("failed to parse display name response", slog.Any("error", err))
		return deriveNameFromPrompt(prompt)
	}

	if len(result.Choices) == 0 {
		slog.Warn("display name LLM returned no choices")
		return deriveNameFromPrompt(prompt)
	}

	name := strings.TrimSpace(strings.ToLower(result.Choices[0].Message.Content))

	if !displayNameRegex.MatchString(name) {
		slog.Warn("generated display name failed validation, falling back to prompt derivation", "name", name)
		return deriveNameFromPrompt(prompt)
	}

	return name
}

// deriveNameFromPrompt creates a simple kebab-case name from the first 5 words
// of a prompt. Used as a fallback when LLM-based name generation fails.
func deriveNameFromPrompt(prompt string) string {
	words := strings.Fields(prompt)
	if len(words) > 5 {
		words = words[:5]
	}
	name := strings.Join(words, "-")
	name = strings.ToLower(name)
	// Remove non-alphanumeric except hyphens
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 50 {
		name = name[:50]
	}
	if name == "" {
		return ""
	}
	return name
}

// protoBackendToCRD maps the proto Backend enum to the CRD BackendType.
func protoBackendToCRD(b apiv1.Backend) aotv1alpha1.BackendType {
	return aotv1alpha1.BackendPod
}

// crdBackendToProto maps the CRD BackendType to the proto Backend enum.
func crdBackendToProto(b aotv1alpha1.BackendType) apiv1.Backend {
	return apiv1.Backend_BACKEND_POD
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
		Backend:           protoBackendToCRD(spec.Backend),
		Repos:             repos,
		Prompt:            spec.Prompt,
		DevboxConfig:      spec.DevboxConfig,
		TTLSeconds:        spec.TtlSeconds,
		EnvVars:           spec.EnvVars,
		ModelTier:         spec.ModelTier,
		Image:             spec.Image,
		SpecContent:       spec.SpecContent,
		SpecSource:        spec.SpecSource,
		WorkspaceName:     spec.WorkspaceName,
		ParentRunID:       spec.ParentRunId,
		OrchestrationMode: protoOrchModeToCRD(spec.OrchestrationMode),
		SpecRunID:         spec.SpecRunId,
		DisplayName:       spec.DisplayName,
		MaxBudget:         spec.MaxBudget,
		AutoPush:          spec.AutoPush,
		AutoPR:            spec.AutoPr,
		PRBaseBranch:      spec.PrBaseBranch,
		Project:           spec.Project,
		Feature:           spec.Feature,
		Tags:              spec.Tags,
		ProjectRef:        spec.ProjectRef,
		SpecRef:           spec.SpecRef,
	}
	if spec.PipelineConfig != nil {
		crdSpec.PipelineConfig = &aotv1alpha1.PipelineConfig{
			Plan:    protoStageConfigToCRD(spec.PipelineConfig.Plan),
			Execute: protoStageConfigToCRD(spec.PipelineConfig.Execute),
			Verify:  protoStageConfigToCRD(spec.PipelineConfig.Verify),
		}
	}
	if spec.Orchestration != nil && len(spec.Orchestration.Tasks) > 0 {
		orch := &aotv1alpha1.Orchestration{}
		for _, t := range spec.Orchestration.Tasks {
			orch.Tasks = append(orch.Tasks, aotv1alpha1.OrchestrationTask{
				Name:     t.Name,
				Prompt:   t.Prompt,
				RepoURLs: t.RepoUrls,
			})
		}
		crdSpec.Orchestration = orch
	}
	return crdSpec
}

func protoStageConfigToCRD(sc *apiv1.StageConfig) aotv1alpha1.StageConfig {
	if sc == nil {
		return aotv1alpha1.StageConfig{}
	}
	return aotv1alpha1.StageConfig{
		Model:          sc.Model,
		TimeoutSeconds: sc.TimeoutSeconds,
		MaxRetries:     sc.MaxRetries,
		OnFailure:      sc.OnFailure,
	}
}

func crdStageConfigToProto(sc aotv1alpha1.StageConfig) *apiv1.StageConfig {
	return &apiv1.StageConfig{
		Model:          sc.Model,
		TimeoutSeconds: sc.TimeoutSeconds,
		MaxRetries:     sc.MaxRetries,
		OnFailure:      sc.OnFailure,
	}
}

func protoOrchModeToCRD(m apiv1.OrchestrationMode) aotv1alpha1.OrchestrationMode {
	switch m {
	case apiv1.OrchestrationMode_ORCHESTRATION_MODE_AUTO:
		return aotv1alpha1.OrchestrationModeAuto
	case apiv1.OrchestrationMode_ORCHESTRATION_MODE_MANUAL:
		return aotv1alpha1.OrchestrationModeManual
	case apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN:
		return aotv1alpha1.OrchestrationModeSpecDriven
	default:
		return aotv1alpha1.OrchestrationModeSingle
	}
}

func crdOrchModeToProto(m aotv1alpha1.OrchestrationMode) apiv1.OrchestrationMode {
	switch m {
	case aotv1alpha1.OrchestrationModeAuto:
		return apiv1.OrchestrationMode_ORCHESTRATION_MODE_AUTO
	case aotv1alpha1.OrchestrationModeManual:
		return apiv1.OrchestrationMode_ORCHESTRATION_MODE_MANUAL
	case aotv1alpha1.OrchestrationModeSingle:
		return apiv1.OrchestrationMode_ORCHESTRATION_MODE_SINGLE
	case aotv1alpha1.OrchestrationModeSpecDriven:
		return apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN
	default:
		return apiv1.OrchestrationMode_ORCHESTRATION_MODE_UNSPECIFIED
	}
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
	protoSpec := &apiv1.AgentRunSpec{
		Backend:           crdBackendToProto(crd.Spec.Backend),
		Repos:             protoRepos,
		Prompt:            crd.Spec.Prompt,
		DevboxConfig:      crd.Spec.DevboxConfig,
		TtlSeconds:        crd.Spec.TTLSeconds,
		EnvVars:           crd.Spec.EnvVars,
		ModelTier:         crd.Spec.ModelTier,
		Image:             crd.Spec.Image,
		SpecContent:       crd.Spec.SpecContent,
		SpecSource:        crd.Spec.SpecSource,
		WorkspaceName:     crd.Spec.WorkspaceName,
		ParentRunId:       crd.Spec.ParentRunID,
		OrchestrationMode: crdOrchModeToProto(crd.Spec.OrchestrationMode),
		SpecRunId:         crd.Spec.SpecRunID,
		DisplayName:       crd.Spec.DisplayName,
		MaxBudget:         crd.Spec.MaxBudget,
		AutoPush:          crd.Spec.AutoPush,
		AutoPr:            crd.Spec.AutoPR,
		PrBaseBranch:      crd.Spec.PRBaseBranch,
		Project:           crdFieldOrLabel(crd, crd.Spec.Project, "aot.uncworks.io/project"),
		Feature:           crdFieldOrLabel(crd, crd.Spec.Feature, "aot.uncworks.io/feature"),
		Tags:              crdTagsOrLabel(crd),
	}
	if crd.Spec.PipelineConfig != nil {
		protoSpec.PipelineConfig = &apiv1.PipelineConfig{
			Plan:    crdStageConfigToProto(crd.Spec.PipelineConfig.Plan),
			Execute: crdStageConfigToProto(crd.Spec.PipelineConfig.Execute),
			Verify:  crdStageConfigToProto(crd.Spec.PipelineConfig.Verify),
		}
	}
	if crd.Spec.Orchestration != nil {
		orch := &apiv1.Orchestration{}
		for _, t := range crd.Spec.Orchestration.Tasks {
			orch.Tasks = append(orch.Tasks, &apiv1.OrchestrationTask{
				Name:     t.Name,
				Prompt:   t.Prompt,
				RepoUrls: t.RepoURLs,
			})
		}
		protoSpec.Orchestration = orch
	}

	run := &apiv1.AgentRun{
		Id:   crd.Name,
		Name: crd.Name,
		Spec: protoSpec,
		Status: &apiv1.AgentRunStatus{
			Phase:              crdPhaseToProto(crd.Status.Phase),
			Message:            crd.Status.Message,
			PodName:            crd.Status.PodName,
			TraceId:            crd.Status.TraceID,
			WorktreePath:       crd.Status.WorktreePath,
			LogOutput:          crd.Status.LogOutput,
			DeploymentName:     crd.Status.DeploymentName,
			DebugActive:        crd.Status.DebugActive,
			Stage:              crd.Status.Stage,
			RetryCount:         crd.Status.RetryCount,
			VerificationResult: crd.Status.VerificationResult,
			PrUrl:              crd.Status.PRUrl,
		},
		CreatedAt: timestamppb.New(crd.CreationTimestamp.Time),
	}

	if crd.Status.StartedAt != nil {
		run.Status.StartedAt = timestamppb.New(crd.Status.StartedAt.Time)
	}
	if crd.Status.CompletedAt != nil {
		run.Status.CompletedAt = timestamppb.New(crd.Status.CompletedAt.Time)
	}
	if crd.Status.RetainUntil != nil {
		run.Status.RetainUntil = timestamppb.New(crd.Status.RetainUntil.Time)
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
		PrUrl:   state.PRUrl,
	}
}
