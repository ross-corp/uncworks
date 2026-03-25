package temporal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.temporal.io/sdk/testsuite"

	aottemporal "github.com/uncworks/aot/internal/temporal"

	"github.com/uncworks/aot/internal/litellm"
)

func TestProvisionLLMKey_Success(t *testing.T) {
	// Mock server serves both /v1/models (dynamic discovery) and /key/generate.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(litellm.ListModelsResponse{
				Object: "list",
				Data: []litellm.ModelInfo{
					{ID: "default"},
					{ID: "default-cloud"},
				},
			})
			return
		}
		var req litellm.GenerateKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.KeyAlias != "aot-default-test-run" {
			t.Errorf("unexpected alias: %s", req.KeyAlias)
		}
		if len(req.Models) != 2 {
			t.Errorf("expected 2 models for default tier, got %d", len(req.Models))
		}

		_ = json.NewEncoder(w).Encode(litellm.GenerateKeyResponse{Key: "sk-provisioned-123"})
	}))
	defer server.Close()

	activities := &aottemporal.Activities{
		LiteLLMClient: litellm.NewClient(server.URL, "master-key"),
	}

	out, err := activities.ProvisionLLMKey(context.Background(), aottemporal.ProvisionLLMKeyInput{
		AgentRunName: "test-run",
		Namespace:    "default",
		ModelTier:    "default",
		MaxBudget:    2.0,
	})
	if err != nil {
		t.Fatalf("ProvisionLLMKey: %v", err)
	}
	if out.Key != "sk-provisioned-123" {
		t.Errorf("unexpected key: %s", out.Key)
	}
}

func TestProvisionLLMKey_PremiumTier(t *testing.T) {
	// Mock server serves /v1/models with premium-tier model set.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(litellm.ListModelsResponse{
				Object: "list",
				Data: []litellm.ModelInfo{
					{ID: "default"},
					{ID: "default-cloud"},
					{ID: "premium"},
				},
			})
			return
		}
		var req litellm.GenerateKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if len(req.Models) != 3 {
			t.Errorf("expected 3 models for premium tier, got %d: %v", len(req.Models), req.Models)
		}

		_ = json.NewEncoder(w).Encode(litellm.GenerateKeyResponse{Key: "sk-premium"})
	}))
	defer server.Close()

	activities := &aottemporal.Activities{
		LiteLLMClient: litellm.NewClient(server.URL, "master-key"),
	}

	out, err := activities.ProvisionLLMKey(context.Background(), aottemporal.ProvisionLLMKeyInput{
		AgentRunName: "premium-run",
		Namespace:    "default",
		ModelTier:    "premium",
	})
	if err != nil {
		t.Fatalf("ProvisionLLMKey: %v", err)
	}
	if out.Key != "sk-premium" {
		t.Errorf("unexpected key: %s", out.Key)
	}
}

func TestProvisionLLMKey_NoClient(t *testing.T) {
	activities := &aottemporal.Activities{}

	out, err := activities.ProvisionLLMKey(context.Background(), aottemporal.ProvisionLLMKeyInput{
		AgentRunName: "test",
		Namespace:    "default",
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if out.Key != "" {
		t.Errorf("expected empty key when no client, got: %s", out.Key)
	}
}

func TestRevokeLLMKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(litellm.DeleteKeyResponse{DeletedKeys: []string{"sk-to-revoke"}})
	}))
	defer server.Close()

	activities := &aottemporal.Activities{
		LiteLLMClient: litellm.NewClient(server.URL, "master-key"),
	}

	err := activities.RevokeLLMKey(context.Background(), aottemporal.RevokeLLMKeyInput{Key: "sk-to-revoke"})
	if err != nil {
		t.Fatalf("RevokeLLMKey: %v", err)
	}
}

func TestRevokeLLMKey_NoClient(t *testing.T) {
	activities := &aottemporal.Activities{}
	err := activities.RevokeLLMKey(context.Background(), aottemporal.RevokeLLMKeyInput{Key: "sk-any"})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
}

func TestRevokeLLMKey_EmptyKey(t *testing.T) {
	activities := &aottemporal.Activities{
		LiteLLMClient: litellm.NewClient("http://unused", "key"),
	}
	err := activities.RevokeLLMKey(context.Background(), aottemporal.RevokeLLMKeyInput{Key: ""})
	if err != nil {
		t.Fatalf("expected no error for empty key: %v", err)
	}
}

// TestProvisionLLMKey_FallbackOnModelError verifies that when /v1/models returns
// a server error the provisioning falls back to the static modelsForTier list and
// still generates a key successfully.
// It uses testsuite.TestActivityEnvironment so that activity.GetLogger works inside
// the fallback code path.
func TestProvisionLLMKey_FallbackOnModelError(t *testing.T) {
	keyCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			// Simulate server error — forces fallback to static list.
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// /key/generate
		keyCalled = true
		var req litellm.GenerateKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// With the default tier the fallback list has at least 2 models.
		if len(req.Models) < 1 {
			t.Errorf("expected at least 1 model in fallback list, got %d", len(req.Models))
		}

		_ = json.NewEncoder(w).Encode(litellm.GenerateKeyResponse{Key: "sk-fallback"})
	}))
	defer server.Close()

	activities := &aottemporal.Activities{
		LiteLLMClient: litellm.NewClient(server.URL, "master-key"),
	}

	// Use the Temporal test activity environment so that activity.GetLogger(ctx)
	// inside the fallback path doesn't panic.
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	val, err := env.ExecuteActivity(activities.ProvisionLLMKey, aottemporal.ProvisionLLMKeyInput{
		AgentRunName: "fallback-run",
		Namespace:    "default",
		ModelTier:    "default",
	})
	if err != nil {
		t.Fatalf("ProvisionLLMKey: %v", err)
	}
	var out aottemporal.ProvisionLLMKeyOutput
	if err := val.Get(&out); err != nil {
		t.Fatalf("get output: %v", err)
	}
	if out.Key != "sk-fallback" {
		t.Errorf("unexpected key: %s", out.Key)
	}
	if !keyCalled {
		t.Error("key/generate was never called")
	}
}
