//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/websocket"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestSmoke_Shell_WebSocketUpgrade creates a single-mode run, waits for RUNNING,
// then sends a WebSocket upgrade request to the exec endpoint and asserts that
// the server responds with 101 Switching Protocols.
func TestSmoke_Shell_WebSocketUpgrade(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a single-mode run.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            "Create a file called SMOKE_SHELL.txt containing 'hello'",
			TtlSeconds:        300,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SINGLE,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created single-mode run for shell smoke test: %s", runID)

	// Poll until RUNNING.
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

	// Allow workspace to settle.
	time.Sleep(3 * time.Second)

	// Build WebSocket URL from the API base URL.
	base := apiBaseURL()
	wsURL := strings.Replace(base, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/v1/runs/%s/exec", wsURL, runID)

	t.Logf("Attempting WebSocket upgrade to %s", wsURL)

	conn, httpResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// If we got an HTTP response, check if it was an upgrade attempt that failed.
		if httpResp != nil {
			t.Fatalf("WebSocket dial failed with HTTP %d: %v", httpResp.StatusCode, err)
		}
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer conn.Close()

	// If Dial succeeded, the server responded with 101 Switching Protocols.
	t.Log("WebSocket upgrade succeeded (101 Switching Protocols)")

	// Graceful close.
	if err := conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Logf("WebSocket close write: %v (non-fatal)", err)
	}
}
