//go:build regression

// test/regression/run_lifecycle_test.go — Tests that the run lifecycle state
// machine is correctly reflected through the API layer. Uses a fake k8s client
// to simulate state transitions without a real cluster.
package regression

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/test/testutil"
)

// startLifecycleServer starts an in-process ConnectRPC test server and returns
// the client, the underlying k8s fake client (for direct state manipulation),
// and a cleanup function.
func startLifecycleServer(t *testing.T) (apiv1connect.AOTServiceClient, client.Client, func()) {
	t.Helper()
	return testutil.NewAOTServer(t)
}

// TestRunLifecycle_CreateAndGetPending verifies that after creating a run via
// the API it appears in Pending phase when fetched.
func TestRunLifecycle_CreateAndGetPending(t *testing.T) {
	connectClient, _, cleanup := startLifecycleServer(t)
	defer cleanup()

	ctx := context.Background()

	createResp, err := connectClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Regression: write tests for the auth layer"),
	}))
	require.NoError(t, err, "CreateAgentRun should not error")

	runID := createResp.Msg.AgentRun.Id
	require.NotEmpty(t, runID, "created run must have a non-empty ID")

	getResp, err := connectClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err, "GetAgentRun should not error")
	require.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING, getResp.Msg.Status.Phase,
		"freshly created run should be in Pending phase")
}

// TestRunLifecycle_StateTransition_PendingToRunning verifies that when the
// controller updates the k8s object status to Running, the API reflects the
// new phase on the next fetch.
func TestRunLifecycle_StateTransition_PendingToRunning(t *testing.T) {
	connectClient, k8sClient, cleanup := startLifecycleServer(t)
	defer cleanup()

	ctx := context.Background()

	createResp, err := connectClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Regression: lifecycle state transition test"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      runID,
	}, crd), "should be able to fetch the created AgentRun from the fake client")

	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	crd.Status.Message = "Agent pod is running"
	crd.Status.PodName = "agent-" + runID
	crd.Status.StartedAt = &metav1.Time{Time: metav1.Now().Time}
	require.NoError(t, k8sClient.Status().Update(ctx, crd),
		"simulating controller status update should succeed")

	getResp, err := connectClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	require.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING, getResp.Msg.Status.Phase,
		"API should reflect the Running phase set by the controller")
}

// TestRunLifecycle_StateTransition_RunningToSucceeded verifies the full happy-path
// lifecycle: Pending → Running → Succeeded.
func TestRunLifecycle_StateTransition_RunningToSucceeded(t *testing.T) {
	connectClient, k8sClient, cleanup := startLifecycleServer(t)
	defer cleanup()

	ctx := context.Background()

	createResp, err := connectClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Regression: full lifecycle test"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace, Name: runID,
	}, crd))

	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	require.NoError(t, k8sClient.Status().Update(ctx, crd))

	now := metav1.Now()

	// Re-fetch (fake client doesn't always reflect mutations on the same pointer).
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{
		Namespace: testutil.DefaultNamespace, Name: runID,
	}, crd))

	crd.Status.Phase = aotv1alpha1.AgentRunPhaseSucceeded
	crd.Status.Message = "Task completed successfully"
	crd.Status.CompletedAt = &metav1.Time{Time: now.Time}
	require.NoError(t, k8sClient.Status().Update(ctx, crd))

	getResp, err := connectClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	require.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED, getResp.Msg.Status.Phase,
		"API should reflect the Succeeded phase")
	require.Equal(t, "Task completed successfully", getResp.Msg.Status.Message)
}

// TestRunLifecycle_GetNotFound verifies that fetching a non-existent run
// returns a NotFound error.
func TestRunLifecycle_GetNotFound(t *testing.T) {
	connectClient, _, cleanup := startLifecycleServer(t)
	defer cleanup()

	_, err := connectClient.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: "ar-does-not-exist",
	}))
	require.Error(t, err, "GetAgentRun for non-existent ID should return an error")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err),
		"error code should be NotFound")
}
