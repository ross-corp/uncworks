// test/layer2/error_retry_test.go — Layer 2 tests for LLM error and retry scenarios.
// Tests that the litellm client surfaces errors from a 503 response, and that
// a run transitions to Failed when the LLM is permanently unavailable.
package layer2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	internallitellm "github.com/uncworks/aot/internal/litellm"
	"github.com/uncworks/aot/test/stubs"
)

// TestLiteLLMClient_Returns503Error verifies that the litellm admin client
// returns a descriptive error when the server responds with 503.
func TestLiteLLMClient_Returns503Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := internallitellm.NewClient(srv.URL, "test-key")
	_, err := c.GenerateKey(context.Background(), internallitellm.GenerateKeyRequest{
		KeyAlias: "test-alias",
	})
	require.Error(t, err, "expected error when server returns 503")
	assert.Contains(t, err.Error(), "503",
		"error message should mention the 503 status")
}

// TestLiteLLMClient_PermanentUnavailable_MultipleAttempts verifies that
// consecutive calls to an unavailable server all return errors (no silent
// swallowing of failures).
func TestLiteLLMClient_PermanentUnavailable_MultipleAttempts(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		http.Error(w, `{"error":"backend offline"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := internallitellm.NewClient(srv.URL, "test-key")

	for i := 0; i < 3; i++ {
		_, err := c.GenerateKey(context.Background(), internallitellm.GenerateKeyRequest{
			KeyAlias: "test",
		})
		require.Error(t, err, "attempt %d: expected error", i+1)
	}
	assert.Equal(t, 3, callCount, "client should hit the server on each attempt")
}

// TestAgentRunLifecycle_FailsWhenLLMUnavailable verifies that a run can
// be transitioned to Failed when the LLM stub simulates a permanent outage.
// This tests the pipeline boundary: after create, a controller-simulated
// failure (e.g. because LiteLLM returns 503) marks the run failed.
func TestAgentRunLifecycle_FailsWhenLLMUnavailable(t *testing.T) {
	// The LiteLLM stub is not used by the ConnectRPC handler itself — it would
	// be used by the Temporal activity worker. Here we test the observable
	// outcome: the run's status phase is Failed and the message references the
	// outage reason.
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	// Create the run.
	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Task needing LLM",
		},
	}))
	require.NoError(t, err)

	runID := createResp.Msg.AgentRun.Id
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING, createResp.Msg.AgentRun.Status.Phase)

	// Simulate the Temporal activity failing because LiteLLM returned 503.
	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
	crd.Status.Message = "LLM provisioning failed: key/generate returned 503: service unavailable"
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED, getResp.Msg.Status.Phase)
	assert.Contains(t, getResp.Msg.Status.Message, "503")
}

// TestLiteLLMStub_Returns503_ExplicitHandler verifies that a custom stub
// server can be configured to return 503, which exercises the error path
// that a real activity worker would encounter.
func TestLiteLLMStub_Returns503_ExplicitHandler(t *testing.T) {
	// Build a stub that always responds 503.
	unavailableSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"model overloaded"}`))
	}))
	t.Cleanup(unavailableSrv.Close)

	c := internallitellm.NewClient(unavailableSrv.URL, "sk-test")

	_, err := c.ListModels(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

// TestLiteLLMStub_Sequence_503ThenSuccess verifies the pattern of a stub
// that simulates transient failure followed by recovery.
// This models: first call fails (503-like body), second call succeeds.
func TestLiteLLMStub_Sequence_503ThenSuccess(t *testing.T) {
	callN := 0
	transitionalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callN++
		if callN == 1 {
			http.Error(w, `{"error":"temporarily unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		// Second call: success
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"sk-ok-key","key_name":"test"}`))
	}))
	t.Cleanup(transitionalSrv.Close)

	c := internallitellm.NewClient(transitionalSrv.URL, "sk-master")

	// First attempt: expect failure.
	_, err := c.GenerateKey(context.Background(), internallitellm.GenerateKeyRequest{KeyAlias: "a"})
	require.Error(t, err, "first call should fail with 503")

	// Second attempt: expect success.
	resp, err := c.GenerateKey(context.Background(), internallitellm.GenerateKeyRequest{KeyAlias: "a"})
	require.NoError(t, err, "second call should succeed")
	assert.Equal(t, "sk-ok-key", resp.Key)

	// Smoke-test the stubs package DefaultCompletion shape is reusable here.
	_ = stubs.DefaultCompletion("test response content")
}
