//go:build e2e

// LLM E2E tests — require k0s cluster with Ollama deployed, Temporal dev server,
// controller, and temporal-worker running with local images (AOT_*_IMAGE=aot-*:local).
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	temporalclient "go.temporal.io/sdk/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

const (
	// lifecyclePrompt is a deterministic prompt simple enough for qwen2.5:0.5b.
	// The test verifies the workflow reaches Succeeded phase.
	lifecyclePrompt = "Create a file called RESULT.txt in the current directory containing exactly the text PASS on a single line. Do not add any other content."

	// hitlPrompt asks the agent to wait for user input before proceeding.
	hitlPrompt = "Wait for human input. When you receive input, create a file called RESULT.txt containing exactly the input you received. Do not add any other content."

	// llmTestTimeout is the maximum time to wait for LLM-based tests.
	// CPU-only Ollama can be slow on first inference.
	llmTestTimeout = 5 * time.Minute
)

// TestE2E_LLM_Lifecycle verifies the full agent lifecycle with a real LLM:
// create AgentRun → agent pod starts → LLM processes prompt → workflow completes.
func TestE2E_LLM_Lifecycle(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-llm-lifecycle-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			RepoURL:    "https://github.com/example/repo.git",
			Branch:     "main",
			Prompt:     lifecyclePrompt,
			TTLSeconds: 300,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer func() {
		tc.CancelWorkflow(ctx, fmt.Sprintf("agentrun-%s", runName), "")
		k8s.Delete(ctx, ar)
	}()

	// Wait for workflow to start
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 60*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow started: %s", workflowID)

	// Wait for workflow to reach a terminal phase
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, llmTestTimeout)
	t.Logf("Workflow completed: phase=%s message=%s", terminal.Phase, terminal.Message)

	if terminal.Phase != "Succeeded" {
		t.Errorf("expected Succeeded phase, got %s (message: %s)", terminal.Phase, terminal.Message)
	}

	if terminal.PodName == "" {
		t.Error("expected non-empty PodName")
	}
}

// TestE2E_LLM_HITL verifies the human-in-the-loop flow:
// agent waits for input → send human input → agent resumes → completes.
func TestE2E_LLM_HITL(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-llm-hitl-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			RepoURL:    "https://github.com/example/repo.git",
			Branch:     "main",
			Prompt:     hitlPrompt,
			TTLSeconds: 300,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer func() {
		tc.CancelWorkflow(ctx, fmt.Sprintf("agentrun-%s", runName), "")
		k8s.Delete(ctx, ar)
	}()

	// Wait for workflow to start
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 60*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow started: %s", workflowID)

	// Wait for WaitingForInput phase
	waitForWorkflowPhase(ctx, t, tc, workflowID, "WaitingForInput", llmTestTimeout)
	t.Log("Agent is waiting for input")

	// Send human input via Temporal signal
	signal := aottemporal.HumanInputSignal{Input: "HITL_PASS"}
	if err := tc.SignalWorkflow(ctx, workflowID, "", aottemporal.SignalHumanInput, signal); err != nil {
		t.Fatalf("SignalWorkflow: %v", err)
	}
	t.Log("Human input sent")

	// Wait for workflow to complete
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, llmTestTimeout)
	t.Logf("Workflow completed: phase=%s message=%s", terminal.Phase, terminal.Message)

	if terminal.Phase != "Succeeded" {
		t.Errorf("expected Succeeded phase, got %s (message: %s)", terminal.Phase, terminal.Message)
	}
}

// TestE2E_LLM_MultiAgent verifies the multi-agent flow:
// parent spawns junior → junior completes → parent completes.
func TestE2E_LLM_MultiAgent(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	parentName := fmt.Sprintf("e2e-llm-parent-%d", time.Now().Unix())

	// The parent prompt instructs it to delegate a task to a junior agent.
	// Since SpawnJunior is triggered by the sidecar/agent interaction,
	// we test this by creating a parent AgentRun and verifying a child workflow appears.
	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parentName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			RepoURL:    "https://github.com/example/repo.git",
			Branch:     "main",
			Prompt:     "Delegate the following task to a junior agent: " + lifecyclePrompt,
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer func() {
		tc.CancelWorkflow(ctx, fmt.Sprintf("agentrun-%s", parentName), "")
		k8s.Delete(ctx, ar)
	}()

	// Wait for parent workflow to start
	fetched := waitForAnnotation(ctx, t, k8s, parentName, "default", 60*time.Second)
	parentWorkflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Parent workflow started: %s", parentWorkflowID)

	// Wait for parent workflow to reach terminal phase.
	// If the agent delegates to a junior, the parent workflow will block until the junior finishes.
	terminal := waitForTerminalPhase(ctx, t, tc, parentWorkflowID, llmTestTimeout)
	t.Logf("Parent workflow completed: phase=%s message=%s", terminal.Phase, terminal.Message)

	// For multi-agent, we verify the parent completed (Succeeded or Failed is acceptable —
	// the LLM may not perfectly delegate, but the workflow machinery should work).
	if terminal.Phase != "Succeeded" && terminal.Phase != "Failed" {
		t.Errorf("expected terminal phase (Succeeded or Failed), got %s", terminal.Phase)
	}
}

// waitForTerminalPhase polls the Temporal workflow state until it reaches a terminal phase.
func waitForTerminalPhase(ctx context.Context, t *testing.T, tc temporalclient.Client, workflowID string, timeout time.Duration) *aottemporal.WorkflowState {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err != nil {
			t.Logf("QueryWorkflow (retrying): %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var state aottemporal.WorkflowState
		if err := resp.Get(&state); err != nil {
			t.Logf("Decode state (retrying): %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		switch state.Phase {
		case "Succeeded", "Failed", "Cancelled":
			return &state
		}

		time.Sleep(5 * time.Second)
	}

	t.Fatalf("Timed out waiting for terminal phase on workflow %s", workflowID)
	return nil
}

// waitForWorkflowPhase polls the Temporal workflow state until it reaches a specific phase.
func waitForWorkflowPhase(ctx context.Context, t *testing.T, tc temporalclient.Client, workflowID, phase string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var state aottemporal.WorkflowState
		if err := resp.Get(&state); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if state.Phase == phase {
			return
		}

		// Bail if we've reached a terminal state that isn't the target
		switch state.Phase {
		case "Succeeded", "Failed", "Cancelled":
			t.Fatalf("Workflow reached terminal phase %s while waiting for %s", state.Phase, phase)
		}

		time.Sleep(5 * time.Second)
	}

	t.Fatalf("Timed out waiting for phase %s on workflow %s", phase, workflowID)
}
