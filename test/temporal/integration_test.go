package temporal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	aottemporal "github.com/uncworks/aot/internal/temporal"
)

// getTemporalDevClient connects to a running Temporal dev server.
// Skips the test if TEMPORAL_HOST is not set or connection fails.
func getTemporalDevClient(t *testing.T) temporalclient.Client {
	t.Helper()

	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}

	c, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  host,
		Namespace: "default",
	})
	if err != nil {
		t.Skipf("Skipping: cannot connect to Temporal at %s: %v", host, err)
	}

	// Verify the connection actually works
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = c.CheckHealth(ctx, &temporalclient.CheckHealthRequest{})
	if err != nil {
		c.Close()
		t.Skipf("Skipping: Temporal health check failed: %v", err)
	}

	return c
}

// TestIntegration_WorkflowExecution runs a real workflow against the Temporal dev server
// with a worker that uses mock activities so no real pods are created.
func TestIntegration_WorkflowExecution(t *testing.T) {
	c := getTemporalDevClient(t)
	defer c.Close()

	ctx := context.Background()
	taskQueue := "test-integration-" + t.Name()

	// Start a worker with mock activities
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterActivity(&mockActivities{})

	err := w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Execute workflow
	run, err := c.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, aottemporal.AgentRunWorkflow, aottemporal.WorkflowInput{
		AgentRunName: "integration-test-run",
		Namespace:    "default",
		RepoURL:      "https://github.com/example/repo.git",
		Branch:       "main",
		Prompt:       "integration test",
		TTLSeconds:   3600,
	})
	require.NoError(t, err)

	// Query state while running
	resp, err := c.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), aottemporal.QueryGetState)
	require.NoError(t, err)

	var state aottemporal.WorkflowState
	require.NoError(t, resp.Get(&state))
	require.NotEmpty(t, state.Phase)

	// Wait for completion (mock activities complete immediately)
	err = run.Get(ctx, nil)
	require.NoError(t, err)
}

// mockActivities provides stub implementations that return success immediately.
type mockActivities struct{}

func (m *mockActivities) ProvisionLLMKey(_ context.Context, _ aottemporal.ProvisionLLMKeyInput) (*aottemporal.ProvisionLLMKeyOutput, error) {
	return &aottemporal.ProvisionLLMKeyOutput{}, nil
}

func (m *mockActivities) CreateAgentPod(_ context.Context, _ aottemporal.CreateAgentPodInput) (*aottemporal.CreateAgentPodOutput, error) {
	return &aottemporal.CreateAgentPodOutput{PodName: "mock-pod"}, nil
}

func (m *mockActivities) WaitForHydration(_ context.Context, _ aottemporal.WaitForHydrationInput) error {
	return nil
}

func (m *mockActivities) StartAgent(_ context.Context, _ aottemporal.StartAgentInput) error {
	return nil
}

func (m *mockActivities) GetAgentStatus(_ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
	return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil
}

func (m *mockActivities) ForwardHumanInput(_ context.Context, _ aottemporal.ForwardHumanInputInput) error {
	return nil
}

func (m *mockActivities) StopAgent(_ context.Context, _ aottemporal.StopAgentInput) error {
	return nil
}

func (m *mockActivities) CleanupPod(_ context.Context, _ aottemporal.CleanupPodInput) error {
	return nil
}

func (m *mockActivities) RevokeLLMKey(_ context.Context, _ aottemporal.RevokeLLMKeyInput) error {
	return nil
}
