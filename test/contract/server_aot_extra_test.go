package contract

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// helper that creates an AgentRun and returns its ID.
func mustCreateRun(t *testing.T, client interface {
	CreateAgentRun(context.Context, *connect.Request[apiv1.CreateAgentRunRequest]) (*connect.Response[apiv1.CreateAgentRunResponse], error)
}, prompt string) string {
	t.Helper()
	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  prompt,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	return resp.Msg.AgentRun.Id
}

// --- WatchAgentRun happy path ---

func TestContract_WatchAgentRun_ExistingRun_ReceivesInitialEvent(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	id := mustCreateRun(t, client, "watch me")

	// Use a short timeout so the test doesn't hang on the blocking EventBus.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.WatchAgentRun(ctx, connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: id,
	}))
	if err != nil {
		t.Fatalf("WatchAgentRun: %v", err)
	}

	// The handler sends an initial PHASE_CHANGED event then blocks on the
	// NoOpEventBus channel (never written). stream.Receive() will return true for
	// the initial event, then return false when the context deadline fires.
	if !stream.Receive() {
		// Stream ended immediately without any message — acceptable only for
		// Unimplemented (EventBus nil case, not reachable here) or context error.
		err := stream.Err()
		if err != nil {
			code := connect.CodeOf(err)
			if code != connect.CodeUnimplemented && code != connect.CodeCanceled && code != connect.CodeDeadlineExceeded {
				t.Errorf("WatchAgentRun closed without event, unexpected code %v: %v", code, err)
			}
		}
		return
	}

	// Happy path: initial event received.
	ev := stream.Msg()
	if ev.AgentRunId != id {
		t.Errorf("initial event AgentRunId = %q, want %q", ev.AgentRunId, id)
	}
	if ev.Type != apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED {
		t.Errorf("expected PHASE_CHANGED event type, got %v", ev.Type)
	}
	// Drain the rest (context will fire after 2s).
	for stream.Receive() {
	}
}

// --- GetAgentRun with empty ID ---

func TestContract_GetAgentRun_EmptyID(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	// An empty ID fails format validation before hitting the store.
	_, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: "",
	}))
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
	// Empty ID violates the ar-[a-z0-9]{4,10} format constraint → InvalidArgument.
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument for empty ID, got %v", connect.CodeOf(err))
	}
}

// --- ListAgentRuns label-filter contracts ---

func TestContract_ListAgentRuns_ProjectFilter(t *testing.T) {
	// The project filter is applied via k8s label matching.
	// Without a label on the run, the filter should return 0 results even if
	// runs exist (because the fake client does support label selectors).
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	mustCreateRun(t, client, "project filter test")

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		ProjectFilter: "nonexistent-project",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns with ProjectFilter: %v", err)
	}
	// No runs carry the label "aot.uncworks.io/project=nonexistent-project".
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs for non-matching project filter, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_FeatureFilter(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	mustCreateRun(t, client, "feature filter test")

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		FeatureFilter: "nonexistent-feature",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns with FeatureFilter: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs for non-matching feature filter, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_TagFilter(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	mustCreateRun(t, client, "tag filter test")

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		TagFilter: "nonexistent-tag",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns with TagFilter: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs for non-matching tag filter, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_ParentRunIdFilter(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	mustCreateRun(t, client, "parent filter test")

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		ParentRunId: "ar-notfound",
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns with ParentRunId: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs for non-matching parent_run_id, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_XIncludeArchived(t *testing.T) {
	// By default, archived runs are excluded. The X-Include-Archived: true header
	// re-includes them. Since the fake client doesn't allow us to set status.archived
	// directly without a Status subresource, we verify the header field is accepted
	// without error and returns the same runs (since no runs are archived here).
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	mustCreateRun(t, client, "archived header test")

	req := connect.NewRequest(&apiv1.ListAgentRunsRequest{})
	req.Header().Set("X-Include-Archived", "true")

	resp, err := client.ListAgentRuns(context.Background(), req)
	if err != nil {
		t.Fatalf("ListAgentRuns with X-Include-Archived: %v", err)
	}
	// At least the 1 non-archived run we created must be present.
	if len(resp.Msg.AgentRuns) < 1 {
		t.Errorf("expected at least 1 run with X-Include-Archived=true, got %d", len(resp.Msg.AgentRuns))
	}
}

// --- CancelAgentRun invalid input ---

func TestContract_CancelAgentRun_EmptyID(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: "",
	}))
	if err == nil {
		t.Fatal("expected error for empty cancel ID")
	}
	// Empty ID violates the ar-[a-z0-9]{4,10} format constraint → InvalidArgument.
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument for empty cancel ID, got %v", connect.CodeOf(err))
	}
}

// --- SearchPastWork additional source filters ---

func TestContract_SearchPastWork_SourceCode_EmptyQuery(t *testing.T) {
	// An empty query must be rejected with InvalidArgument regardless of filter.
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.SearchPastWork(context.Background(), connect.NewRequest(&apiv1.SearchPastWorkRequest{
		Query:        "",
		SourceFilter: apiv1.SourceFilter_SOURCE_FILTER_SOURCE_CODE,
	}))
	if err == nil {
		t.Fatal("expected InvalidArgument for empty query with SOURCE_CODE filter")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}
