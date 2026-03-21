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

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestSmoke_Files_DuringRun creates a single-mode run, waits until it is RUNNING,
// then hits the REST file-listing endpoint and verifies:
//   - The entries array is non-empty.
//   - No entry is named ".aot" or ".bare" (internal directories must be hidden).
func TestSmoke_Files_DuringRun(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a single-mode run.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            "Create a file called SMOKE_FILES.txt containing 'hello'",
			TtlSeconds:        300,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SINGLE,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created single-mode run for file smoke test: %s", runID)

	// Poll until the run reaches RUNNING phase.
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for run to reach RUNNING phase")
		default:
		}

		getResp, err := apiClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		phase := getResp.Msg.Status.Phase
		if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING {
			t.Logf("Run %s is now RUNNING", runID)
			break
		}
		if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED {
			t.Fatalf("Run reached terminal phase %v before RUNNING", phase)
		}

		time.Sleep(3 * time.Second)
	}

	// Give the workspace a moment to hydrate.
	time.Sleep(5 * time.Second)

	// GET /api/v1/runs/{id}/files?path=/workspace
	filesURL := fmt.Sprintf("%s/api/v1/runs/%s/files?path=/workspace", apiBaseURL(), runID)
	httpResp, err := http.Get(filesURL)
	if err != nil {
		t.Fatalf("GET %s: %v", filesURL, err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", httpResp.StatusCode, string(body))
	}

	var result struct {
		Entries []struct {
			Name string `json:"name"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Parse JSON: %v (body: %s)", err, string(body))
	}

	if len(result.Entries) == 0 {
		t.Fatal("Expected non-empty entries array in /workspace listing")
	}
	t.Logf("Listed %d entries in /workspace", len(result.Entries))

	// Assert no internal directories are exposed.
	for _, entry := range result.Entries {
		if entry.Name == ".aot" || entry.Name == ".bare" {
			t.Errorf("Internal directory %q should not appear in file listing", entry.Name)
		}
	}
}
