//go:build e2e

// Temporal integration E2E tests.
// Requires: k0s cluster, Temporal dev server, controller, and temporal-worker running.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	temporalclient "go.temporal.io/sdk/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

func getTemporalClient(t *testing.T) temporalclient.Client {
	t.Helper()

	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}

	c, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  host,
		Namespace: "default",
	})
	if err != nil {
		t.Skipf("Skipping Temporal E2E test: cannot connect to Temporal at %s: %v", host, err)
	}

	return c
}

// waitForAnnotation polls until the AgentRun has the workflow-id annotation.
func waitForAnnotation(ctx context.Context, t *testing.T, k8s client.Client, name, namespace string, timeout time.Duration) *aotv1alpha1.AgentRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ar := &aotv1alpha1.AgentRun{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ar); err != nil {
			time.Sleep(time.Second)
			continue
		}
		if ar.Annotations["aot.uncworks.io/workflow-id"] != "" {
			return ar
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Timed out waiting for workflow-id annotation on %s", name)
	return nil
}

// waitForPhase polls until the AgentRun reaches the given phase.
func waitForPhase(ctx context.Context, t *testing.T, k8s client.Client, name, namespace string, phase aotv1alpha1.AgentRunPhase, timeout time.Duration) *aotv1alpha1.AgentRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ar := &aotv1alpha1.AgentRun{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ar); err != nil {
			time.Sleep(time.Second)
			continue
		}
		if ar.Status.Phase == phase {
			return ar
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Timed out waiting for phase %s on %s", phase, name)
	return nil
}

// waitForDeletion polls until the AgentRun no longer exists.
func waitForDeletion(ctx context.Context, t *testing.T, k8s client.Client, name, namespace string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ar := &aotv1alpha1.AgentRun{}
		err := k8s.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ar)
		if err != nil {
			return // deleted
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Timed out waiting for deletion of %s", name)
}

func TestE2E_Temporal_WorkflowStarts(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-wf-start-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:     "E2E: verify workflow starts",
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer k8s.Delete(ctx, ar)

	// Wait for workflow annotation
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 30*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow started: %s", workflowID)

	if workflowID != fmt.Sprintf("agentrun-%s", runName) {
		t.Errorf("unexpected workflow ID: %s", workflowID)
	}

	// Verify finalizer is set
	hasFinalizer := false
	for _, f := range fetched.Finalizers {
		if f == "aot.uncworks.io/workflow-cleanup" {
			hasFinalizer = true
			break
		}
	}
	if !hasFinalizer {
		t.Error("expected finalizer aot.uncworks.io/workflow-cleanup to be set")
	}

	// Query workflow state via Temporal
	resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	var state aottemporal.WorkflowState
	if err := resp.Get(&state); err != nil {
		t.Fatalf("Decode state: %v", err)
	}
	t.Logf("Workflow state: phase=%s message=%s pod=%s", state.Phase, state.Message, state.PodName)

	if state.PodName == "" {
		t.Error("expected non-empty PodName in workflow state")
	}
}

func TestE2E_Temporal_CancelViaWorkflow(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-wf-cancel-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:     "E2E: cancel via workflow",
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer k8s.Delete(ctx, ar)

	// Wait for workflow
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 30*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow: %s", workflowID)

	// Cancel workflow
	if err := tc.CancelWorkflow(ctx, workflowID, ""); err != nil {
		t.Fatalf("CancelWorkflow: %v", err)
	}
	t.Log("Cancelled workflow")

	// Wait for CRD to reach Cancelled phase (controller syncs every 30s)
	fetched = waitForPhase(ctx, t, k8s, runName, "default", aotv1alpha1.AgentRunPhaseCancelled, 60*time.Second)
	t.Logf("CRD synced: phase=%s message=%s", fetched.Status.Phase, fetched.Status.Message)

	if fetched.Status.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestE2E_Temporal_DeleteCRDCancelsWorkflow(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-wf-delete-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:     "E2E: delete CRD cancels workflow",
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Wait for workflow
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 30*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]
	t.Logf("Workflow: %s", workflowID)

	// Delete CRD
	if err := k8s.Delete(ctx, fetched); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Wait for CRD to be gone
	waitForDeletion(ctx, t, k8s, runName, "default", 30*time.Second)
	t.Log("CRD deleted")

	// Verify workflow was cancelled
	time.Sleep(5 * time.Second)
	desc, err := tc.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		t.Fatalf("DescribeWorkflow: %v", err)
	}

	status := desc.GetWorkflowExecutionInfo().GetStatus().String()
	t.Logf("Workflow status after CRD deletion: %s", status)
	if status != "Canceled" {
		t.Errorf("expected workflow to be Canceled, got %s", status)
	}
}

func TestE2E_Temporal_KubeVirtRejection(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-wf-kubevirt-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendKubeVirt,
			Repos:   []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git"}},
			Prompt:  "E2E: KubeVirt should fail with message",
			KubeVirtConfig: &aotv1alpha1.KubeVirtBackendConfig{
				CPUs:     2,
				MemoryMB: 4096,
				DiskGB:   20,
			},
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer k8s.Delete(ctx, ar)

	// Should reach Failed quickly — no workflow should be created
	fetched := waitForPhase(ctx, t, k8s, runName, "default", aotv1alpha1.AgentRunPhaseFailed, 30*time.Second)
	t.Logf("Phase: %s, Message: %s", fetched.Status.Phase, fetched.Status.Message)

	if fetched.Status.Message != "KubeVirt backend is not yet implemented" {
		t.Errorf("unexpected message: %s", fetched.Status.Message)
	}

	// Verify no Temporal workflow was created
	workflowID := fmt.Sprintf("agentrun-%s", runName)
	_, err := tc.DescribeWorkflowExecution(ctx, workflowID, "")
	if err == nil {
		t.Error("expected no workflow for KubeVirt backend, but found one")
	}
}

func TestE2E_Temporal_StateSync(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-wf-sync-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:    aotv1alpha1.BackendPod,
			Repos:      []aotv1alpha1.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
			Prompt:     "E2E: verify state sync",
			TTLSeconds: 600,
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() {
		tc.CancelWorkflow(ctx, fmt.Sprintf("agentrun-%s", runName), "")
		k8s.Delete(ctx, ar)
	}()

	// Wait for workflow annotation
	fetched := waitForAnnotation(ctx, t, k8s, runName, "default", 30*time.Second)
	workflowID := fetched.Annotations["aot.uncworks.io/workflow-id"]

	// Query Temporal state
	resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	var state aottemporal.WorkflowState
	if err := resp.Get(&state); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	// Wait for controller to sync state to CRD (reconcile interval is 30s)
	time.Sleep(35 * time.Second)

	synced := &aotv1alpha1.AgentRun{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: runName, Namespace: "default"}, synced); err != nil {
		t.Fatalf("Get: %v", err)
	}

	t.Logf("Temporal state: phase=%s message=%s", state.Phase, state.Message)
	t.Logf("CRD status: phase=%s message=%s podName=%s", synced.Status.Phase, synced.Status.Message, synced.Status.PodName)

	// The CRD message should match the Temporal state message
	if synced.Status.Message != state.Message {
		t.Errorf("CRD message %q does not match Temporal state message %q", synced.Status.Message, state.Message)
	}

	if synced.Status.PodName == "" {
		t.Error("expected podName to be set on CRD status")
	}
}
