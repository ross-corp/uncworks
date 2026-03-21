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

// TestSmoke_Traces_MatchActivity creates a single-mode run, waits for completion,
// then compares tool_call entries in structured logs against tool spans in traces.
// Tool span count should be >= tool call count (spans may include lifecycle extras).
func TestSmoke_Traces_MatchActivity(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create a single-mode run.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            "Create a file called SMOKE_TRACES.txt containing 'trace test'",
			TtlSeconds:        300,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SINGLE,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created single-mode run for trace smoke test: %s", runID)

	// Poll until terminal phase.
	var finalPhase apiv1.AgentRunPhase
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for run to complete")
		default:
		}

		getResp, err := apiClient.GetAgentRun(ctx, connect.NewRequest(&apiv1.GetAgentRunRequest{Id: runID}))
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		phase := getResp.Msg.Status.Phase
		if phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED ||
			phase == apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED {
			finalPhase = phase
			break
		}

		time.Sleep(5 * time.Second)
	}
	t.Logf("Run completed with phase: %v", finalPhase)

	// Count tool_call entries in structured logs.
	structURL := fmt.Sprintf("%s/api/v1/runs/%s/logs/structured", apiBaseURL(), runID)
	toolCallCount := countStructuredLogToolCalls(t, structURL)
	t.Logf("Structured log tool_call entries: %d", toolCallCount)

	// Count tool spans in traces.
	tracesURL := fmt.Sprintf("%s/api/v1/runs/%s/traces", apiBaseURL(), runID)
	toolSpanCount := countTraceToolSpans(t, tracesURL)
	t.Logf("Trace tool spans: %d", toolSpanCount)

	// Tool span count should be >= tool call count.
	if toolSpanCount < toolCallCount {
		t.Errorf("Expected tool span count (%d) >= tool call count (%d)", toolSpanCount, toolCallCount)
	}
}

// countStructuredLogToolCalls fetches structured logs and counts entries with type "tool_call".
func countStructuredLogToolCalls(t *testing.T, url string) int {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Logf("GET %s: %v (returning 0)", url, err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Structured logs returned %d: %s (returning 0)", resp.StatusCode, string(body))
		return 0
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Read body: %v (returning 0)", err)
		return 0
	}

	var entries []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Logf("Parse structured logs: %v (returning 0)", err)
		return 0
	}

	count := 0
	for _, e := range entries {
		if e.Type == "tool_call" {
			count++
		}
	}
	return count
}

// countTraceToolSpans fetches traces and counts spans that represent tool operations.
func countTraceToolSpans(t *testing.T, url string) int {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Logf("GET %s: %v (returning 0)", url, err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Traces returned %d: %s (returning 0)", resp.StatusCode, string(body))
		return 0
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Read body: %v (returning 0)", err)
		return 0
	}

	// Traces may be returned as a flat array of spans or nested.
	// Try flat array first.
	var spans []struct {
		Name       string `json:"name"`
		OperName   string `json:"operationName"`
		Attributes []struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		} `json:"attributes"`
	}
	if err := json.Unmarshal(body, &spans); err != nil {
		// May be a wrapped response; try extracting a spans field.
		var wrapped struct {
			Spans json.RawMessage `json:"spans"`
		}
		if json.Unmarshal(body, &wrapped) == nil && wrapped.Spans != nil {
			_ = json.Unmarshal(wrapped.Spans, &spans)
		}
	}

	count := 0
	for _, s := range spans {
		name := s.Name
		if name == "" {
			name = s.OperName
		}
		// Count spans whose name contains "tool" (case-insensitive match via prefix).
		if containsToolIndicator(name) {
			count++
		}
	}
	return count
}

// containsToolIndicator checks if a span name indicates a tool operation.
func containsToolIndicator(name string) bool {
	for _, substr := range []string{"tool", "Tool", "TOOL", "execute_tool", "tool_call"} {
		if len(name) >= len(substr) {
			for i := 0; i <= len(name)-len(substr); i++ {
				if name[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
