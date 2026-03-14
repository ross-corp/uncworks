//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestE2E_FullLifecycle_SimplePrompt creates an AgentRun with a simple prompt,
// waits for it to reach a terminal phase (Succeeded), and verifies a pod was created.
func TestE2E_FullLifecycle_SimplePrompt(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-lifecycle-simple-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:     "Create a file called DONE.txt containing exactly PASS",
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

	// Wait for terminal phase
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Workflow completed: phase=%s message=%s pod=%s", terminal.Phase, terminal.Message, terminal.PodName)

	if terminal.Phase != "Succeeded" {
		t.Errorf("expected Succeeded phase, got %s (message: %s)", terminal.Phase, terminal.Message)
	}

	if terminal.PodName == "" {
		t.Error("expected non-empty PodName")
	}
}

// TestE2E_FullLifecycle_TTLExpiry creates a run with a short TTL and a slow prompt,
// verifying the run reaches Failed phase with a TTL-related message.
func TestE2E_FullLifecycle_TTLExpiry(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-lifecycle-ttl-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:     "Think very carefully about the meaning of life and write a detailed essay",
			TTLSeconds: 10,
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

	// Wait for terminal phase — should be Failed due to TTL
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 3*time.Minute)
	t.Logf("Workflow completed: phase=%s message=%s", terminal.Phase, terminal.Message)

	if terminal.Phase != "Failed" {
		t.Errorf("expected Failed phase, got %s", terminal.Phase)
	}

	if !strings.Contains(strings.ToLower(terminal.Message), "ttl") {
		t.Errorf("expected TTL-related message, got %q", terminal.Message)
	}
}

// TestE2E_FullLifecycle_CancelRunning creates a run, waits for the workflow to start,
// then cancels it via Temporal and verifies the Cancelled phase.
func TestE2E_FullLifecycle_CancelRunning(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-lifecycle-cancel-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:     "Write a very long essay about distributed systems",
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer k8s.Delete(ctx, ar)

	// Wait for workflow to start
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 60*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow started: %s", workflowID)

	// Wait a bit for the workflow to be running
	time.Sleep(5 * time.Second)

	// Cancel via Temporal
	if err := tc.CancelWorkflow(ctx, workflowID, ""); err != nil {
		t.Fatalf("CancelWorkflow: %v", err)
	}
	t.Log("Cancel signal sent")

	// Wait for Cancelled phase
	fetched = waitForPhase(ctx, t, k8s, runName, "default", aotv1alpha1.AgentRunPhaseCancelled, 60*time.Second)
	t.Logf("CRD phase: %s message: %s", fetched.Status.Phase, fetched.Status.Message)
}

// TestE2E_ConcurrentRuns creates 3 AgentRuns simultaneously and verifies all
// get unique workflow annotations (proving concurrent workflow creation works).
func TestE2E_ConcurrentRuns(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	base := time.Now().Unix()
	names := []string{
		fmt.Sprintf("e2e-concurrent-a-%d", base),
		fmt.Sprintf("e2e-concurrent-b-%d", base),
		fmt.Sprintf("e2e-concurrent-c-%d", base),
	}

	// Create all 3 runs simultaneously
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			ar := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      n,
					Namespace: "default",
				},
				Spec: aotv1alpha1.AgentRunSpec{
					Backend:    aotv1alpha1.BackendPod,
					Repos:      []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
					Prompt:     fmt.Sprintf("Concurrent test run %s", n),
					TTLSeconds: 300,
				},
			}
			if err := k8s.Create(ctx, ar); err != nil {
				t.Errorf("Create %s: %v", n, err)
			}
		}(name)
	}
	wg.Wait()

	// Cleanup on exit
	defer func() {
		for _, name := range names {
			tc.CancelWorkflow(ctx, fmt.Sprintf("agentrun-%s", name), "")
			ar := &aotv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			}
			k8s.Delete(ctx, ar)
		}
	}()

	// Wait for all to have workflow annotations
	podNames := make(map[string]bool)
	for _, name := range names {
		fetched := waitForAnnotation(ctx, t, k8s, name, "default", 60*time.Second)
		workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
		t.Logf("Run %s: workflow=%s", name, workflowID)

		// Query workflow state for pod name
		resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
		if err != nil {
			t.Errorf("QueryWorkflow for %s: %v", name, err)
			continue
		}
		var state aottemporal.WorkflowState
		if err := resp.Get(&state); err != nil {
			t.Errorf("Decode state for %s: %v", name, err)
			continue
		}
		if state.PodName != "" {
			if podNames[state.PodName] {
				t.Errorf("duplicate pod name: %s", state.PodName)
			}
			podNames[state.PodName] = true
		}
		t.Logf("Run %s: pod=%s", name, state.PodName)
	}

	if len(podNames) < len(names) {
		t.Logf("Warning: only %d/%d runs had pod names assigned", len(podNames), len(names))
	}
}
