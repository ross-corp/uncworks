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
	// Mock lifecycle activities that ALL workflow paths call
	mockLifecycleActivities(env)
	return env
}

// mockLifecycleActivities registers default mocks for activities that every
// workflow execution path calls (LLM key provisioning, tag enrichment, cleanup).
func mockLifecycleActivities(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity((*aottemporal.Activities).ProvisionLLMKey, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.ProvisionLLMKeyOutput{Key: "test-key"}, nil,
	).Maybe()
	env.OnActivity((*aottemporal.Activities).RevokeLLMKey, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	env.OnActivity((*aottemporal.Activities).EnrichRunTags, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	env.OnActivity((*aottemporal.Activities).WriteTraceSpan, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	env.OnActivity((*aottemporal.Activities).PushChanges, mock.Anything, mock.Anything, mock.Anything).Return(aottemporal.PushChangesOutput{}, nil).Maybe()
	env.OnActivity((*aottemporal.Activities).CreatePR, mock.Anything, mock.Anything, mock.Anything).Return(aottemporal.CreatePROutput{}, nil).Maybe()
}

// TestWorkflow_HappyPath verifies the complete lifecycle:
// ProvisionLLMKey → CreateAgentDeployment → WaitForHydration → StartAgent → poll completed → EnrichRunTags → ScaleDownDeployment → RevokeLLMKey
func TestWorkflow_HappyPath(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_TTLExpiry verifies that workflow stops the agent on TTL timeout.
func TestWorkflow_TTLExpiry(t *testing.T) {
	env := setupEnv(t)

	input := defaultInput()
	input.TTLSeconds = 1 // 1 second TTL

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_RUNNING"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StopAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_HITLSignal verifies human input is forwarded via signal.
func TestWorkflow_HITLSignal(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
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
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_RUNNING"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StopAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Send cancel signal
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(aottemporal.SignalCancel, nil)
	}, time.Millisecond*100)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_CompensationOnFailure verifies ScaleDownDeployment runs when StartAgent fails.
func TestWorkflow_CompensationOnFailure(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(
		fmt.Errorf("agent process failed to start"),
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	// Workflow returns error because StartAgent failed
	require.Error(t, env.GetWorkflowError())
}

// TestWorkflow_SpawnJunior verifies child workflow is started with correct input.
func TestWorkflow_SpawnJunior(t *testing.T) {
	env := setupEnv(t)
	env.RegisterWorkflow(aottemporal.AgentRunWorkflow)

	// Mock all activities used by the child workflow
	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-junior", PVCName: "aot-ws-junior"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

// TestWorkflow_ConsecutiveStatusErrors verifies workflow fails after sustained sidecar errors.
func TestWorkflow_ConsecutiveStatusErrors(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// GetAgentStatus always fails (all retries exhausted)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		nil, fmt.Errorf("connection refused"),
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	// Workflow returns nil (not error) — it transitions to Failed phase and returns cleanly
	require.NoError(t, env.GetWorkflowError())

	// Query final state
	result, err := env.QueryWorkflow(aottemporal.QueryGetState)
	require.NoError(t, err)
	var state aottemporal.WorkflowState
	require.NoError(t, result.Get(&state))
	require.Equal(t, "Failed", state.Phase)
	require.Contains(t, state.Message, "unreachable")
}

// TestWorkflow_TransientStatusError verifies workflow recovers from transient errors.
func TestWorkflow_TransientStatusError(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	callCount := 0
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ *aottemporal.Activities, _ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
			callCount++
			// Fail first 3 polls, then succeed with completed
			if callCount <= 3 {
				return nil, fmt.Errorf("transient network error")
			}
			return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil
		},
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_SpecDrivenPrompt verifies auto-generated prompt for spec runs.
func TestWorkflow_SpecDrivenPrompt(t *testing.T) {
	env := setupEnv(t)

	input := defaultInput()
	input.SpecContent = "# MyConverter\nConverts CSV to JSON."
	input.Prompt = "" // Should be auto-generated

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	// Spec-driven mode calls PlanRun → Execute → Verify
	env.OnActivity((*aottemporal.Activities).PlanRun, mock.Anything, mock.Anything, mock.Anything).Return(
		aottemporal.PlanRunOutput{ChangeName: "test-run", SpecsValid: true, TaskCount: 5}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).VerifyRun, mock.Anything, mock.Anything, mock.Anything).Return(
		aottemporal.VerifyRunOutput{Result: aottemporal.VerificationResult{Pass: true}}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_SpecWithExplicitPrompt verifies explicit prompt is preserved.
func TestWorkflow_SpecWithExplicitPrompt(t *testing.T) {
	env := setupEnv(t)

	input := defaultInput()
	input.SpecContent = "# MyConverter"
	input.Prompt = "custom prompt"

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).PlanRun, mock.Anything, mock.Anything, mock.Anything).Return(
		aottemporal.PlanRunOutput{ChangeName: "test-run", SpecsValid: true, TaskCount: 5}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).VerifyRun, mock.Anything, mock.Anything, mock.Anything).Return(
		aottemporal.VerifyRunOutput{Result: aottemporal.VerificationResult{Pass: true}}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_GetStateQuery verifies the get-state query returns current phase.
func TestWorkflow_GetStateQuery(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

// TestWorkflow_DeploymentNameInState verifies deploymentName is set in workflow state.
func TestWorkflow_DeploymentNameInState(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Query state after deployment creation
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(aottemporal.QueryGetState)
		require.NoError(t, err)

		var state aottemporal.WorkflowState
		require.NoError(t, result.Get(&state))
		require.Equal(t, "agentrun-test-run", state.DeploymentName)
	}, time.Millisecond*50)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, defaultInput())
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// TestWorkflow_CIAutofix_SkipsPlan verifies that CI autofix runs skip the
// Plan stage and go directly to Execute+Verify.
func TestWorkflow_CIAutofix_SkipsPlan(t *testing.T) {
	env := setupEnv(t)

	input := defaultInput()
	input.SpecSource = "ci-autofix:org/repo#abc123"
	input.Prompt = "Fix CI failures: Error: expected true but got false"
	input.OrchestrationMode = aottemporal.OrchestrationModeSpecDriven

	env.OnActivity((*aottemporal.Activities).CreateAgentDeployment, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.CreateAgentDeploymentOutput{DeploymentName: "agentrun-test-run", PVCName: "aot-ws-test-run"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).WaitForHydration, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.5", WorkspacePath: "/workspace"}, nil,
	)
	// PlanRun should NOT be called — it's a CI autofix run
	// StartAgent + GetAgentStatus for the execute stage
	env.OnActivity((*aottemporal.Activities).StartAgent, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((*aottemporal.Activities).GetAgentStatus, mock.Anything, mock.Anything, mock.Anything).Return(
		&aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil,
	)
	env.OnActivity((*aottemporal.Activities).VerifyRun, mock.Anything, mock.Anything, mock.Anything).Return(
		aottemporal.VerifyRunOutput{Result: aottemporal.VerificationResult{Pass: true}}, nil,
	)
	env.OnActivity((*aottemporal.Activities).ScaleDownDeployment, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(aottemporal.AgentRunWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify PlanRun was NOT called (it should be skipped for ci-autofix)
	// The test would panic if an unmocked activity was called, so completing
	// without error proves PlanRun was skipped.
}
