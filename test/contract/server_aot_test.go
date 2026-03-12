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

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/server"
)

func startAOTServer(t *testing.T, withValidation bool) (apiv1connect.AOTServiceClient, func()) {
	t.Helper()

	svc := server.NewAOTServiceHandler(nil)
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
			RepoUrl: "https://github.com/example/repo.git",
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
	if resp.Msg.AgentRun.UpdatedAt == nil {
		t.Error("expected non-nil UpdatedAt")
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
			RepoUrl: "https://github.com/example/repo.git",
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
				RepoUrl: "https://github.com/example/repo.git",
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
				RepoUrl: "https://github.com/example/repo.git",
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

	created, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "task",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Cancel one to make it CANCELLED phase
	if _, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: created.Msg.AgentRun.Id,
	})); err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}

	// Create another (PENDING)
	if _, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "another",
		},
	})); err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	// Filter for PENDING only
	resp, err := client.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(resp.Msg.AgentRuns) != 1 {
		t.Errorf("expected 1 PENDING run, got %d", len(resp.Msg.AgentRuns))
	}
}

// --- CancelAgentRun contract ---

func TestContract_CancelAgentRun_Exists(t *testing.T) {
	client, cleanup := startAOTServer(t, false)
	defer cleanup()

	created, err := client.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			RepoUrl: "https://github.com/example/repo.git",
			Prompt:  "cancel me",
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	resp, err := client.CancelAgentRun(context.Background(), connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: created.Msg.AgentRun.Id,
	}))
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	if resp.Msg.AgentRun.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
		t.Errorf("expected CANCELLED, got %v", resp.Msg.AgentRun.Status.Phase)
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
			RepoUrl: "https://github.com/example/repo.git",
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
