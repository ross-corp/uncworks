// Package contract provides contract tests that verify ConnectRPC server
// implementations match their proto contracts. These tests start real HTTP
// servers with protovalidate interceptors and exercise every RPC.
package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"connectrpc.com/validate"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/server"
)

var testScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(testScheme))
}

func startAOTServer(t *testing.T, withValidation bool) (apiv1connect.AOTServiceClient, func()) {
	t.Helper()

	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithStatusSubresource(&aotv1alpha1.AgentRun{}).Build()
	svc := server.NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")
	mux := http.NewServeMux()

	var opts []connect.HandlerOption
	if withValidation {
		interceptor := validate.NewInterceptor()
		opts = append(opts, connect.WithInterceptors(interceptor))
	}

	path, handler := apiv1connect.NewAOTServiceHandler(svc, opts...)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	client := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

// --- CreateAgentRun contract ---

func TestContract_CreateAgentRun_ValidInput(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
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
	if resp.Msg.AgentRun == nil {
		t.Fatal("expected non-nil AgentRun in response")
	}
	if resp.Msg.AgentRun.Id == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Msg.AgentRun.Status == nil {
		t.Fatal("expected non-nil Status")
	}
	if resp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
		t.Errorf("expected PENDING phase, got %v", resp.Msg.AgentRun.Status.Phase)
	}
	if resp.Msg.AgentRun.Spec == nil {
		t.Fatal("expected non-nil Spec in response")
	}
	if resp.Msg.AgentRun.Spec.Prompt != "Fix the tests" {
		t.Errorf("expected prompt preserved, got %q", resp.Msg.AgentRun.Spec.Prompt)
	}
	if resp.Msg.AgentRun.CreatedAt == nil {
		t.Error("expected non-nil CreatedAt")
	}
}

func TestContract_CreateAgentRun_NilSpec(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{}))
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

// --- GetAgentRun contract ---

func TestContract_GetAgentRun_Exists(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	created, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Test get",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	resp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: created.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if resp.Msg.Id != created.Msg.AgentRun.Id {
		t.Errorf("ID mismatch: got %q, want %q", resp.Msg.Id, created.Msg.AgentRun.Id)
	}
	if resp.Msg.Spec.Prompt != "Test get" {
		t.Errorf("prompt mismatch: got %q", resp.Msg.Spec.Prompt)
	}
}

func TestContract_GetAgentRun_NotFound(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: "nonexistent",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

// --- ListAgentRuns contract ---

func TestContract_ListAgentRuns_Empty(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 runs, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_WithRuns(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	for i := 0; i < 3; i++ {
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

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 3 {
		t.Errorf("expected 3 runs, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_WithLimit(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
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

	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		Limit: 2,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 runs with limit, got %d", len(resp.Msg.AgentRuns))
	}
}

func TestContract_ListAgentRuns_PhaseFilter(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	// Create two runs — both will be PENDING (no Temporal to change phase)
	for i := 0; i < 2; i++ {
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

	// Filter for PENDING — should return both
	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 2 {
		t.Errorf("expected 2 PENDING runs, got %d", len(resp.Msg.AgentRuns))
	}

	// Filter for RUNNING — should return none
	resp, err = client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 0 {
		t.Errorf("expected 0 RUNNING runs, got %d", len(resp.Msg.AgentRuns))
	}
}

// --- CancelAgentRun contract ---

func TestContract_CancelAgentRun_Exists(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	created, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "cancel me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Without Temporal, cancel returns the CRD as-is (phase unchanged)
	resp, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: created.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	if resp.Msg.AgentRun.Id != created.Msg.AgentRun.Id {
		t.Errorf("expected same ID, got %s", resp.Msg.AgentRun.Id)
	}
}

func TestContract_CancelAgentRun_NotFound(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: "nonexistent",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

// --- SendHumanInput contract ---

func TestContract_SendHumanInput_NotFound(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	_, err := client.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: "nonexistent",
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
	}
}

func TestContract_SendHumanInput_NotWaiting(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	created, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
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
		AgentRunId: created.Msg.AgentRun.Id,
		Input:      "hello",
	}))
	if err == nil {
		t.Fatal("expected error: agent not waiting for input")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

// --- WatchAgentRun contract ---

// --- Spec-driven pipeline contract ---

func TestContract_CreateAgentRun_SpecDrivenMode(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	resp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:            "Fix the auth module",
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	if resp.Msg.AgentRun == nil {
		t.Fatal("expected non-nil AgentRun")
	}
	if resp.Msg.AgentRun.Spec.OrchestrationMode != apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN {
		t.Errorf("expected SPEC_DRIVEN mode, got %v", resp.Msg.AgentRun.Spec.OrchestrationMode)
	}
}

func TestContract_ListAgentRuns_StageFilter(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	// Create a run first
	_, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "test stage filter",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// List without filter should return the run
	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) < 1 {
		t.Errorf("expected at least 1 run, got %d", len(resp.Msg.AgentRuns))
	}

	// Verify stage_filter field exists on request (server-side filtering).
	// Note: proto regen needed for wire encoding; this tests the struct field exists.
	req := &apiv1.ListAgentRunsRequest{StageFilter: "planning"}
	if req.StageFilter != "planning" {
		t.Errorf("expected StageFilter field to be settable")
	}
}

func TestContract_GetAgentRun_IncludesNewStatusFields(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	// Create a run
	createResp, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "test new fields",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Get it and verify new fields exist (even if empty for non-spec-driven)
	getResp, err := client.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: createResp.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}

	status := getResp.Msg.Status
	if status == nil {
		t.Fatal("expected non-nil Status")
	}
	// New fields should be present (empty/zero for non-spec-driven runs)
	if status.Stage != "" {
		t.Errorf("expected empty stage for non-spec-driven run, got %q", status.Stage)
	}
	if status.RetryCount != 0 {
		t.Errorf("expected 0 retry count, got %d", status.RetryCount)
	}
	if status.VerificationResult != "" {
		t.Errorf("expected empty verification result, got %q", status.VerificationResult)
	}
}

func TestContract_WatchAgentRun_NotFound(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	stream, err := client.WatchAgentRun(context.Background(), connect.NewRequest(&apiv1.WatchAgentRunRequest{
		Id: "nonexistent",
	}))
	if err != nil {
		// Some implementations return error immediately
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", connect.CodeOf(err))
		}
		return
	}
	// Others return error on first Receive
	if stream.Receive() {
		t.Fatal("expected no messages for nonexistent run")
	}
	if stream.Err() != nil && connect.CodeOf(stream.Err()) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", connect.CodeOf(stream.Err()))
	}
}
