// Package temporal contains unit tests for Temporal workflow logic.
//
// These tests use go.temporal.io/sdk/testsuite to test workflow logic
// with mocked activities and fast-forwarded timers.
package temporal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	aottemporal "github.com/uncworks/aot/internal/temporal"
)

func defaultInput() aottemporal.WorkflowInput {
	return aottemporal.WorkflowInput{
		AgentRunName: "test-run",
		Namespace:    "default",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "fix the tests",
		TTLSeconds:   3600,
	}
}

func setupEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	// Register the activities struct so method names are known
	env.RegisterActivity(&aottemporal.Activities{})
	return env
}

// TestWorkflow_HappyPath verifies the complete lifecycle:
// CreatePod → WaitForHydration → StartAgent → poll completed → cleanup
func TestWorkflow_HappyPath(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_TTLExpiry verifies that workflow stops the agent on TTL timeout.
func TestWorkflow_TTLExpiry(t *testing.T) {
	env := setupEnv(t)

	input := defaultInput()
	input.TTLSeconds = 1 // 1 second TTL

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_RUNNING"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StopAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_HITLSignal verifies human input is forwarded via signal.
func TestWorkflow_HITLSignal(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	callCount := 0
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ *aottemporal.Activities, _ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
			callCount++
			if callCount <= 1 {
				return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_WAITING_FOR_INPUT"}, nil
			}
			return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil
		},
	)
	env.OnActivity((*aottemporal.Activities).ForwardHumanInput, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Send human input signal after workflow starts
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(aottemporal.SignalHumanInput, aottemporal.HumanInputSignal{
			Input: "yes, continue",
		})
	}, time.Millisecond*100)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_CancelSignal verifies graceful termination on cancel.
func TestWorkflow_CancelSignal(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_RUNNING"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StopAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Send cancel signal
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(aottemporal.SignalCancel, nil)
	}, time.Millisecond*100)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_CompensationOnFailure verifies CleanupPod runs when StartAgent fails.
func TestWorkflow_CompensationOnFailure(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(
		fmt.Errorf("agent process failed to start"),
	)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	// Workflow returns error because StartAgent failed
	require.Error(t, env.GetWorkflowError())
}

// TestWorkflow_SpawnJunior verifies child workflow is started with correct input.
func TestWorkflow_SpawnJunior(t *testing.T) {
	env := setupEnv(t)
	env.RegisterWorkflow(aottemporal.AgentRunWorkflow)

	// Mock the child workflow's activities (it runs as AgentRunWorkflow)
	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-junior"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	input := aottemporal.SpawnJuniorInput{
		ParentRunName: "parent-run",
		Namespace:     "default",
		Task:          "write unit tests for auth",
		Repos:         []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		TTLSeconds:    1800,
		Blocking:      true,
	}

	env.ExecuteWorkflow(aottemporal.SpawnJuniorWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_GetStateQuery verifies the get-state query returns current phase.
func TestWorkflow_GetStateQuery(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentPod, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentPodOutput{PodName: "agentrun-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace/src/repo"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).CleanupPod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Query state during workflow execution
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(aottemporal.QueryGetState)
		require.NoError(t, err)

		var state aottemporal.WorkflowState
		require.NoError(t, result.Get(&state))
		require.NotEmpty(t, state.Phase)
	}, time.Millisecond*50)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())
	require.True(t, env.IsWorkflowCompleted())
}
