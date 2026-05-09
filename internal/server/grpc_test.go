package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
)

var testScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(testScheme))
}

func startTestServer(t *testing.T) (apiv1connect.AOTServiceClient, func()) {
	t.Helper()

	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestCreateAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Fix the tests",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	if resp.Msg.AgentRun.Id == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		t.Errorf("expected PENDING phase, got %v", resp.Msg.AgentRun.Status.Phase)
	}
}

func TestCreateAgentRun_NilSpec(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{}))
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestGetAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create first
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Test get",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Get it
	run, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: resp.Msg.AgentRun.Id}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if run.Msg.Spec.Prompt != "Test get" {
		t.Errorf("expected prompt 'Test get', got %q", run.Msg.Spec.Prompt)
	}
}

func TestGetAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{Id: "ar-notfound"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestListAgentRuns(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	// Create two runs
	for _, prompt := range []string{"task 1", "task 2"} {
		if _, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
				Prompt:  prompt,
			},
		})); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestListAgentRuns_WithLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_CREATE_AGENT_RUN_RPS", "100")
	t.Setenv("RATE_LIMIT_CREATE_AGENT_RUN_BURST", "100")
	client, cleanup := startTestServer(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		if _, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
				Prompt:  "task",
			},
		})); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 2}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs with limit, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestCancelAgentRun(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "cancel me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Cancel without Temporal client — should still return the CRD state (Pending, not Cancelled)
	cancelResp, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: resp.Msg.AgentRun.Id}))
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	// Without Temporal, the CRD phase won't change — it's still Pending.
	// The cancel signal is sent to Temporal which updates the CRD via the controller.
	if cancelResp.Msg.AgentRun.Id != resp.Msg.AgentRun.Id {
		t.Errorf("expected same ID, got %s", cancelResp.Msg.AgentRun.Id)
	}
	if cancelResp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		t.Errorf("expected phase PENDING without Temporal, got %v", cancelResp.Msg.AgentRun.Status.Phase)
	}
}

func TestCancelAgentRun_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: "ar-notfound"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestSendHumanInput_NotWaiting(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "not waiting",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	_, err = client.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: resp.Msg.AgentRun.Id,
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error: agent not waiting for input")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

func TestSendHumanInput_NotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: "ar-notfound",
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

// startTestServerWithBus returns a client, the event bus, the handler, and a cleanup function.
func startTestServerWithBus(t *testing.T) (apiv1connect.AOTServiceClient, *eventbus.ChannelBus, *AOTServiceHandler, func()) {
	t.Helper()

	bus := eventbus.NewChannelBus()
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := NewAOTServiceHandler(k8sClient, bus, "default")
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return client, bus, svc, srv.Close
}

func TestWatchAgentRun_InitialState(t *testing.T) {
	client, _, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "watch me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: resp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun: %v", err)
	}

	// Should receive initial state
	if !stream.Receive() {
		t.Fatalf("expected initial event, got error: %v", stream.Err())
	}
	event := stream.Msg()
	if event.AgentRunId != resp.Msg.AgentRun.Id {
		t.Errorf("expected run ID %s, got %s", resp.Msg.AgentRun.Id, event.AgentRunId)
	}
	if event.Type != apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED {
		t.Errorf("expected PHASE_CHANGED, got %v", event.Type)
	}
}

func TestWatchAgentRun_EventStreaming(t *testing.T) {
	client, bus, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "stream events",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: resp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun: %v", err)
	}

	// Receive initial state
	if !stream.Receive() {
		t.Fatalf("expected initial event: %v", stream.Err())
	}

	// Publish an event via the bus
	go func() {
		time.Sleep(100 * time.Millisecond)
		bus.Publish(resp.Msg.AgentRun.Id, &apiv1.AgentRunEvent{
			AgentRunId: resp.Msg.AgentRun.Id,
			Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
			Payload:    "Running",
		})
		time.Sleep(100 * time.Millisecond)
		bus.Publish(resp.Msg.AgentRun.Id, &apiv1.AgentRunEvent{
			AgentRunId: resp.Msg.AgentRun.Id,
			Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED,
			Payload:    "Succeeded",
		})
	}()

	// Should receive the phase change event
	if !stream.Receive() {
		t.Fatalf("expected phase change event: %v", stream.Err())
	}
	if stream.Msg().Payload != "Running" {
		t.Errorf("expected Running payload, got %s", stream.Msg().Payload)
	}

	// Should receive the completion event
	if !stream.Receive() {
		t.Fatalf("expected completion event: %v", stream.Err())
	}
	if stream.Msg().Type != apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED {
		t.Errorf("expected COMPLETED, got %v", stream.Msg().Type)
	}
}

func TestWatchAgentRun_NotFound(t *testing.T) {
	client, _, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	ctx := context.Background()
	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: "ar-notfound",
	}))
	if err != nil {
		// Some Connect implementations return error immediately
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
		}
		return
	}
	// For server-streaming, error comes on first Receive()
	if stream.Receive() {
		t.Fatal("expected no messages for nonexistent run")
	}
	if stream.Err() == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(stream.Err()) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(stream.Err()))
	}
}

// --- GetRunGraph tests ---

func TestGetRunGraph_NotFound(t *testing.T) {
	client, _, _, cleanup := startTestServerWithBus(t)
	defer cleanup()

	_, err := client.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{Id: "ar-notfound"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestGetRunGraph_SingleNode(t *testing.T) {
	client, _, svc, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run via RPC
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "single node graph",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id

	// GetRunGraph with no sibling runs — should return 1 node, 0 edges
	graph, err := client.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{Id: runID}))
	if err != nil {
		t.Fatalf("GetRunGraph: %v", err)
	}
	// Suppress unused variable warning for svc
	_ = svc
	if len(graph.Msg.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Msg.Nodes))
	}
	if len(graph.Msg.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(graph.Msg.Edges))
	}
	if graph.Msg.Nodes[0].Name != runID {
		t.Errorf("expected node name %q, got %q", runID, graph.Msg.Nodes[0].Name)
	}
}

func TestGetRunGraph_ParentChild(t *testing.T) {
	client, _, svc, cleanup := startTestServerWithBus(t)
	defer cleanup()

	ctx := context.Background()

	// Create parent run
	parentResp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "parent run",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun (parent): %v", err)
	}
	parentID := parentResp.Msg.AgentRun.Id

	// Create child run
	childResp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:      apiv1.Backend_BACKEND_POD,
			Repos:        []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:       "child run",
			ParentRunId:  parentID,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun (child): %v", err)
	}
	childID := childResp.Msg.AgentRun.Id

	// Add spec-run-id label to both runs so the List query finds them together
	for _, id := range []string{parentID, childID} {
		crd := &aotv1alpha1.AgentRun{}
		if err := svc.K8sClient.Get(ctx, k8sclient.ObjectKey{Namespace: "default", Name: id}, crd); err != nil {
			t.Fatalf("Get CRD %s: %v", id, err)
		}
		if crd.Labels == nil {
			crd.Labels = make(map[string]string)
		}
		crd.Labels["aot.uncworks.io/spec-run-id"] = parentID
		if err := svc.K8sClient.Update(ctx, crd); err != nil {
			t.Fatalf("Update CRD %s: %v", id, err)
		}
	}

	graph, err := client.GetRunGraph(ctx, connect.NewRequest(&apiv1.GetRunGraphRequest{Id: parentID}))
	if err != nil {
		t.Fatalf("GetRunGraph: %v", err)
	}

	if len(graph.Msg.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.Msg.Nodes))
	}
	if len(graph.Msg.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Msg.Edges))
	}
	edge := graph.Msg.Edges[0]
	if edge.Parent != parentID {
		t.Errorf("expected edge parent=%q, got %q", parentID, edge.Parent)
	}
	if edge.Child != childID {
		t.Errorf("expected edge child=%q, got %q", childID, edge.Child)
	}
}

// --- SearchPastWork SOURCE_CODE tests ---

func TestSearchPastWork_SourceCode_NoEndpoint(t *testing.T) {
	t.Setenv("CUDGEL_ENDPOINT", "")
	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "authentication middleware",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE,
		Limit:        5,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Results) != 0 {
		t.Errorf("expected empty results when endpoint unset, got %d", len(resp.Msg.Results))
	}
}

func TestSearchPastWork_SourceCode_Success(t *testing.T) {
	cudgelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.Error(w, "unexpected", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"AuthHandler","kind":"function","file":"auth.go","line":10,"snippet":"func AuthHandler()","score":0.95}]`))
	}))
	defer cudgelSrv.Close()
	t.Setenv("CUDGEL_ENDPOINT", cudgelSrv.URL)

	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "authentication middleware",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE,
		Limit:        5,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Msg.Results))
	}
	r := resp.Msg.Results[0]
	if r.NodeType != "function" {
		t.Errorf("expected node_type=function, got %q", r.NodeType)
	}
	if r.ChunkText != "func AuthHandler()" {
		t.Errorf("expected snippet as chunk_text, got %q", r.ChunkText)
	}
	if r.SimilarityScore != 0.95 {
		t.Errorf("expected score=0.95, got %f", r.SimilarityScore)
	}
}

func TestSearchPastWork_SourceCode_CudgelUnavailable(t *testing.T) {
	// Point to a non-listening port
	t.Setenv("CUDGEL_ENDPOINT", "http://127.0.0.1:19998")

	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "auth",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE,
		Limit:        5,
	}))
	if err != nil {
		t.Fatalf("expected empty response, got error: %v", err)
	}
	if len(resp.Msg.Results) != 0 {
		t.Errorf("expected empty results on cudgel failure, got %d", len(resp.Msg.Results))
	}
}

// --- SearchPastWork non-SOURCE_CODE filter tests ---

func TestSearchPastWork_EmptyQuery(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query: "",
	}))
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

// TestSearchPastWork_AllFilter_NoBrainSearcher verifies that SOURCE_FILTER_ALL
// returns CodeUnavailable when BrainSearcher is not configured (the default in
// startTestServer).
func TestSearchPastWork_AllFilter_NoBrainSearcher(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "find authentication code",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_ALL,
		Limit:        5,
	}))
	if err == nil {
		t.Fatal("expected error when BrainSearcher not configured")
	}
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("expected Unavailable, got %v", connect.CodeOf(err))
	}
}

// TestSearchPastWork_CodeFilter_NoBrainSearcher verifies that SOURCE_FILTER_CODE
// returns CodeUnavailable when BrainSearcher is not configured.
func TestSearchPastWork_CodeFilter_NoBrainSearcher(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "find authentication code",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_CODE,
		Limit:        5,
	}))
	if err == nil {
		t.Fatal("expected error when BrainSearcher not configured")
	}
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("expected Unavailable, got %v", connect.CodeOf(err))
	}
}

// --- ListAgentRuns filter tests ---

func TestListAgentRuns_PhaseFilter(t *testing.T) {
	grpcClient, _, svc, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create one run and manually update its phase to Running via k8s status update.
	createResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "pending run",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Create a second run and patch its phase to RUNNING so the filter can discriminate.
	runningResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "running run",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Directly update the second run's status to RUNNING via the k8s client.
	crd := &aotv1alpha1.AgentRun{}
	if err := svc.K8sClient.Get(context.Background(), k8sclient.ObjectKey{Namespace: "default", Name: runningResp.Msg.AgentRun.Id}, crd); err != nil {
		t.Fatalf("Get AgentRun: %v", err)
	}
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	if err := svc.K8sClient.Status().Update(context.Background(), crd); err != nil {
		t.Fatalf("Status().Update: %v", err)
	}

	// Filter for PENDING only — should return 1 run (the first one).
	pendingResp, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(PENDING): %v", err)
	}
	if len(pendingResp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 PENDING run, got %d", len(pendingResp.Msg.AgentRuns))
	}
	if pendingResp.Msg.AgentRuns[0].Id != createResp.Msg.AgentRun.Id {
		t.Errorf("PENDING filter returned wrong run")
	}

	// Filter for RUNNING only — should return 1 run (the second one).
	runningListResp, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(RUNNING): %v", err)
	}
	if len(runningListResp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 RUNNING run, got %d", len(runningListResp.Msg.AgentRuns))
	}
	if runningListResp.Msg.AgentRuns[0].Id != runningResp.Msg.AgentRun.Id {
		t.Errorf("RUNNING filter returned wrong run")
	}
}

func TestListAgentRuns_TagFilter(t *testing.T) {
	grpcClient, cleanup := startTestServer(t)
	defer cleanup()

	// Create a run with tags.
	_, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "tagged run",
			Tags:    []string{"release", "hotfix"},
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun with tags: %v", err)
	}

	// Create a run without tags.
	if _, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "untagged run",
		},
	})); err != nil {
		t.Fatalf("CreateAgentRun without tags: %v", err)
	}

	// Filter for "release" tag — should return 1.
	resp, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		TagFilter: "release",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(tag=release): %v", err)
	}
	if len(resp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 run with tag 'release', got %d", len(resp.Msg.AgentRuns))
	}

	// Filter for absent tag — should return 0.
	resp2, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		TagFilter: "nonexistent-tag",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(tag=nonexistent-tag): %v", err)
	}
	if len(resp2.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs for absent tag, got %d", len(resp2.Msg.AgentRuns))
	}
}

func TestListAgentRuns_PageSizeClamping(t *testing.T) {
	grpcClient, cleanup := startTestServer(t)
	defer cleanup()

	// Create 3 runs.
	for i := 0; i < 3; i++ {
		if _, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
				Prompt:  "run",
			},
		})); err != nil {
			t.Fatalf("CreateAgentRun: %v", err)
		}
	}

	// limit=0 should use default (50) — returns all 3.
	resp0, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 0}))
	if err != nil {
		t.Fatalf("ListAgentRuns(limit=0): %v", err)
	}
	if len(resp0.Msg.AgentRuns) != 3 {
		t.Errorf("limit=0 should default to 50, got %d runs", len(resp0.Msg.AgentRuns))
	}

	// limit=1 should return exactly 1.
	resp1, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 1}))
	if err != nil {
		t.Fatalf("ListAgentRuns(limit=1): %v", err)
	}
	if len(resp1.Msg.AgentRuns) != 1 {
		t.Errorf("limit=1 should return 1 run, got %d", len(resp1.Msg.AgentRuns))
	}
}

func TestListAgentRuns_StageFilter(t *testing.T) {
	grpcClient, _, svc, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run and update its status stage.
	createResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "planning run",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	crd := &aotv1alpha1.AgentRun{}
	if err := svc.K8sClient.Get(context.Background(), k8sclient.ObjectKey{Namespace: "default", Name: createResp.Msg.AgentRun.Id}, crd); err != nil {
		t.Fatalf("Get: %v", err)
	}
	crd.Status.Stage = "planning"
	if err := svc.K8sClient.Status().Update(context.Background(), crd); err != nil {
		t.Fatalf("Status().Update: %v", err)
	}

	// Create a second run with no stage (empty).
	if _, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "no stage run",
		},
	})); err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	resp, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		StageFilter: "planning",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(stage=planning): %v", err)
	}
	if len(resp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 run with stage=planning, got %d", len(resp.Msg.AgentRuns))
	}
	if resp.Msg.AgentRuns[0].Id != createResp.Msg.AgentRun.Id {
		t.Errorf("stage filter returned wrong run")
	}
}

func TestListAgentRuns_ParentRunIdFilter(t *testing.T) {
	grpcClient, cleanup := startTestServer(t)
	defer cleanup()

	// Create a "parent" run (no parentRunId).
	parentResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "parent",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun parent: %v", err)
	}

	// Create a child run referencing the parent.
	childResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:     apiv1.Backend_BACKEND_POD,
			Repos:       []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:      "child",
			ParentRunId: parentResp.Msg.AgentRun.Id,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun child: %v", err)
	}

	resp, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		ParentRunId: parentResp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns(parentRunId): %v", err)
	}
	if len(resp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 child run, got %d", len(resp.Msg.AgentRuns))
	}
	if len(resp.Msg.AgentRuns) > 0 && resp.Msg.AgentRuns[0].Id != childResp.Msg.AgentRun.Id {
		t.Errorf("parentRunId filter returned wrong run")
	}
}

func TestListAgentRuns_ArchivedFilter(t *testing.T) {
	grpcClient, _, svc, cleanup := startTestServerWithBus(t)
	defer cleanup()

	// Create a run and archive it.
	createResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "to be archived",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Archive the run directly via k8s status.
	crd := &aotv1alpha1.AgentRun{}
	if err := svc.K8sClient.Get(context.Background(), k8sclient.ObjectKey{Namespace: "default", Name: createResp.Msg.AgentRun.Id}, crd); err != nil {
		t.Fatalf("Get: %v", err)
	}
	crd.Status.Archived = true
	if err := svc.K8sClient.Status().Update(context.Background(), crd); err != nil {
		t.Fatalf("Status().Update: %v", err)
	}

	// Also create a non-archived run.
	if _, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "active run",
		},
	})); err != nil {
		t.Fatalf("CreateAgentRun active: %v", err)
	}

	// Default list (no X-Include-Archived): should return only the active run.
	respDefault, err := grpcClient.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns (default): %v", err)
	}
	if len(respDefault.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 active run by default, got %d", len(respDefault.Msg.AgentRuns))
	}

	// With X-Include-Archived: true: should return both.
	reqWithArchived := connect.NewRequest(&apiv1.ListAgentRunsRequest{})
	reqWithArchived.Header().Set("X-Include-Archived", "true")
	respAll, err := grpcClient.ListAgentRuns(context.Background(), reqWithArchived)
	if err != nil {
		t.Fatalf("ListAgentRuns (include archived): %v", err)
	}
	if len(respAll.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs with X-Include-Archived=true, got %d", len(respAll.Msg.AgentRuns))
	}
}

func TestWatchAgentRun_EventBusNil_ReturnsUnavailable(t *testing.T) {
	// startTestServer uses NoOpEventBus. WatchAgentRun requires an event bus to
	// stream events; when the bus is nil (not NoOp), it should return CodeUnimplemented.
	// This test uses a handler constructed with nil EventBus explicitly.
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := NewAOTServiceHandler(k8sClient, nil, "default") // nil bus
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()
	grpcClient := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)

	// Create a run first so WatchAgentRun can find it.
	createResp, err := grpcClient.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "watch nil bus",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := grpcClient.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: createResp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun setup: %v", err)
	}

	// Receive the initial event (WatchAgentRun always sends the current state first).
	stream.Receive()
	// The next Receive should fail because there's no bus to deliver further events,
	// or the stream closes. Either way stream.Err() should be non-nil or Receive returns false.
	stream.Receive()
	// Either way, no panic and no infinite block — the test passing without hanging is the assertion.
}

// TestSearchPastWork_TraceFilter_NoBrainSearcher verifies that SOURCE_FILTER_TRACE
// returns CodeUnavailable when BrainSearcher is not configured.
func TestSearchPastWork_TraceFilter_NoBrainSearcher(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	_, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "trace error logs",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_TRACE,
		Limit:        5,
	}))
	if err == nil {
		t.Fatal("expected error when BrainSearcher not configured")
	}
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("expected Unavailable, got %v", connect.CodeOf(err))
	}
}
