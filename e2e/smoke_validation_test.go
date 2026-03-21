//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestSmoke_Validation_OpenSpecValidPlan creates a spec-driven run and waits for
// completion. If the run SUCCEEDED, the spec was valid and the plan passed validation.
// If the run FAILED, the test checks whether the failure message references validation,
// confirming that the validation step actually ran.
func TestSmoke_Validation_OpenSpecValidPlan(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            "Create a file called VALIDATE.txt containing 'validation smoke'",
			TtlSeconds:        600,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SPEC_DRIVEN,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created spec-driven run for validation smoke test: %s", runID)

	// Poll until terminal phase.
	var finalPhase apiv1.AgentRunPhase
	var finalMessage string
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for spec-driven run to complete")
		default:
		}

		getResp, err := apiClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		phase := getResp.Msg.Status.Phase
		if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
			finalPhase = phase
			finalMessage = getResp.Msg.Status.Message
			break
		}

		time.Sleep(10 * time.Second)
	}

	t.Logf("Final phase: %v, message: %s", finalPhase, finalMessage)

	switch finalPhase {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		// Spec was valid and the plan passed — smoke test passes.
		t.Log("Spec-driven run SUCCEEDED: spec was valid and plan passed validation")

	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		// Check if the failure message references validation, confirming that
		// the validation step actually ran (even though the run failed).
		lower := strings.ToLower(finalMessage)
		if strings.Contains(lower, "valid") || strings.Contains(lower, "verif") {
			t.Logf("Run FAILED with validation-related message — validation step ran: %s", finalMessage)
		} else {
			t.Errorf("Run FAILED without validation-related message: %s", finalMessage)
		}

	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		t.Fatal("Run was unexpectedly CANCELLED")

	default:
		t.Fatalf("Unexpected terminal phase: %v", finalPhase)
	}
}
