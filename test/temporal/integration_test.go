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
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
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

// TestIntegration_HITLSignalFlow verifies the HITL flow against a real Temporal server:
// agent reports waiting → signal human input → agent resumes → completes.
func TestIntegration_HITLSignalFlow(t *testing.T) {
	c := getTemporalDevClient(t)
	defer c.Close()

	ctx := context.Background()
	taskQueue := "test-integration-" + t.Name()

	mock := &hitlMockActivities{}

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterActivity(mock)

	require.NoError(t, w.Start())
	defer w.Stop()

	run, err := c.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, aottemporal.AgentRunWorkflow, aottemporal.WorkflowInput{
		AgentRunName: "hitl-integration-test",
		Namespace:    "default",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "integration HITL test",
		TTLSeconds:   3600,
	})
	require.NoError(t, err)

	// Wait for agent to report waiting state
	time.Sleep(8 * time.Second)

	// Query state — should be waiting for input
	resp, err := c.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), aottemporal.QueryGetState)
	require.NoError(t, err)
	var state aottemporal.WorkflowState
	require.NoError(t, resp.Get(&state))
	t.Logf("State before signal: phase=%s", state.Phase)

	// Send human input signal
	err = c.SignalWorkflow(ctx, run.GetID(), run.GetRunID(), aottemporal.SignalHumanInput, aottemporal.HumanInputSignal{
		Input: "yes, approved",
	})
	require.NoError(t, err)
	t.Log("Sent human input signal")

	// Wait for workflow to complete
	err = run.Get(ctx, nil)
	require.NoError(t, err)
	t.Log("Workflow completed after HITL signal")
}

// TestIntegration_TTLExpiry verifies TTL enforcement against a real Temporal server.
func TestIntegration_TTLExpiry(t *testing.T) {
	c := getTemporalDevClient(t)
	defer c.Close()

	ctx := context.Background()
	taskQueue := "test-integration-" + t.Name()

	mock := &ttlMockActivities{}

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(aottemporal.AgentRunWorkflow)
	w.RegisterActivity(mock)

	require.NoError(t, w.Start())
	defer w.Stop()

	run, err := c.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, aottemporal.AgentRunWorkflow, aottemporal.WorkflowInput{
		AgentRunName: "ttl-integration-test",
		Namespace:    "default",
		Repos:        []aottemporal.Repository{{URL: "https://github.com/example/repo.git", Branch: "main"}},
		Prompt:       "integration TTL test",
		TTLSeconds:   5, // Short TTL
	})
	require.NoError(t, err)

	// Wait for TTL to fire and workflow to complete
	err = run.Get(ctx, nil)
	require.NoError(t, err)

	require.True(t, mock.stopCalled, "StopAgent should have been called on TTL expiry")
	require.True(t, mock.cleanupCalled, "CleanupPod should have been called after TTL")
	t.Log("Workflow completed via TTL expiry")
}

// mockActivities provides stub implementations that return success immediately.
type mockActivities struct{}

func (m *mockActivities) ProvisionLLMKey(_ context.Context, _ aottemporal.ProvisionLLMKeyInput) (*aottemporal.ProvisionLLMKeyOutput, error) {
	return &aottemporal.ProvisionLLMKeyOutput{}, nil
}

func (m *mockActivities) CreateAgentPod(_ context.Context, _ aottemporal.CreateAgentPodInput) (*aottemporal.CreateAgentPodOutput, error) {
	return &aottemporal.CreateAgentPodOutput{PodName: "mock-pod"}, nil
}

func (m *mockActivities) WaitForHydration(_ context.Context, _ aottemporal.WaitForHydrationInput) (*aottemporal.WaitForHydrationOutput, error) {
	return &aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.10", WorkspacePath: "/workspace/src/repo"}, nil
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

// hitlMockActivities simulates an agent that waits for input then completes.
type hitlMockActivities struct {
	statusCalls    int
	inputForwarded bool
}

func (m *hitlMockActivities) ProvisionLLMKey(_ context.Context, _ aottemporal.ProvisionLLMKeyInput) (*aottemporal.ProvisionLLMKeyOutput, error) {
	return &aottemporal.ProvisionLLMKeyOutput{}, nil
}
func (m *hitlMockActivities) CreateAgentPod(_ context.Context, _ aottemporal.CreateAgentPodInput) (*aottemporal.CreateAgentPodOutput, error) {
	return &aottemporal.CreateAgentPodOutput{PodName: "mock-hitl-pod"}, nil
}
func (m *hitlMockActivities) WaitForHydration(_ context.Context, _ aottemporal.WaitForHydrationInput) (*aottemporal.WaitForHydrationOutput, error) {
	return &aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.11", WorkspacePath: "/workspace/src/repo"}, nil
}
func (m *hitlMockActivities) StartAgent(_ context.Context, _ aottemporal.StartAgentInput) error {
	return nil
}
func (m *hitlMockActivities) GetAgentStatus(_ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
	m.statusCalls++
	if m.inputForwarded {
		return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_COMPLETED"}, nil
	}
	return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_WAITING_FOR_INPUT"}, nil
}
func (m *hitlMockActivities) ForwardHumanInput(_ context.Context, _ aottemporal.ForwardHumanInputInput) error {
	m.inputForwarded = true
	return nil
}
func (m *hitlMockActivities) StopAgent(_ context.Context, _ aottemporal.StopAgentInput) error {
	return nil
}
func (m *hitlMockActivities) CleanupPod(_ context.Context, _ aottemporal.CleanupPodInput) error {
	return nil
}
func (m *hitlMockActivities) RevokeLLMKey(_ context.Context, _ aottemporal.RevokeLLMKeyInput) error {
	return nil
}

// ttlMockActivities simulates an agent that stays running until TTL expires.
type ttlMockActivities struct {
	stopCalled    bool
	cleanupCalled bool
}

func (m *ttlMockActivities) ProvisionLLMKey(_ context.Context, _ aottemporal.ProvisionLLMKeyInput) (*aottemporal.ProvisionLLMKeyOutput, error) {
	return &aottemporal.ProvisionLLMKeyOutput{}, nil
}
func (m *ttlMockActivities) CreateAgentPod(_ context.Context, _ aottemporal.CreateAgentPodInput) (*aottemporal.CreateAgentPodOutput, error) {
	return &aottemporal.CreateAgentPodOutput{PodName: "mock-ttl-pod"}, nil
}
func (m *ttlMockActivities) WaitForHydration(_ context.Context, _ aottemporal.WaitForHydrationInput) (*aottemporal.WaitForHydrationOutput, error) {
	return &aottemporal.WaitForHydrationOutput{PodIP: "10.244.0.12", WorkspacePath: "/workspace/src/repo"}, nil
}
func (m *ttlMockActivities) StartAgent(_ context.Context, _ aottemporal.StartAgentInput) error {
	return nil
}
func (m *ttlMockActivities) GetAgentStatus(_ context.Context, _ aottemporal.GetAgentStatusInput) (*aottemporal.GetAgentStatusOutput, error) {
	// Always report running — TTL should stop it
	return &aottemporal.GetAgentStatusOutput{State: "AGENT_PROCESS_STATE_RUNNING"}, nil
}
func (m *ttlMockActivities) ForwardHumanInput(_ context.Context, _ aottemporal.ForwardHumanInputInput) error {
	return nil
}
func (m *ttlMockActivities) StopAgent(_ context.Context, _ aottemporal.StopAgentInput) error {
	m.stopCalled = true
	return nil
}
func (m *ttlMockActivities) CleanupPod(_ context.Context, _ aottemporal.CleanupPodInput) error {
	m.cleanupCalled = true
	return nil
}
func (m *ttlMockActivities) RevokeLLMKey(_ context.Context, _ aottemporal.RevokeLLMKeyInput) error {
	return nil
}
