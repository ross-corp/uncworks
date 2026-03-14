//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// TestE2E_SpecDrivenRun creates an AgentRun with SpecContent and no Prompt,
// verifying that spec-driven runs are accepted and can start workflows.
func TestE2E_SpecDrivenRun(t *testing.T) {
	k8s := getE2EClient(t)
	tc := getTemporalClient(t)
	defer tc.Close()
	ctx := context.Background()

	runName := fmt.Sprintf("e2e-spec-driven-%d", time.Now().Unix())

	ar := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend:     aotv1alpha1.BackendPod,
			Repos:       []aotv1alpha1.Repository{{URL: getSoftServeRepoURL("e2e-repo"), Branch: "main"}},
			Prompt:      "",
			SpecContent: "# TestSpec\nCreate DONE.txt with PASS",
			TTLSeconds:  300,
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
	t.Logf("Spec-driven workflow started: %s", workflowID)

	// Wait for terminal phase
	terminal := waitForTerminalPhase(ctx, t, tc, workflowID, 5*time.Minute)
	t.Logf("Spec-driven workflow completed: phase=%s message=%s", terminal.Phase, terminal.Message)
}
