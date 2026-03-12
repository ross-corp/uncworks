package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// TaskQueue is the default Temporal task queue for agent runs.
	TaskQueue = "aot-agent-runs"

	// Signal names
	SignalHumanInput = "human-input"
	SignalCancel     = "cancel"

	// Query names
	QueryGetState = "get-state"

	// Default polling interval for agent status checks.
	statusPollInterval = 5 * time.Second

	// Default activity timeout.
	activityTimeout = 5 * time.Minute

	// Heartbeat interval for long-running activities.
	heartbeatInterval = 10 * time.Second
)

// WorkflowInput contains the parameters for starting an AgentRunWorkflow.
type WorkflowInput struct {
	AgentRunName string
	Namespace    string
	RepoURL      string
	Branch       string
	Prompt       string
	DevboxConfig string
	TTLSeconds   int32
	Image        string
	EnvVars      map[string]string
}

// WorkflowState represents the current state of the workflow, returned by queries.
type WorkflowState struct {
	Phase   string
	Message string
	PodName string
}

// HumanInputSignal is the payload for the human-input signal.
type HumanInputSignal struct {
	Input string
}

// Activity function references used by the workflow.
// These are set by the worker at startup and used for activity dispatch.
// Using variable references allows the test suite to mock them.
var (
	CreateAgentPodActivity    func(Activities) func(interface{}, CreateAgentPodInput) (*CreateAgentPodOutput, error)
	WaitForHydrationActivity  func(Activities) func(interface{}, WaitForHydrationInput) error
	StartAgentActivity        func(Activities) func(interface{}, StartAgentInput) error
	GetAgentStatusActivity    func(Activities) func(interface{}, GetAgentStatusInput) (*GetAgentStatusOutput, error)
	ForwardHumanInputActivity func(Activities) func(interface{}, ForwardHumanInputInput) error
	StopAgentActivity         func(Activities) func(interface{}, StopAgentInput) error
	CleanupPodActivity        func(Activities) func(interface{}, CleanupPodInput) error
)

// AgentRunWorkflow orchestrates the full lifecycle of an agent run.
//
// Lifecycle: CreatePod → WaitForHydration → StartAgent → poll status → cleanup
// Signals: human-input (forwards HITL input), cancel (graceful termination)
// Queries: get-state (returns current phase, message, pod name)
func AgentRunWorkflow(ctx workflow.Context, input WorkflowInput) error {
	state := &WorkflowState{
		Phase:   "Pending",
		Message: "Workflow started",
	}

	// Register query handler for get-state
	if err := workflow.SetQueryHandler(ctx, QueryGetState, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return fmt.Errorf("set query handler: %w", err)
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: activityTimeout,
		HeartbeatTimeout:    heartbeatInterval * 3,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, activityOpts)

	// Activity references — using the struct method names for proper registration
	var a *Activities

	// Compensation: ensure pod cleanup on any failure
	var podName string
	defer func() {
		if podName != "" {
			cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
			cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
			})
			_ = workflow.ExecuteActivity(cleanupCtx, a.CleanupPod, CleanupPodInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(cleanupCtx, nil)
		}
	}()

	// --- Step 1: Create agent pod ---
	state.Phase = "Creating"
	state.Message = "Creating agent pod"

	podInput := CreateAgentPodInput{
		Name:         fmt.Sprintf("agentrun-%s", input.AgentRunName),
		Namespace:    input.Namespace,
		AgentRunName: input.AgentRunName,
		RepoURL:      input.RepoURL,
		Branch:       input.Branch,
		Prompt:       input.Prompt,
		DevboxConfig: input.DevboxConfig,
		Image:        input.Image,
		EnvVars:      input.EnvVars,
	}

	var createOutput CreateAgentPodOutput
	if err := workflow.ExecuteActivity(actCtx, a.CreateAgentPod, podInput).Get(ctx, &createOutput); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during pod creation"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Failed to create pod: %v", err)
		return err
	}
	podName = createOutput.PodName
	state.PodName = podName

	// --- Step 2: Wait for hydration ---
	state.Phase = "Hydrating"
	state.Message = "Waiting for workspace hydration"

	hydrationOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	hydrationCtx := workflow.WithActivityOptions(ctx, hydrationOpts)

	if err := workflow.ExecuteActivity(hydrationCtx, a.WaitForHydration, WaitForHydrationInput{
		PodName:   podName,
		Namespace: input.Namespace,
	}).Get(ctx, nil); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during hydration"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Hydration failed: %v", err)
		return err
	}

	// --- Step 3: Start agent ---
	state.Phase = "Running"
	state.Message = "Starting agent"

	if err := workflow.ExecuteActivity(actCtx, a.StartAgent, StartAgentInput{
		PodName:   podName,
		Namespace: input.Namespace,
		Prompt:    input.Prompt,
	}).Get(ctx, nil); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during agent start"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Failed to start agent: %v", err)
		return err
	}

	state.Message = "Agent running"

	// --- Step 4: Set up signal handlers and TTL timer, poll for completion ---
	cancelCh := workflow.GetSignalChannel(ctx, SignalCancel)
	humanInputCh := workflow.GetSignalChannel(ctx, SignalHumanInput)

	// TTL timer
	var ttlDuration time.Duration
	if input.TTLSeconds > 0 {
		ttlDuration = time.Duration(input.TTLSeconds) * time.Second
	} else {
		ttlDuration = time.Hour // Default 1 hour
	}
	ttlTimer := workflow.NewTimer(ctx, ttlDuration)

	// Status polling ticker
	pollTimer := workflow.NewTimer(ctx, statusPollInterval)

	for {
		selector := workflow.NewSelector(ctx)

		// Handle cancel signal
		selector.AddReceive(cancelCh, func(ch workflow.ReceiveChannel, more bool) {
			ch.Receive(ctx, nil)
			state.Phase = "Cancelling"
			state.Message = "Cancel signal received"

			_ = workflow.ExecuteActivity(actCtx, a.StopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(ctx, nil)

			state.Phase = "Cancelled"
			state.Message = "Cancelled by user"
		})

		// Handle human input signal
		selector.AddReceive(humanInputCh, func(ch workflow.ReceiveChannel, more bool) {
			var signal HumanInputSignal
			ch.Receive(ctx, &signal)

			_ = workflow.ExecuteActivity(actCtx, a.ForwardHumanInput, ForwardHumanInputInput{
				PodName:   podName,
				Namespace: input.Namespace,
				Input:     signal.Input,
			}).Get(ctx, nil)

			state.Phase = "Running"
			state.Message = "Human input forwarded"
		})

		// Handle TTL expiry
		selector.AddFuture(ttlTimer, func(f workflow.Future) {
			state.Phase = "Failed"
			state.Message = "Exceeded TTL"

			_ = workflow.ExecuteActivity(actCtx, a.StopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(ctx, nil)
		})

		// Poll agent status
		selector.AddFuture(pollTimer, func(f workflow.Future) {
			var statusOutput GetAgentStatusOutput
			err := workflow.ExecuteActivity(actCtx, a.GetAgentStatus, GetAgentStatusInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(ctx, &statusOutput)

			if err == nil {
				switch statusOutput.State {
				case "AGENT_PROCESS_STATE_COMPLETED":
					state.Phase = "Succeeded"
					state.Message = "Agent completed successfully"
				case "AGENT_PROCESS_STATE_FAILED":
					state.Phase = "Failed"
					state.Message = fmt.Sprintf("Agent failed: %s", statusOutput.Error)
				case "AGENT_PROCESS_STATE_WAITING_FOR_INPUT":
					state.Phase = "WaitingForInput"
					state.Message = "Agent waiting for human input"
				}
			}

			// Reset poll timer for next check
			pollTimer = workflow.NewTimer(ctx, statusPollInterval)
		})

		selector.Select(ctx)

		// Check terminal states
		switch state.Phase {
		case "Succeeded", "Failed", "Cancelled":
			// Pod cleanup happens via defer
			return nil
		}
	}
}

// SpawnJuniorInput contains parameters for spawning a child workflow.
type SpawnJuniorInput struct {
	ParentRunName string
	Namespace     string
	Task          string
	RepoURL       string
	Branch        string
	DevboxConfig  string
	TTLSeconds    int32
	Image         string
	EnvVars       map[string]string
	Blocking      bool
}

// SpawnJuniorWorkflow starts a child AgentRunWorkflow for a junior agent.
func SpawnJuniorWorkflow(ctx workflow.Context, input SpawnJuniorInput) error {
	juniorName := fmt.Sprintf("%s-junior-%d",
		input.ParentRunName,
		workflow.Now(ctx).UnixMilli()%100000)

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: juniorName,
		TaskQueue:  TaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	childInput := WorkflowInput{
		AgentRunName: juniorName,
		Namespace:    input.Namespace,
		RepoURL:      input.RepoURL,
		Branch:       input.Branch,
		Prompt:       input.Task,
		DevboxConfig: input.DevboxConfig,
		TTLSeconds:   input.TTLSeconds,
		Image:        input.Image,
		EnvVars:      input.EnvVars,
	}

	future := workflow.ExecuteChildWorkflow(childCtx, AgentRunWorkflow, childInput)

	if input.Blocking {
		return future.Get(ctx, nil)
	}

	// Fire-and-forget: just wait for the child to start
	return future.GetChildWorkflowExecution().Get(ctx, nil)
}
