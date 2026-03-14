//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// labelSelector returns the label selector for resources associated with an agent run.
func labelSelector(runID string) client.MatchingLabels {
	return client.MatchingLabels{"aot.uncworks.io/agentrun": runID}
}

// ---------- 16.1: TestE2E_DeploymentLifecycle ----------

func TestE2E_DeploymentLifecycle(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()
	ns := aotNamespace()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for workflow to start and pod to be running.
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)

	// Verify Deployment exists.
	var deployments appsv1.DeploymentList
	if err := k8s.List(ctx, &deployments, client.InNamespace(ns), labelSelector(runID)); err != nil {
		t.Fatalf("List Deployments: %v", err)
	}
	if len(deployments.Items) == 0 {
		t.Fatal("Expected at least one Deployment for the agent run")
	}
	t.Logf("Found %d Deployment(s) for run %s", len(deployments.Items), runID)

	// Verify PVC exists.
	var pvcs corev1.PersistentVolumeClaimList
	if err := k8s.List(ctx, &pvcs, client.InNamespace(ns), labelSelector(runID)); err != nil {
		t.Fatalf("List PVCs: %v", err)
	}
	if len(pvcs.Items) == 0 {
		t.Fatal("Expected at least one PVC for the agent run")
	}
	t.Logf("Found %d PVC(s) for run %s", len(pvcs.Items), runID)

	// Wait for terminal phase.
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Workflow completed: phase=%s", terminal.Phase)

	// Give the controller time to sync.
	time.Sleep(10 * time.Second)

	// Verify Deployment still exists with replicas=0.
	if err := k8s.List(ctx, &deployments, client.InNamespace(ns), labelSelector(runID)); err != nil {
		t.Fatalf("List Deployments after completion: %v", err)
	}
	if len(deployments.Items) == 0 {
		t.Fatal("Deployment was deleted after completion; expected it to persist with replicas=0")
	}
	replicas := int32(1)
	if deployments.Items[0].Spec.Replicas != nil {
		replicas = *deployments.Items[0].Spec.Replicas
	}
	if replicas != 0 {
		t.Errorf("Expected Deployment replicas=0 after completion, got %d", replicas)
	}

	// Verify PVC still exists.
	if err := k8s.List(ctx, &pvcs, client.InNamespace(ns), labelSelector(runID)); err != nil {
		t.Fatalf("List PVCs after completion: %v", err)
	}
	if len(pvcs.Items) == 0 {
		t.Fatal("PVC was deleted after completion; expected it to persist")
	}
	t.Log("Deployment lifecycle verified: Deployment replicas=0 and PVC persists after completion")
}

// ---------- 16.2: TestE2E_PersistentFiles ----------

func TestE2E_PersistentFiles(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for completion.
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Run completed: phase=%s", terminal.Phase)

	// Give the API server time to detect the scaled-down Deployment.
	time.Sleep(10 * time.Second)

	// GET /api/v1/runs/{id}/files?path=/workspace — should return entries from disk.
	url := fmt.Sprintf("%s/api/v1/runs/%s/files?path=/workspace", apiBaseURL(), runID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response status=%d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Parse JSON: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Error("Expected at least one file entry in /workspace listing from disk")
	} else {
		t.Logf("Listed %d entries from persistent workspace", len(result.Entries))
	}
}

// ---------- 16.3: TestE2E_PersistentLogs ----------

func TestE2E_PersistentLogs(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for completion.
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Run completed: phase=%s", terminal.Phase)

	// Give the API server time to detect the scaled-down Deployment.
	time.Sleep(10 * time.Second)

	// GET /api/v1/runs/{id}/logs — should return content from disk.
	url := fmt.Sprintf("%s/api/v1/runs/%s/logs", apiBaseURL(), runID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response status=%d body_len=%d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		t.Error("Expected non-empty log content from persistent workspace")
	} else {
		t.Logf("Retrieved %d bytes of logs from persistent workspace", len(body))
	}
}

// ---------- 16.4: TestE2E_DebugPod ----------

func TestE2E_DebugPod(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()
	ns := aotNamespace()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for completion.
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Run completed: phase=%s", terminal.Phase)

	time.Sleep(10 * time.Second)

	// POST /api/v1/runs/{id}/debug — start debug session.
	debugURL := fmt.Sprintf("%s/api/v1/runs/%s/debug", apiBaseURL(), runID)
	req, _ := http.NewRequest(http.MethodPost, debugURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /debug: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 from POST /debug, got %d", resp.StatusCode)
	}
	t.Log("Debug session started")

	// Verify CRD has debugActive=true.
	time.Sleep(5 * time.Second)
	ar := &aotv1alpha1.AgentRun{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: runID, Namespace: ns}, ar); err != nil {
		t.Fatalf("Get AgentRun: %v", err)
	}
	if !ar.Status.DebugActive {
		t.Error("Expected debugActive=true after POST /debug")
	}

	// DELETE /api/v1/runs/{id}/debug — stop debug session.
	req, _ = http.NewRequest(http.MethodDelete, debugURL, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /debug: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 from DELETE /debug, got %d", resp.StatusCode)
	}
	t.Log("Debug session stopped")

	// Verify CRD has debugActive=false.
	time.Sleep(5 * time.Second)
	if err := k8s.Get(ctx, types.NamespacedName{Name: runID, Namespace: ns}, ar); err != nil {
		t.Fatalf("Get AgentRun after debug stop: %v", err)
	}
	if ar.Status.DebugActive {
		t.Error("Expected debugActive=false after DELETE /debug")
	}
	t.Log("Debug pod lifecycle verified")
}

// ---------- 16.5: TestE2E_Traces ----------

func TestE2E_Traces(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for completion.
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Run completed: phase=%s", terminal.Phase)

	time.Sleep(10 * time.Second)

	// GET /api/v1/runs/{id}/traces — should return JSON array.
	url := fmt.Sprintf("%s/api/v1/runs/%s/traces", apiBaseURL(), runID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response status=%d body_len=%d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Verify it is a valid JSON array (may be empty if no tool calls happened).
	var spans []json.RawMessage
	if err := json.Unmarshal(body, &spans); err != nil {
		t.Fatalf("Expected JSON array from /traces, got parse error: %v", err)
	}
	t.Logf("Retrieved %d trace spans", len(spans))
}

// ---------- 16.6: TestE2E_TraceDiff ----------

func TestE2E_TraceDiff(t *testing.T) {
	// Skip: requires a run that made tool calls with file modifications to produce
	// meaningful diffs. This would need a more complex agent prompt and verification
	// that specific spans exist before fetching their diffs.
	t.Skip("Skipping TraceDiff test: requires a run that made tool calls with file modifications")
}

// ---------- 16.7: TestE2E_DevcontainerJson ----------

func TestE2E_DevcontainerJson(t *testing.T) {
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runID := createRunViaAPI(t, "Create a file called DONE.txt containing PASS", 300, 0)

	// Wait for completion (ensures hydration has run).
	workflowID := fmt.Sprintf("agentrun-%s", runID)
	_ = waitForRunningPod(ctx, t, runID, 90*time.Second)
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Run completed: phase=%s", terminal.Phase)

	time.Sleep(10 * time.Second)

	// GET /api/v1/runs/{id}/files/content?path=/workspace/.devcontainer/devcontainer.json
	url := fmt.Sprintf("%s/api/v1/runs/%s/files/content?path=/workspace/.devcontainer/devcontainer.json", apiBaseURL(), runID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response status=%d body_len=%d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Verify it is valid JSON.
	var devcontainer map[string]interface{}
	if err := json.Unmarshal(body, &devcontainer); err != nil {
		t.Fatalf("Expected valid JSON from devcontainer.json, got parse error: %v", err)
	}
	t.Logf("devcontainer.json is valid JSON with %d keys", len(devcontainer))
}

// waitForTerminalPhase polls the Temporal workflow until it reaches a terminal phase.
// This is a local helper that wraps the temporal query-based waiting.
func waitForTerminalPhaseViaK8s(ctx context.Context, t *testing.T, k8s client.Client, runID, ns string, timeout time.Duration) *aotv1alpha1.AgentRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ar := &aotv1alpha1.AgentRun{}
		if err := k8s.Get(ctx, types.NamespacedName{Name: runID, Namespace: ns}, ar); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if ar.Status.Phase == aotv1alpha1.AgentRunPhaseSucceeded ||
			ar.Status.Phase == aotv1alpha1.AgentRunPhaseFailed ||
			ar.Status.Phase == aotv1alpha1.AgentRunPhaseCancelled {
			return ar
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("Timed out waiting for terminal phase on %s", runID)
	return nil
}
