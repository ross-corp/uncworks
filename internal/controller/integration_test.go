package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

func getTemporalClient(t *testing.T) temporalclient.Client {
	t.Helper()
	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}
	c, err := temporalclient.Dial(temporalclient.Options{HostPort: host, Namespace: "default"})
	if err != nil {
		t.Skipf("Skipping: cannot connect to Temporal at %s: %v", host, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.CheckHealth(ctx, &temporalclient.CheckHealthRequest{}); err != nil {
		c.Close()
		t.Skipf("Skipping: Temporal health check failed: %v", err)
	}
	return c
}

// TestIntegration_ReconcileStartsWorkflow verifies the CRD → Temporal workflow bridge:
// creating an AgentRun CRD should start a Temporal workflow and annotate the CRD.
func TestIntegration_ReconcileStartsWorkflow(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()

	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	taskQueue := fmt.Sprintf("test-ctrl-integration-%d", time.Now().UnixNano())
	reconciler.TemporalClient = tc
	reconciler.TaskQueue = taskQueue

	// Start a worker with mock activities so the workflow can make progress
	w := worker.New(tc, taskQueue, worker.Options{})
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterActivity(&mockActivities{})
	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	ctx := context.Background()
	runName := fmt.Sprintf("integ-test-%d", time.Now().UnixNano()%100000)

	ar := newAgentRun(runName)
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create agentrun: %v", err)
	}

	// First reconcile: adds finalizer
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ar)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}

	// Second reconcile: starts workflow
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile (start workflow): %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected RequeueAfter > 0")
	}

	// Re-fetch to get latest state (annotation update + status update are separate writes)
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}
	workflowID := updated.Annotations["aot.uncworks.io/workflow-id"]
	if workflowID == "" {
		t.Fatal("expected workflow-id annotation to be set")
	}
	if workflowID != fmt.Sprintf("agentrun-%s", runName) {
		t.Errorf("unexpected workflow ID: %s", workflowID)
	}

	// Status subresource may not be split in envtest — check phase tolerantly
	t.Logf("CRD phase after reconcile: %s", updated.Status.Phase)
	if updated.Status.Phase != aotv1alpha1.AgentRunPhaseRunning && updated.Status.Phase != aotv1alpha1.AgentRunPhasePending {
		t.Errorf("expected Running or Pending phase, got %s", updated.Status.Phase)
	}

	// Verify the workflow is actually running in Temporal
	desc, err := tc.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		t.Fatalf("describe workflow: %v", err)
	}
	status := desc.GetWorkflowExecutionInfo().GetStatus().String()
	if status != "Running" && status != "Completed" {
		t.Errorf("expected workflow Running or Completed, got %s", status)
	}
}

// TestIntegration_ReconcileSyncsState verifies workflow state → CRD status sync:
// after a workflow completes, reconcile should update the CRD to terminal phase.
func TestIntegration_ReconcileSyncsState(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()

	reconciler, k8sClient, cleanup := setupReconciler(t)
	defer cleanup()

	taskQueue := fmt.Sprintf("test-ctrl-sync-%d", time.Now().UnixNano())
	reconciler.TemporalClient = tc
	reconciler.TaskQueue = taskQueue

	// Start worker with mock activities that complete immediately
	w := worker.New(tc, taskQueue, worker.Options{})
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterActivity(&mockActivities{})
	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	ctx := context.Background()
	runName := fmt.Sprintf("sync-test-%d", time.Now().UnixNano()%100000)

	ar := newAgentRun(runName)
	if err := k8sClient.Create(ctx, ar); err != nil {
		t.Fatalf("create: %v", err)
	}

	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ar)}

	// Reconcile: finalizer
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile (finalizer): %v", err)
	}

	// Reconcile: start workflow
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile (start): %v", err)
	}

	// Wait for workflow to complete (mock activities finish immediately, but poll loop takes ~5s)
	time.Sleep(8 * time.Second)

	// Reconcile: sync state from completed workflow
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile (sync): %v", err)
	}

	// Verify CRD reflects terminal state
	var updated aotv1alpha1.AgentRun
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ar), &updated); err != nil {
		t.Fatalf("get: %v", err)
	}

	if !isTerminal(updated.Status.Phase) {
		t.Errorf("expected terminal phase, got %s (message: %s)", updated.Status.Phase, updated.Status.Message)
	}
}

// mockActivities provides stub implementations for controller integration tests.
type mockActivities struct{}

func (m *mockActivities) ProvisionLLMKey(_ context.Context, _ aottemporal.ProvisionLLMKeyInput) (*aottemporal.ProvisionLLMKeyOutput, error) {
	return &aottemporal.ProvisionLLMKeyOutput{}, nil
}

func (m *mockActivities) CreateAgentPod(_ context.Context, input aottemporal.CreateAgentPodInput) (*aottemporal.CreateAgentPodOutput, error) {
	return &aottemporal.CreateAgentPodOutput{PodName: input.Name}, nil
}

func (m *mockActivities) WaitForHydration(_ context.Context, _ aottemporal.WaitForHydrationInput) error {
	return nil
}

func (m *mockActivities) StartAgent(_ context.Context, _ aottemporal.StartAgentInput) error {
	return nil
}

func (m *mockActivities) GetAgentStatus(_ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
	return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil
}

func (m *mockActivities) ForwardHumanInput(_ context.Context, _ aottemporal.ForwardHumanInputInput) error {
	return nil
}

func (m *mockActivities) StopAgent(_ context.Context, _ aottemporal.StopAgentInput) error {
	return nil
}

func (m *mockActivities) CleanupPod(_ context.Context, _ aottemporal.CleanupPodInput) error {
	return nil
}

func (m *mockActivities) RevokeLLMKey(_ context.Context, _ aottemporal.RevokeLLMKeyInput) error {
	return nil
}
