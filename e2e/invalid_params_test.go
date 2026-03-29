//go:build e2e

// e2e/invalid_params_test.go — tests that CreateAgentRun rejects invalid inputs.
package e2e

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestE2E_API_InvalidParams_MissingBackend verifies that CreateAgentRun returns
// an error when Backend is unspecified (BACKEND_UNSPECIFIED).
func TestE2E_API_InvalidParams_MissingBackend(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	_, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			// Backend intentionally omitted (zero value = BACKEND_UNSPECIFIED).
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     "should be rejected",
			TtlSeconds: 60,
		},
	}))
	if err == nil {
		t.Fatal("expected an error for BACKEND_UNSPECIFIED, got nil")
	}
	code := connect.CodeOf(err)
	// protovalidate rejects the request at the API layer — expect InvalidArgument or
	// a server-side validation error; anything except success is acceptable.
	t.Logf("Got expected error for missing backend: code=%v err=%v", code, err)
	if code != connect.CodeInvalidArgument && code != connect.CodeInternal {
		t.Errorf("expected InvalidArgument or Internal, got %v", code)
	}
}

// TestE2E_API_InvalidParams_EmptyRepos verifies that CreateAgentRun returns an
// error when the repos list is empty (protovalidate min_items = 1).
func TestE2E_API_InvalidParams_EmptyRepos(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	_, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{}, // empty — violates min_items=1
			Prompt:     "should be rejected",
			TtlSeconds: 60,
		},
	}))
	if err == nil {
		t.Fatal("expected an error for empty repos, got nil")
	}
	code := connect.CodeOf(err)
	t.Logf("Got expected error for empty repos: code=%v err=%v", code, err)
	if code != connect.CodeInvalidArgument && code != connect.CodeInternal {
		t.Errorf("expected InvalidArgument or Internal, got %v", code)
	}
}

// TestE2E_API_InvalidParams_EmptyPromptAndSpec verifies that CreateAgentRun
// either rejects or accepts a run with no prompt and no spec_content.
// The system may accept it and let the agent fail, or reject at validation.
// This test documents the observed behavior.
func TestE2E_API_InvalidParams_EmptyPromptAndSpec(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	resp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     "",  // no prompt
			TtlSeconds: 60,
		},
	}))
	if err != nil {
		// Server may enforce non-empty prompt — that is also acceptable.
		t.Logf("CreateAgentRun rejected empty prompt: code=%v err=%v", connect.CodeOf(err), err)
		return
	}
	// If accepted, we at least have an ID.
	runID := resp.Msg.AgentRun.Id
	if runID == "" {
		t.Fatal("expected non-empty run ID even for empty-prompt run")
	}
	t.Logf("CreateAgentRun accepted empty prompt; run created with ID: %s", runID)
}

// TestE2E_API_InvalidParams_NegativeTTL verifies that a negative TTL is rejected
// (protovalidate gte = 0).
func TestE2E_API_InvalidParams_NegativeTTL(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	_, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     "negative ttl test",
			TtlSeconds: -1, // violates gte=0
		},
	}))
	if err == nil {
		t.Fatal("expected an error for negative TTL, got nil")
	}
	code := connect.CodeOf(err)
	t.Logf("Got expected error for negative TTL: code=%v err=%v", code, err)
	if code != connect.CodeInvalidArgument && code != connect.CodeInternal {
		t.Errorf("expected InvalidArgument or Internal, got %v", code)
	}
}

// TestE2E_API_GetNonExistent verifies that GetAgentRun returns NotFound (or an
// equivalent error) for a run ID that does not exist.
func TestE2E_API_GetNonExistent(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	_, err := client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: "non-existent-run-id-xyz-123",
	}))
	if err == nil {
		t.Fatal("expected an error for non-existent run ID, got nil")
	}
	code := connect.CodeOf(err)
	t.Logf("Got expected error for non-existent ID: code=%v err=%v", code, err)
	if code != connect.CodeNotFound && code != connect.CodeInternal {
		t.Errorf("expected NotFound or Internal, got %v", code)
	}
}

// TestE2E_API_CancelNonExistent verifies that CancelAgentRun returns an error
// for a run ID that does not exist.
func TestE2E_API_CancelNonExistent(t *testing.T) {
	client := getAPIClient(t)
	ctx := context.Background()

	_, err := client.CancelAgentRun(ctx, connect.NewRequest(&apiv1.CancelAgentRunRequest{
		Id: "non-existent-cancel-run-xyz-456",
	}))
	if err == nil {
		t.Fatal("expected an error for cancelling non-existent run, got nil")
	}
	t.Logf("Got expected error for cancel non-existent: code=%v err=%v", connect.CodeOf(err), err)
}
