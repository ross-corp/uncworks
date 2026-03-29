// test/layer2/agentrun_lifecycle_test.go — Layer 2 pipeline tests for AgentRun lifecycle.
// Uses a fake k8s client and the ConnectRPC handler directly via httptest.
package layer2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/server"
)

var layer2Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(layer2Scheme))
	utilruntime.Must(aotv1alpha1.AddToScheme(layer2Scheme))
}

// newTestServer starts an httptest server backed by a fake k8s client.
// Returns the ConnectRPC client, the fake k8s client (for direct status mutations),
// and a cleanup function.
func newTestServer(t *testing.T) (apiv1connect.AOTServiceClient, client.Client, func()) {
	t.Helper()

	k8sClient := fake.NewClientBuilder().
		WithScheme(layer2Scheme).
		WithStatusSubresource(&aotv1alpha1.AgentRun{}).
		Build()

	svc := server.NewAOTServiceHandler(k8sClient, &eventbus.NoOpEventBus{}, "default")
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAOTServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()

	c := apiv1connect.NewAOTServiceClient(srv.Client(), srv.URL)
	return c, k8sClient, srv.Close
}

// TestAgentRunLifecycle_CreateReturnsInitialPhase verifies that a newly
// created AgentRun starts in the Pending phase.
func TestAgentRunLifecycle_CreateReturnsInitialPhase(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	resp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Add unit tests",
		},
	}))
	require.NoError(t, err)

	run := resp.Msg.AgentRun
	assert.NotEmpty(t, run.Id, "created run must have a non-empty ID")
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING, run.Status.Phase,
		"new run must start in PENDING phase")
}

// TestAgentRunLifecycle_GetAfterCreate verifies that GetAgentRun returns the
// same run data that was set at creation time.
func TestAgentRunLifecycle_GetAfterCreate(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	const prompt = "Refactor auth module"

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  prompt,
		},
	}))
	require.NoError(t, err)

	runID := createResp.Msg.AgentRun.Id
	require.NotEmpty(t, runID)

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)

	assert.Equal(t, runID, getResp.Msg.Id)
	assert.Equal(t, prompt, getResp.Msg.Spec.Prompt)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING, getResp.Msg.Status.Phase)
}

// TestAgentRunLifecycle_StatusReflectsUpdate verifies that after the controller
// mutates the CRD's status phase, GetAgentRun returns the updated phase.
func TestAgentRunLifecycle_StatusReflectsUpdate(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Implement feature X",
		},
	}))
	require.NoError(t, err)

	runID := createResp.Msg.AgentRun.Id

	// Simulate the controller moving the run to Running.
	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      runID,
	}, crd))

	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	crd.Status.Message = "Agent pod scheduled"
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING, getResp.Msg.Status.Phase)
}

// TestAgentRunLifecycle_FailedPhase verifies the full lifecycle through to Failed.
func TestAgentRunLifecycle_FailedPhase(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Task that will fail",
		},
	}))
	require.NoError(t, err)

	runID := createResp.Msg.AgentRun.Id

	// Simulate controller marking as failed.
	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseFailed
	crd.Status.Message = "LLM unreachable"
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED, getResp.Msg.Status.Phase)
	assert.Equal(t, "LLM unreachable", getResp.Msg.Status.Message)
}

// TestAgentRunLifecycle_GetNotFound verifies that GetAgentRun returns a not-found
// error for a run that does not exist.
func TestAgentRunLifecycle_GetNotFound(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	_, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: "nonexistent-run-id",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// TestAgentRunLifecycle_ListRuns verifies that ListAgentRuns returns all created runs.
func TestAgentRunLifecycle_ListRuns(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	prompts := []string{"task one", "task two", "task three"}
	for _, p := range prompts {
		_, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Backend: apiv1.Backend_BACKEND_POD,
				Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
				Prompt:  p,
			},
		}))
		require.NoError(t, err)
	}

	listResp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Msg.AgentRuns), len(prompts),
		"list should return at least the runs we created")
}
