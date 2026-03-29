//go:build regression

// test/regression/chain_execution_test.go — Tests that chain creation and
// execution ordering are correctly reflected through the REST API. Uses a fake
// k8s client to simulate the ChainRun step-status transitions that the
// chain-controller would produce in a real cluster.
package regression

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/test/testutil"
)

// startChainServer starts an in-process HTTP test server wired to a
// ChainHandler backed by a fake k8s client. Returns the server URL, the
// underlying fake client, and a cleanup func.
func startChainServer(t *testing.T) (string, client.Client, func()) {
	t.Helper()

	k8s := fake.NewClientBuilder().
		WithScheme(testutil.NewScheme()).
		WithStatusSubresource(&aotv1alpha1.ChainRun{}).
		Build()

	mux := http.NewServeMux()
	ch := &server.ChainHandler{K8sClient: k8s, Namespace: testutil.DefaultNamespace}
	ch.RegisterChainHandlers(mux)

	srv := httptest.NewServer(mux)
	return srv.URL, k8s, srv.Close
}

// TestChainExecution_CreateChainReturns201 verifies that creating a valid chain
// with at least one step returns 201 Created.
func TestChainExecution_CreateChainReturns201(t *testing.T) {
	baseURL, _, cleanup := startChainServer(t)
	defer cleanup()

	resp := testutil.PostJSON(t, baseURL+"/api/v1/chains", map[string]interface{}{
		"name":        "my-chain",
		"displayName": "My Chain",
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": "tmpl-a"},
		},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode,
		"creating a valid chain must return 201")

	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	spec := got["spec"].(map[string]interface{})
	require.Equal(t, "My Chain", spec["displayName"])
}

// TestChainExecution_TriggerCreatesChainRun verifies that POSTing to
// /api/v1/chains/{name}/trigger creates a ChainRun and returns its ID.
func TestChainExecution_TriggerCreatesChainRun(t *testing.T) {
	baseURL, k8s, cleanup := startChainServer(t)
	defer cleanup()

	// Create a chain first.
	cr := testutil.PostJSON(t, baseURL+"/api/v1/chains", map[string]interface{}{
		"name": "trigger-chain",
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": "tmpl-a"},
		},
	})
	cr.Body.Close()
	require.Equal(t, http.StatusCreated, cr.StatusCode)

	// Trigger the chain.
	trigResp, err := http.Post(baseURL+"/api/v1/chains/trigger-chain/trigger", "application/json", nil) //nolint:noctx
	require.NoError(t, err)
	defer trigResp.Body.Close()
	require.Equal(t, http.StatusCreated, trigResp.StatusCode,
		"triggering an existing chain must return 201")

	var trigBody map[string]interface{}
	require.NoError(t, json.NewDecoder(trigResp.Body).Decode(&trigBody))
	chainRunID, ok := trigBody["chainRunId"].(string)
	require.True(t, ok && chainRunID != "",
		"trigger response must include a non-empty chainRunId")
	require.Equal(t, "trigger-chain", trigBody["chain"])

	// Verify the ChainRun CRD was created in the fake client.
	var list aotv1alpha1.ChainRunList
	require.NoError(t, k8s.List(context.Background(), &list))
	require.Len(t, list.Items, 1, "exactly one ChainRun must be created")
	require.Equal(t, "trigger-chain", list.Items[0].Spec.ChainRef)
	require.Equal(t, "manual", list.Items[0].Spec.TriggeredBy)
}

// TestChainExecution_StepOrderReflectedInStatus verifies that when the
// controller advances ChainRun step statuses — first step-a (no deps) followed
// by step-b (depends on step-a) — the API returns steps in the correct order
// with the expected phases.
func TestChainExecution_StepOrderReflectedInStatus(t *testing.T) {
	baseURL, k8s, cleanup := startChainServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a two-step chain: step-b depends on step-a.
	cr := testutil.PostJSON(t, baseURL+"/api/v1/chains", map[string]interface{}{
		"name":        "ordered-chain",
		"displayName": "Ordered Chain",
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": "tmpl-a"},
			{"name": "step-b", "templateRef": "tmpl-b", "dependsOn": []string{"step-a"}},
		},
	})
	cr.Body.Close()
	require.Equal(t, http.StatusCreated, cr.StatusCode)

	// Trigger to produce a ChainRun.
	trigResp, err := http.Post(baseURL+"/api/v1/chains/ordered-chain/trigger", "application/json", nil) //nolint:noctx
	require.NoError(t, err)
	defer trigResp.Body.Close()
	require.Equal(t, http.StatusCreated, trigResp.StatusCode)

	var trigBody map[string]interface{}
	require.NoError(t, json.NewDecoder(trigResp.Body).Decode(&trigBody))
	chainRunID := trigBody["chainRunId"].(string)

	// Simulate controller: step-a running, step-b still pending.
	run := &aotv1alpha1.ChainRun{}
	require.NoError(t, k8s.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      chainRunID,
	}, run))

	now := metav1.Now()
	run.Status.Phase = aotv1alpha1.ChainRunPhaseRunning
	run.Status.StartedAt = &metav1.Time{Time: now.Time}
	run.Status.Steps = []aotv1alpha1.ChainRunStepStatus{
		{
			Name:      "step-a",
			Phase:     aotv1alpha1.ChainRunStepPhaseRunning,
			RunID:     "ar-step-a",
			StartedAt: &metav1.Time{Time: now.Time},
		},
		{
			Name:  "step-b",
			Phase: aotv1alpha1.ChainRunStepPhasePending,
		},
	}
	require.NoError(t, k8s.Status().Update(ctx, run),
		"simulating controller advancing step-a to running should succeed")

	// Verify the API reflects that step-a is running and step-b is pending.
	getResp, err := http.Get(baseURL + "/api/v1/chainruns/" + chainRunID) //nolint:noctx
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var chainRunBody aotv1alpha1.ChainRun
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&chainRunBody))
	require.Equal(t, aotv1alpha1.ChainRunPhaseRunning, chainRunBody.Status.Phase)
	require.Len(t, chainRunBody.Status.Steps, 2)
	require.Equal(t, "step-a", chainRunBody.Status.Steps[0].Name)
	require.Equal(t, aotv1alpha1.ChainRunStepPhaseRunning, chainRunBody.Status.Steps[0].Phase,
		"step-a must be reflected as running before step-b can start")
	require.Equal(t, "step-b", chainRunBody.Status.Steps[1].Name)
	require.Equal(t, aotv1alpha1.ChainRunStepPhasePending, chainRunBody.Status.Steps[1].Phase,
		"step-b must remain pending while its dependency step-a is still running")

	// Simulate controller: step-a succeeded, step-b now running.
	require.NoError(t, k8s.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      chainRunID,
	}, run))

	completed := metav1.Now()
	run.Status.Steps = []aotv1alpha1.ChainRunStepStatus{
		{
			Name:        "step-a",
			Phase:       aotv1alpha1.ChainRunStepPhaseSucceeded,
			RunID:       "ar-step-a",
			StartedAt:   &metav1.Time{Time: now.Time},
			CompletedAt: &metav1.Time{Time: completed.Time},
		},
		{
			Name:      "step-b",
			Phase:     aotv1alpha1.ChainRunStepPhaseRunning,
			RunID:     "ar-step-b",
			StartedAt: &metav1.Time{Time: completed.Time},
		},
	}
	require.NoError(t, k8s.Status().Update(ctx, run),
		"simulating controller advancing step-b after step-a succeeded should succeed")

	getResp2, err := http.Get(baseURL + "/api/v1/chainruns/" + chainRunID) //nolint:noctx
	require.NoError(t, err)
	defer getResp2.Body.Close()

	var chainRunBody2 aotv1alpha1.ChainRun
	require.NoError(t, json.NewDecoder(getResp2.Body).Decode(&chainRunBody2))
	require.Equal(t, aotv1alpha1.ChainRunStepPhaseSucceeded, chainRunBody2.Status.Steps[0].Phase,
		"step-a must be succeeded before step-b can run")
	require.Equal(t, aotv1alpha1.ChainRunStepPhaseRunning, chainRunBody2.Status.Steps[1].Phase,
		"step-b must start only after step-a succeeds")
}

// TestChainExecution_InvalidDAG_MissingDependency verifies that creating a
// chain whose step references an undefined dependency name is rejected with 400.
func TestChainExecution_InvalidDAG_MissingDependency(t *testing.T) {
	baseURL, _, cleanup := startChainServer(t)
	defer cleanup()

	resp := testutil.PostJSON(t, baseURL+"/api/v1/chains", map[string]interface{}{
		"name": "bad-chain",
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": "tmpl-a", "dependsOn": []string{"nonexistent"}},
		},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"chain with a step referencing an undefined dependency must be rejected with 400")
}

// TestChainExecution_GetChainRunNotFound verifies that fetching a non-existent
// ChainRun returns 404.
func TestChainExecution_GetChainRunNotFound(t *testing.T) {
	baseURL, _, cleanup := startChainServer(t)
	defer cleanup()

	resp, err := http.Get(baseURL + "/api/v1/chainruns/cr-does-not-exist") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"GET for a non-existent ChainRun must return 404")
}

// TestChainExecution_TriggerNonExistentChain verifies that triggering a chain
// that does not exist returns 404.
func TestChainExecution_TriggerNonExistentChain(t *testing.T) {
	baseURL, _, cleanup := startChainServer(t)
	defer cleanup()

	resp, err := http.Post(baseURL+"/api/v1/chains/ghost-chain/trigger", "application/json", nil) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"triggering a non-existent chain must return 404")
}

// TestChainExecution_ListChainRunsReflectsAll verifies that all ChainRuns
// created by successive triggers appear in GET /api/v1/chainruns.
func TestChainExecution_ListChainRunsReflectsAll(t *testing.T) {
	baseURL, _, cleanup := startChainServer(t)
	defer cleanup()

	// Create a chain.
	cr := testutil.PostJSON(t, baseURL+"/api/v1/chains", map[string]interface{}{
		"name": "multi-run-chain",
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": "tmpl-a"},
		},
	})
	cr.Body.Close()
	require.Equal(t, http.StatusCreated, cr.StatusCode)

	// Trigger it twice.
	for i := 0; i < 2; i++ {
		r, err := http.Post(baseURL+"/api/v1/chains/multi-run-chain/trigger", "application/json", nil) //nolint:noctx
		require.NoError(t, err)
		r.Body.Close()
		require.Equal(t, http.StatusCreated, r.StatusCode)
	}

	listResp, err := http.Get(baseURL + "/api/v1/chainruns") //nolint:noctx
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	var runs []aotv1alpha1.ChainRun
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&runs))
	require.Len(t, runs, 2, "list must return all ChainRuns created by triggers")
}
