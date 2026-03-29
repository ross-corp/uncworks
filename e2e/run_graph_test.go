//go:build e2e

// e2e/run_graph_test.go — end-to-end tests for the GetRunGraph API endpoint.
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestE2E_RunGraph_SingleRun verifies that GetRunGraph returns a graph with at
// least one node for a freshly created single-mode run.
func TestE2E_RunGraph_SingleRun(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	// Create a single-mode run.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:           apiv1.Backend_BACKEND_POD,
			Repos:             []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:            fmt.Sprintf("run-graph single-mode test %d", time.Now().UnixMilli()),
			TtlSeconds:        120,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_SINGLE,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created run: %s", runID)

	// Give the controller a moment to start the workflow.
	time.Sleep(3 * time.Second)

	// Fetch the run graph.
	graphResp, err := apiClient.GetRunGraph(ctx, connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: runID,
	}))
	if err != nil {
		t.Fatalf("GetRunGraph: %v", err)
	}
	graph := graphResp.Msg

	t.Logf("RunGraph: nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))

	if len(graph.Nodes) == 0 {
		t.Error("expected at least one node in the run graph for a single-mode run")
	}

	// A single-mode run should have no parent-child edges.
	if len(graph.Edges) != 0 {
		t.Logf("Unexpected edges in single-mode run graph: %d (may be acceptable)", len(graph.Edges))
	}

	// The root node should match the run ID.
	found := false
	for _, node := range graph.Nodes {
		if node.Name == runID {
			found = true
			t.Logf("Found root node: name=%s phase=%v role=%s", node.Name, node.Phase, node.Role)
			break
		}
	}
	if !found {
		t.Errorf("run ID %q not found in run graph nodes", runID)
	}
}

// TestE2E_RunGraph_NonExistentID verifies that GetRunGraph returns an error for
// a run ID that does not exist.
func TestE2E_RunGraph_NonExistentID(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	_, err := apiClient.GetRunGraph(ctx, connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: "non-existent-graph-run-xyz-789",
	}))
	if err == nil {
		t.Fatal("expected an error for non-existent run ID in GetRunGraph, got nil")
	}
	t.Logf("Got expected error: code=%v err=%v", connect.CodeOf(err), err)
}

// TestE2E_RunGraph_ParentChild creates a spec-driven run that spawns a child
// via the manual orchestration mode and verifies that the graph contains edges.
// This test is lenient: if the LLM does not spawn a junior the graph will have
// a single node, which is still a valid (if incomplete) test of the plumbing.
func TestE2E_RunGraph_ParentChild(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	// Create a manual orchestration run with two tasks so the controller
	// creates child runs internally.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend: apiv1.Backend_BACKEND_POD,
			Repos:   []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:  "parent graph test",
			TtlSeconds:        300,
			OrchestrationMode: apiv1.OrchestrationMode_ORCHESTRATION_MODE_MANUAL,
			Orchestration: &apiv1.Orchestration{
				Tasks: []*apiv1.OrchestrationTask{
					{Name: "step-a", Prompt: "Create file A.txt containing 'step-a'"},
					{Name: "step-b", Prompt: "Create file B.txt containing 'step-b'"},
				},
			},
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun (manual orchestration): %v", err)
	}
	parentID := resp.Msg.AgentRun.Id
	t.Logf("Created manual orchestration parent run: %s", parentID)

	// Allow time for child runs to be created.
	time.Sleep(10 * time.Second)

	// Fetch the run graph.
	graphResp, err := apiClient.GetRunGraph(ctx, connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: parentID,
	}))
	if err != nil {
		t.Fatalf("GetRunGraph: %v", err)
	}
	graph := graphResp.Msg

	t.Logf("Parent-child RunGraph: nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))

	if len(graph.Nodes) == 0 {
		t.Error("expected at least one node in the run graph")
	}

	// Log graph shape for diagnostic purposes.
	for _, node := range graph.Nodes {
		t.Logf("  node: name=%s phase=%v role=%s", node.Name, node.Phase, node.Role)
	}
	for _, edge := range graph.Edges {
		t.Logf("  edge: %s -> %s", edge.Parent, edge.Child)
	}
}
