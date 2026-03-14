package temporal

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// repoNameFromURL derives a directory name from a git URL.
func repoNameFromURL(repoURL string) string {
	if u, err := url.Parse(repoURL); err == nil && u.Path != "" {
		base := filepath.Base(u.Path)
		return strings.TrimSuffix(base, ".git")
	}
	base := filepath.Base(repoURL)
	return strings.TrimSuffix(base, ".git")
}

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

	// Maximum consecutive GetAgentStatus failures before failing the workflow.
	maxConsecutiveStatusErrors = 5
)

// Repository describes a single git repository for an agent run.
type Repository struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// WorkflowInput contains the parameters for starting an AgentRunWorkflow.
type WorkflowInput struct {
	AgentRunName     string
	Namespace        string
	Repos            []Repository
	Prompt           string
	DevboxConfig     string
	TTLSeconds       int32
	Image            string
	EnvVars          map[string]string
	ModelTier        string
	MaxBudget        float64
	LiteLLMBaseURL   string
	SpecContent      string
	WorkspaceName    string
	RetainPodMinutes int32
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

// Activity name constants — must match the method names registered on the Activities struct.
// The Temporal SDK resolves these by name against the struct registered on the worker.
const (
	ActivityProvisionLLMKey   = "ProvisionLLMKey"
	ActivityRevokeLLMKey      = "RevokeLLMKey"
	ActivityCreateAgentPod    = "CreateAgentPod"
	ActivityWaitForHydration  = "WaitForHydration"
	ActivityStartAgent        = "StartAgent"
	ActivityGetAgentStatus    = "GetAgentStatus"
	ActivityForwardHumanInput = "ForwardHumanInput"
	ActivityStopAgent         = "StopAgent"
	ActivityCleanupPod        = "CleanupPod"
	ActivityCollectLogs       = "CollectLogs"
)

// AgentRunWorkflow orchestrates the full lifecycle of an agent run.
//
// Lifecycle: CreatePod → WaitForHydration → StartAgent → poll status → cleanup
// Signals: human-input (forwards HITL input), cancel (graceful termination)
// Queries: get-state (returns current phase, message, pod name)
func AgentRunWorkflow(ctx workflow.Context, input WorkflowInput) error {
	// Auto-generate prompt for spec-driven runs
	if input.SpecContent != "" && input.Prompt == "" {
		input.Prompt = "Run `codespeak build` in the workspace directory. The spec file has been placed at spec/main.cs.md with a codespeak.json config. Execute the build and verify the output compiles/passes tests."
	}

	state := &WorkflowState{
		Phase:   "Pending",
		Message: "Workflow started",
	}

	// Register query handler and signal channels immediately at workflow start
	if err := workflow.SetQueryHandler(ctx, QueryGetState, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return fmt.Errorf("set query handler: %w", err)
	}

	cancelCh := workflow.GetSignalChannel(ctx, SignalCancel)
	humanInputCh := workflow.GetSignalChannel(ctx, SignalHumanInput)

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

	// Compensation: ensure pod cleanup and LLM key revocation on any exit
	var podName string
	var llmKey string
	defer func() {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		if llmKey != "" {
			if err := workflow.ExecuteActivity(cleanupCtx, ActivityRevokeLLMKey, RevokeLLMKeyInput{
				Key: llmKey,
			}).Get(cleanupCtx, nil); err != nil {
				workflow.GetLogger(ctx).Error("Failed to revoke LLM key during cleanup", "key", llmKey, "error", err)
			}
		}
		if podName != "" {
			// Collect logs before pod deletion
			logCleanupCtx := workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 60 * time.Second,
			})
			var logOutput CollectLogsOutput
			if err := workflow.ExecuteActivity(logCleanupCtx, ActivityCollectLogs, CollectLogsInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(logCleanupCtx, &logOutput); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to collect logs", "error", err)
			}

			// Pod retention: wait before deleting
			retainMinutes := input.RetainPodMinutes
			if retainMinutes < 0 {
				retainMinutes = 0
			}
			if retainMinutes == 0 && input.RetainPodMinutes == 0 {
				retainMinutes = 30 // Default 30 minutes
			}
			if retainMinutes > 0 {
				workflow.GetLogger(ctx).Info("Retaining pod", "minutes", retainMinutes)
				_ = workflow.Sleep(cleanupCtx, time.Duration(retainMinutes)*time.Minute)
			}

			if err := workflow.ExecuteActivity(cleanupCtx, ActivityCleanupPod, CleanupPodInput{
				PodName:   podName,
				Namespace: input.Namespace,
			}).Get(cleanupCtx, nil); err != nil {
				workflow.GetLogger(ctx).Error("Failed to cleanup pod", "podName", podName, "error", err)
			}
		}
	}()

	// --- Step 1: Provision LLM key ---
	state.Phase = "Creating"
	state.Message = "Provisioning LLM key"

	var keyOutput ProvisionLLMKeyOutput
	if err := workflow.ExecuteActivity(actCtx, ActivityProvisionLLMKey, ProvisionLLMKeyInput{
		AgentRunName: input.AgentRunName,
		Namespace:    input.Namespace,
		ModelTier:    input.ModelTier,
		MaxBudget:    input.MaxBudget,
	}).Get(ctx, &keyOutput); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during LLM key provisioning"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Failed to provision LLM key: %v", err)
		return err
	}
	llmKey = keyOutput.Key

	// --- Step 2: Create agent pod ---
	state.Message = "Creating agent pod"

	podInput := CreateAgentPodInput{
		Name:           fmt.Sprintf("agentrun-%s", input.AgentRunName),
		Namespace:      input.Namespace,
		AgentRunName:   input.AgentRunName,
		Repos:          input.Repos,
		Prompt:         input.Prompt,
		DevboxConfig:   input.DevboxConfig,
		Image:          input.Image,
		EnvVars:        input.EnvVars,
		LLMKey:         llmKey,
		LiteLLMBaseURL: input.LiteLLMBaseURL,
		ModelID:        modelIDFromTier(input.ModelTier),
		SpecContent:    input.SpecContent,
	}

	var createOutput CreateAgentPodOutput
	if err := workflow.ExecuteActivity(actCtx, ActivityCreateAgentPod, podInput).Get(ctx, &createOutput); err != nil {
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

	// --- Step 3: Wait for hydration ---
	state.Phase = "Hydrating"
	state.Message = "Waiting for workspace hydration"

	hydrationOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	hydrationCtx := workflow.WithActivityOptions(ctx, hydrationOpts)

	var hydrationOutput WaitForHydrationOutput
	if err := workflow.ExecuteActivity(hydrationCtx, ActivityWaitForHydration, WaitForHydrationInput{
		PodName:   podName,
		Namespace: input.Namespace,
	}).Get(ctx, &hydrationOutput); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during hydration"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Hydration failed: %v", err)
		return err
	}
	podIP := hydrationOutput.PodIP

	// Use the workspace root as the working directory for multi-repo support.
	workspacePath := "/workspace"

	// --- Step 4: Start agent ---
	state.Phase = "Running"
	state.Message = "Starting agent"

	if err := workflow.ExecuteActivity(actCtx, ActivityStartAgent, StartAgentInput{
		PodName:   podName,
		Namespace: input.Namespace,
		PodIP:     podIP,
		Prompt:    input.Prompt,
		RepoPath:  workspacePath,
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

	// --- Step 5: Set up TTL timer and poll for completion ---
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
	consecutiveErrors := 0

	for {
		selector := workflow.NewSelector(ctx)

		// Handle cancel signal
		selector.AddReceive(cancelCh, func(ch workflow.ReceiveChannel, more bool) {
			ch.Receive(ctx, nil)
			state.Phase = "Cancelling"
			state.Message = "Cancel signal received"

			if err := workflow.ExecuteActivity(actCtx, ActivityStopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
				PodIP:     podIP,
			}).Get(ctx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to stop agent during cancel", "error", err)
			}

			state.Phase = "Cancelled"
			state.Message = "Cancelled by user"
		})

		// Handle human input signal
		selector.AddReceive(humanInputCh, func(ch workflow.ReceiveChannel, more bool) {
			var signal HumanInputSignal
			ch.Receive(ctx, &signal)

			if err := workflow.ExecuteActivity(actCtx, ActivityForwardHumanInput, ForwardHumanInputInput{
				AgentRunID: input.AgentRunName,
				PodName:    podName,
				Namespace:  input.Namespace,
				PodIP:      podIP,
				Input:      signal.Input,
			}).Get(ctx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to forward human input", "error", err)
			}

			state.Phase = "Running"
			state.Message = "Human input forwarded"
		})

		// Handle TTL expiry
		selector.AddFuture(ttlTimer, func(f workflow.Future) {
			state.Phase = "Failed"
			state.Message = "Exceeded TTL"

			if err := workflow.ExecuteActivity(actCtx, ActivityStopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
				PodIP:     podIP,
			}).Get(ctx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to stop agent after TTL", "error", err)
			}
		})

		// Poll agent status
		selector.AddFuture(pollTimer, func(f workflow.Future) {
			var statusOutput GetAgentStatusOutput
			err := workflow.ExecuteActivity(actCtx, ActivityGetAgentStatus, GetAgentStatusInput{
				PodName:   podName,
				Namespace: input.Namespace,
				PodIP:     podIP,
			}).Get(ctx, &statusOutput)

			if err != nil {
				consecutiveErrors++
				workflow.GetLogger(ctx).Warn("GetAgentStatus failed",
					"error", err,
					"consecutiveErrors", consecutiveErrors,
					"maxConsecutiveErrors", maxConsecutiveStatusErrors,
				)
				if consecutiveErrors >= maxConsecutiveStatusErrors {
					state.Phase = "Failed"
					state.Message = fmt.Sprintf("Sidecar unreachable after %d consecutive errors: %v", consecutiveErrors, err)
				}
			} else {
				consecutiveErrors = 0
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
	ParentRunName  string
	Namespace      string
	Task           string
	Repos          []Repository
	DevboxConfig   string
	TTLSeconds     int32
	Image          string
	EnvVars        map[string]string
	Blocking       bool
	ModelTier      string
	MaxBudget      float64
	LiteLLMBaseURL string
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
		AgentRunName:   juniorName,
		Namespace:      input.Namespace,
		Repos:          input.Repos,
		Prompt:         input.Task,
		DevboxConfig:   input.DevboxConfig,
		TTLSeconds:     input.TTLSeconds,
		Image:          input.Image,
		EnvVars:        input.EnvVars,
		ModelTier:      input.ModelTier,
		MaxBudget:      input.MaxBudget,
		LiteLLMBaseURL: input.LiteLLMBaseURL,
	}

	future := workflow.ExecuteChildWorkflow(childCtx, AgentRunWorkflow, childInput)

	if input.Blocking {
		return future.Get(ctx, nil)
	}

	// Fire-and-forget: just wait for the child to start
	return future.GetChildWorkflowExecution().Get(ctx, nil)
}

// modelIDFromTier maps a model tier name to a pi-coding-agent model identifier.
// LiteLLM exposes models as OpenAI-compatible, so we use the openai/ prefix.
func modelIDFromTier(tier string) string {
	if tier == "" {
		return "litellm/default"
	}
	return "litellm/" + tier
}
