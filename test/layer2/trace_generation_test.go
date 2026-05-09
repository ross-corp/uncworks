// test/layer2/trace_generation_test.go — Layer 2 tests for trace-related observable behavior.
//
// Tracing in this system works at two levels:
//   1. The AgentRunStatus.TraceID field — an OpenTelemetry trace ID stored on the
//      CRD status and surfaced via GetAgentRun.
//   2. The AgentRunStatus.Stage field — the current pipeline stage (planning,
//      executing, verifying) that maps to stage child spans in the agent's trace.
//   3. The GetRunGraph RPC — returns the tree of parent/child runs built from the
//      spec-run-id label and ParentRunID spec field, which mirrors the span hierarchy.
//
// These tests verify observable API behavior only; they do not access the JSONL
// spans file directly (that is tested in internal/server/traces_test.go).
package layer2

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/test/testutil"
)

// TestTraceGeneration_TraceIDSurfacedViaAPI verifies that the TraceID set on
// the CRD status (i.e., by the controller after span creation) is returned
// by GetAgentRun.
func TestTraceGeneration_TraceIDSurfacedViaAPI(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Generate trace for this run"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	// Simulate the controller writing the root span's trace ID to the CRD status.
	const fakeTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	crd.Status.TraceID = fakeTraceID
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	assert.Equal(t, fakeTraceID, getResp.Msg.Status.TraceId,
		"TraceID written to CRD status must be returned by GetAgentRun")
}

// TestTraceGeneration_StageSurfacedViaAPI verifies that the Stage field on the
// CRD status (set by the controller as pipeline stage child spans begin) is
// returned by GetAgentRun.
func TestTraceGeneration_StageSurfacedViaAPI(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("spec-driven run with stages"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	stages := []string{"planning", "executing", "verifying"}

	for _, stage := range stages {
		crd := &aotv1alpha1.AgentRun{}
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: testutil.DefaultNamespace,
			Name:      runID,
		}, crd))
		crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
		crd.Status.Stage = stage
		require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

		getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
			Id: runID,
		}))
		require.NoError(t, err)
		assert.Equal(t, stage, getResp.Msg.Status.Stage,
			"Stage %q must be surfaced by GetAgentRun", stage)
	}
}

// TestTraceGeneration_RootSpan_SingleNode verifies that GetRunGraph returns a
// single-node graph for a standalone run (root span with no children).
func TestTraceGeneration_RootSpan_SingleNode(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Standalone run"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	graphResp, err := c.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: runID,
	}))
	require.NoError(t, err)

	graph := graphResp.Msg
	require.Len(t, graph.Nodes, 1, "standalone run must produce a single-node graph")
	assert.Equal(t, runID, graph.Nodes[0].Name,
		"the single node must be the root run")
	assert.Empty(t, graph.Edges, "standalone run must have no edges")
}

// TestTraceGeneration_ParentChildRelationship verifies that GetRunGraph correctly
// represents the parent-child span relationship between a senior run and its
// junior sub-run.  This mirrors how stage child spans are structured: the parent
// run is the root span and each junior run represents a child span in the trace.
func TestTraceGeneration_ParentChildRelationship(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	const specRunID = "spec-run-abc123"

	// Create the parent (senior) run.
	parentResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:   apiv1.Backend_BACKEND_POD,
			Repos:     testutil.DefaultRepo(),
			Prompt:    "Orchestrate sub-tasks",
			SpecRunId: specRunID,
		},
	}))
	require.NoError(t, err)
	parentID := parentResp.Msg.AgentRun.Id

	// Tag the parent with the spec-run-id label so GetRunGraph can find it.
	parentCRD := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      parentID,
	}, parentCRD))
	if parentCRD.Labels == nil {
		parentCRD.Labels = make(map[string]string)
	}
	parentCRD.Labels["aot.uncworks.io/spec-run-id"] = specRunID
	parentCRD.Labels["aot.uncworks.io/run-role"] = "senior"
	require.NoError(t, k8sClient.Update(context.Background(), parentCRD))

	// Create a child (junior) run linked to the parent.
	childResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:     apiv1.Backend_BACKEND_POD,
			Repos:       testutil.DefaultRepo(),
			Prompt:      "Execute sub-task",
			ParentRunId: parentID,
			SpecRunId:   specRunID,
		},
	}))
	require.NoError(t, err)
	childID := childResp.Msg.AgentRun.Id

	// Tag the child with the spec-run-id label.
	childCRD := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      childID,
	}, childCRD))
	if childCRD.Labels == nil {
		childCRD.Labels = make(map[string]string)
	}
	childCRD.Labels["aot.uncworks.io/spec-run-id"] = specRunID
	childCRD.Labels["aot.uncworks.io/run-role"] = "junior"
	require.NoError(t, k8sClient.Update(context.Background(), childCRD))

	// GetRunGraph via the parent ID — must return both nodes and the edge.
	graphResp, err := c.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: parentID,
	}))
	require.NoError(t, err)

	graph := graphResp.Msg
	require.Len(t, graph.Nodes, 2, "graph must contain parent and child nodes")
	require.Len(t, graph.Edges, 1, "graph must contain exactly one parent→child edge")

	edge := graph.Edges[0]
	assert.Equal(t, parentID, edge.Parent, "edge.Parent must be the senior run ID")
	assert.Equal(t, childID, edge.Child, "edge.Child must be the junior run ID")
}

// TestTraceGeneration_StartedAt_SurfacedViaAPI verifies that the StartedAt
// timestamp (written when the root span begins) is returned by GetAgentRun.
func TestTraceGeneration_StartedAt_SurfacedViaAPI(t *testing.T) {
	c, k8sClient, cleanup := newTestServer(t)
	defer cleanup()

	createResp, err := c.CreateAgentRun(context.Background(), connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: testutil.MinimalSpec("Timed run"),
	}))
	require.NoError(t, err)
	runID := createResp.Msg.AgentRun.Id

	startedAt := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	crd := &aotv1alpha1.AgentRun{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: testutil.DefaultNamespace,
		Name:      runID,
	}, crd))
	crd.Status.Phase = aotv1alpha1.AgentRunPhaseRunning
	crd.Status.StartedAt = &metav1.Time{Time: startedAt}
	require.NoError(t, k8sClient.Status().Update(context.Background(), crd))

	getResp, err := c.GetAgentRun(context.Background(), connect.NewRequest(&apiv1.GetAgentRunRequest{
		Id: runID,
	}))
	require.NoError(t, err)
	require.NotNil(t, getResp.Msg.Status.StartedAt,
		"StartedAt must be present after the controller sets it")
	assert.Equal(t, startedAt.Unix(), getResp.Msg.Status.StartedAt.AsTime().Unix(),
		"StartedAt timestamp must round-trip through the API")
}

// TestTraceGeneration_GetRunGraph_NotFound verifies that GetRunGraph returns
// NotFound for a run that does not exist.
func TestTraceGeneration_GetRunGraph_NotFound(t *testing.T) {
	c, _, cleanup := newTestServer(t)
	defer cleanup()

	_, err := c.GetRunGraph(context.Background(), connect.NewRequest(&apiv1.GetRunGraphRequest{
		Id: "ar-notfound",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
