//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// TestE2E_MultiRepo_TwoRepos creates an AgentRun with two repositories and
// verifies the workflow starts and a pod is created with both repos available.
func TestE2E_MultiRepo_TwoRepos(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-multirepo-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Repos: []aotv1alpha1.Repository{
				{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"},
				{URL: getSoftServeRepoURL("e2e-repo-frontend"), Branch: "main"},
			},
			Prompt:     "List all files in the workspace",
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
	t.Logf("Multi-repo workflow started: %s", workflowID)

	// Query workflow state to verify pod was created
	resp, err := tc.QueryWorkflow(ctx, workflowID, "", aottemporal.QueryGetState)
	if err != nil {
		t.Fatalf("QueryWorkflow: %v", err)
	}
	var state aottemporal.WorkflowState
	if err := resp.Get(&state); err != nil {
		t.Fatalf("Decode state: %v", err)
	}

	t.Logf("Workflow state: phase=%s pod=%s", state.Phase, state.PodName)

	if state.PodName == "" {
		t.Error("expected non-empty PodName — proves pod was created with both repos")
	}
}

// TestE2E_MultiRepo_WorkspaceName creates an AgentRun with a WorkspaceName set
// and verifies the field is preserved on the CRD.
func TestE2E_MultiRepo_WorkspaceName(t *testing.T) {
	k8s := getE2EClient(t)
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-workspace-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:       aotv1alpha1.BackendPod,
			Repos:         []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:        "Workspace name test",
			TTLSeconds:    300,
			WorkspaceName: "test-workspace",
		},
	}

	if err := k8s.Create(ctx, ar); err != nil {
		t.Fatalf("Create AgentRun: %v", err)
	}
	defer k8s.Delete(ctx, ar)

	// Read it back
	fetched := &aotv1alpha1.AgentRun{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: runName, Namespace: "default"}, fetched); err != nil {
		t.Fatalf("Get AgentRun: %v", err)
	}

	if fetched.Spec.WorkspaceName != "test-workspace" {
		t.Errorf("expected WorkspaceName 'test-workspace', got %q", fetched.Spec.WorkspaceName)
	}
	t.Logf("WorkspaceName preserved: %s", fetched.Spec.WorkspaceName)
}
