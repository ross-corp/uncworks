//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestSmoke_SpecDrivenPipeline_Succeeds creates a spec-driven run with a trivial
// prompt and polls until it reaches a terminal phase, asserting SUCCEEDED.
func TestSmoke_SpecDrivenPipeline_Succeeds(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            "Create a file called SMOKE.txt containing 'smoke test pass'",
			TtlSeconds:        600,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created spec-driven smoke run: %s", runID)

	// Poll until SUCCEEDED or FAILED (up to 15 min context timeout).
	var finalPhase apiv1.AgentRunPhase
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for spec-driven run to complete")
		default:
		}

		getResp, err := apiClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
		if err != nil {
			t.Logf("GetAgentRun error (will retry): %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		phase := getResp.Msg.Status.Phase
		stage := getResp.Msg.Status.Stage
		t.Logf("Phase: %v, Stage: %s", phase, stage)

		if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
			finalPhase = phase
			break
		}

		time.Sleep(10 * time.Second)
	}

	if finalPhase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED {
		t.Fatalf("Expected SUCCEEDED, got %v", finalPhase)
	}

	t.Logf("Spec-driven smoke run SUCCEEDED: %s", runID)
}
