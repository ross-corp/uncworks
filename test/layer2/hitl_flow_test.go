// test/layer2/hitl_flow_test.go — Layer 2 tests for the human-in-the-loop (HITL) flow.
// Tests observable API behavior: WaitingForInput phase transitions and SendHumanInput
// precondition enforcement via the ConnectRPC handler backed by a fake k8s client.
package layer2

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestHITLFlow_PhaseTransitionToWaitingForInput verifies that a run's status
// phase is correctly exposed as WAITING_FOR_INPUT after the controller sets it.
func TestHITLFlow_PhaseTransitionToWaitingForInput(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Ask me a question",
		},
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	// Simulate the controller pausing the run and awaiting human input.
	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseWaitingForInput
	crd.Status.Message = "Waiting for human guidance"
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT, getResp.Msg.Status.Phase,
		"phase must be WAITING_FOR_INPUT after controller sets it")
	assert.Equal(t, "Waiting for human guidance", getResp.Msg.Status.Message)
}

// TestHITLFlow_SendHumanInput_WhenNotWaiting verifies that SendHumanInput is
// rejected with FailedPrecondition when the run is not in WaitingForInput phase.
func TestHITLFlow_SendHumanInput_WhenNotWaiting(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Run quietly without pausing",
		},
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	_, err = c.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: runID,
		Input:      "yes, proceed",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err),
		"sending input to a non-waiting run must return FailedPrecondition")
}

// TestHITLFlow_SendHumanInput_NotFound verifies that SendHumanInput returns
// NotFound for a run that does not exist.
func TestHITLFlow_SendHumanInput_NotFound(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	_, err := c.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: "nonexistent-run",
		Input:      "any input",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// TestHITLFlow_SendHumanInput_WhenWaiting_NoTemporal verifies that SendHumanInput
// returns Unavailable (not FailedPrecondition) when the run is correctly in
// WaitingForInput phase but the Temporal client is not configured.
// This isolates the precondition check from the downstream Temporal dependency.
func TestHITLFlow_SendHumanInput_WhenWaiting_NoTemporal(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Pause for input",
		},
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	// Move the run into WaitingForInput.
	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseWaitingForInput
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	// Without a Temporal client, the handler returns Unavailable after passing the
	// precondition check.
	_, err = c.SendHumanInput(context.Background(), connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: runID,
		Input:      "proceed with option A",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err),
		"WaitingForInput run without Temporal should return Unavailable (not FailedPrecondition)")
}

// TestHITLFlow_WaitingForInput_IsListable verifies that a run in WaitingForInput
// phase appears in list results and is filterable by phase.
func TestHITLFlow_WaitingForInput_IsListable(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	// Create two runs; move one to WaitingForInput.
	createWaiting, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Will pause",
		},
	}))
	require.NoError(t, err)
	waitingID := createWaiting.Msg.AgentRun.Id

	_, err = c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: "https://github.com/example/repo.git"}},
			Prompt:  "Will not pause",
		},
	}))
	require.NoError(t, err)

	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      waitingID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseWaitingForInput
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	// Phase filter: only WaitingForInput runs should be returned.
	listResp, err := c.ListAgentRuns(context.Background(), connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT,
	}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.AgentRuns, 1,
		"phase filter must return exactly the WaitingForInput run")
	assert.Equal(t, waitingID, listResp.Msg.AgentRuns[0].Id)
	assert.Equal(t, apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT,
		listResp.Msg.AgentRuns[0].Status.Phase)
}
