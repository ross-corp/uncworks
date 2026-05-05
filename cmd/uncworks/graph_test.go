package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// captureStdout temporarily replaces os.Stdout and returns the captured string.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintGraphSingleNode(t *testing.T) {
	graph := &apiv1.RunGraph{
		Nodes: []*apiv1.RunGraphNode{
			{Name: "run-abc", Phase: apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED, Role: "orchestrator"},
		},
	}
	out := captureStdout(func() { printGraph("run-abc", graph) })
	if !strings.Contains(out, "▶ run-abc (orchestrator) [DONE]") {
		t.Errorf("unexpected output for single node:\n%s", out)
	}
}

func TestPrintGraphTree(t *testing.T) {
	// root → child1 → grandchild
	//       └─ child2
	graph := &apiv1.RunGraph{
		Nodes: []*apiv1.RunGraphNode{
			{Name: "root", Phase: apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING},
			{Name: "child1", Phase: apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED},
			{Name: "child2", Phase: apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING},
			{Name: "grandchild", Phase: apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING},
		},
		Edges: []*apiv1.RunGraphEdge{
			{Parent: "root", Child: "child1"},
			{Parent: "root", Child: "child2"},
			{Parent: "child1", Child: "grandchild"},
		},
	}
	out := captureStdout(func() { printGraph("root", graph) })
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	wantLines := []string{
		"▶ root [RUNNING]",
		"  ├─ child1 [DONE]",
		"  │  └─ grandchild [PENDING]",
		"  └─ child2 [RUNNING]",
	}
	if len(lines) != len(wantLines) {
		t.Fatalf("expected %d lines, got %d:\n%s", len(wantLines), len(lines), out)
	}
	for i, want := range wantLines {
		if lines[i] != want {
			t.Errorf("line %d: got %q, want %q", i, lines[i], want)
		}
	}
}
