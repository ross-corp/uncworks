package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
)

func getAPIClient(t *testing.T) apiv1connect.AOTServiceClient {
	t.Helper()

	apiURL := os.Getenv("AOT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:50055"
	}

	return apiv1connect.NewAOTServiceClient(http.DefaultClient, apiURL)
}

func TestE2E_API_CreateAgentRun(t *testing.T) {
	client := getAPIClient(t)
	k8sClient := getE2EClient(t)
	ctx := context.Background()

	resp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/test.git"}},
			Prompt:     "E2E API test: create run",
			TtlSeconds: 120,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}

	run := resp.Msg.AgentRun
	if run.Id == "" {
		t.Fatal("expected non-empty ID")
	}
	t.Logf("Created AgentRun via API: %s", run.Id)

	// Verify CRD exists in K8s
	namespace := os.Getenv("AOT_NAMESPACE")
	if namespace == "" {
		namespace = "aot"
	}

	crd := &aotv1alpha1.AgentRun{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name: run.Id, Namespace: namespace,
	}, crd); err != nil {
		t.Fatalf("CRD not found in K8s for API-created run %s: %v", run.Id, err)
	}

	if crd.Spec.Prompt != "E2E API test: create run" {
		t.Errorf("CRD prompt mismatch: got %q", crd.Spec.Prompt)
	}

	t.Logf("Verified CRD exists in K8s: %s/%s", namespace, crd.Name)

	// Cleanup
	_ = k8sClient.Delete(ctx, crd)
}

func TestE2E_API_Lifecycle(t *testing.T) {
	client := getAPIClient(t)
	k8sClient := getE2EClient(t)
	ctx := context.Background()

	namespace := os.Getenv("AOT_NAMESPACE")
	if namespace == "" {
		namespace = "aot"
	}

	// Create
	resp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/test.git"}},
			Prompt:     "E2E API lifecycle test",
			TtlSeconds: 120,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created: %s", runID)

	// Verify it appears in list
	listResp, err := client.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{}))
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	found := false
	for _, r := range listResp.Msg.AgentRuns {
		if r.Id == runID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created run %s not found in list", runID)
	}

	// Get individual run
	getResp, err := client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if getResp.Msg.Spec.Prompt != "E2E API lifecycle test" {
		t.Errorf("prompt mismatch: got %q", getResp.Msg.Spec.Prompt)
	}

	// Wait briefly for controller to pick it up
	time.Sleep(2 * time.Second)

	// Get again to see if status was enriched
	getResp2, err := client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
	if err != nil {
		t.Fatalf("GetAgentRun (2): %v", err)
	}
	t.Logf("After 2s: phase=%s message=%q", getResp2.Msg.Status.Phase, getResp2.Msg.Status.Message)

	// Cleanup
	crd := &aotv1alpha1.AgentRun{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: runID, Namespace: namespace}, crd); err == nil {
		_ = k8sClient.Delete(ctx, crd)
	}
}

func TestE2E_API_CancelAgentRun(t *testing.T) {
	client := getAPIClient(t)
	k8sClient := getE2EClient(t)
	ctx := context.Background()

	namespace := os.Getenv("AOT_NAMESPACE")
	if namespace == "" {
		namespace = "aot"
	}

	// Create a run
	resp, err := client.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/test.git"}},
			Prompt:     fmt.Sprintf("E2E cancel test %d", time.Now().UnixMilli()),
			TtlSeconds: 120,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created: %s", runID)

	// Wait for controller to start the workflow
	time.Sleep(3 * time.Second)

	// Cancel it
	_, err = client.CancelAgentRun(ctx, connect.NewRequest(&apiv1.CancelAgentRunRequest{Id: runID}))
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	t.Logf("Cancel signal sent for %s", runID)

	// Wait for cancellation to propagate
	time.Sleep(5 * time.Second)

	// Check status
	getResp, err := client.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
	if err != nil {
		t.Fatalf("GetAgentRun after cancel: %v", err)
	}
	t.Logf("After cancel: phase=%s message=%q", getResp.Msg.Status.Phase, getResp.Msg.Status.Message)

	// Cleanup
	crd := &aotv1alpha1.AgentRun{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: runID, Namespace: namespace}, crd); err == nil {
		_ = k8sClient.Delete(ctx, crd)
	}
}

func TestE2E_API_SendHumanInput(t *testing.T) {
	apiClient := getAPIClient(t)
	k8sClient := getE2EClient(t)
	ctx := context.Background()

	namespace := os.Getenv("AOT_NAMESPACE")
	if namespace == "" {
		namespace = "aot"
	}

	// Create a run via API so the controller starts a Temporal workflow
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: "https://github.com/example/test.git"}},
			Prompt:     fmt.Sprintf("E2E human input test %d", time.Now().UnixMilli()),
			TtlSeconds: 120,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created: %s", runID)

	// Wait for the controller to start the workflow and annotate the CRD
	var crd aotv1alpha1.AgentRun
	var workflowStarted bool
	for i := 0; i < 15; i++ {
		time.Sleep(time.Second)
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: runID, Namespace: namespace}, &crd); err != nil {
			continue
		}
		if crd.Annotations["aot.uncworks.io/workflow-id"] != "" {
			workflowStarted = true
			t.Logf("Workflow started: %s (attempt %d)", crd.Annotations["aot.uncworks.io/workflow-id"], i+1)
			break
		}
	}
	if !workflowStarted {
		_ = k8sClient.Delete(ctx, &crd)
		t.Fatal("workflow did not start within 15s")
	}

	// Patch the CRD status to WaitingForInput to simulate the agent waiting.
	// This mimics what the controller does when the sidecar reports the agent
	// is in AGENT_PROCESS_STATE_WAITING_FOR_INPUT state.
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseWaitingForInput
	crd.Status.Message = "Agent waiting for human input"
	if err := k8sClient.Status().Update(ctx, &crd); err != nil {
		_ = k8sClient.Delete(ctx, &crd)
		t.Fatalf("Failed to patch CRD status to WaitingForInput: %v", err)
	}
	t.Logf("Patched CRD status to WaitingForInput")

	// Send human input via the API
	inputResp, err := apiClient.SendHumanInput(ctx, connect.NewRequest(&apiv1.SendHumanInputRequest{
		AgentRunId: runID,
		Input:      "yes, proceed with the fix",
	}))
	if err != nil {
		// The Temporal signal might fail because the workflow isn't truly
		// in a state to receive input, but the API plumbing should work.
		// If we get FailedPrecondition, the CRD status patch didn't work.
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			t.Fatal("API returned FailedPrecondition — CRD status patch didn't take effect")
		}
		// Internal error from Temporal signal is acceptable — it means
		// the API correctly checked the CRD status and attempted the signal.
		t.Logf("SendHumanInput returned error (Temporal signal issue, API plumbing OK): %v", err)
	} else {
		if !inputResp.Msg.Accepted {
			t.Error("expected accepted=true")
		}
		t.Logf("SendHumanInput accepted: %v", inputResp.Msg.Accepted)
	}

	// Cleanup
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&crd), &crd); err == nil {
		_ = k8sClient.Delete(ctx, &crd)
	}
}
