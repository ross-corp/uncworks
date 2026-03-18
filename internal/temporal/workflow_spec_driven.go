package temporal

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var (
	// Max retries for spec-driven verification failures. Configurable via AOT_PIPELINE_MAX_RETRIES.
	defaultMaxRetries = envOrDefaultInt("AOT_PIPELINE_MAX_RETRIES", 3)

	// Planning stage timeout. Configurable via AOT_PIPELINE_PLAN_TIMEOUT (seconds).
	defaultPlanTimeout = envOrDefaultDuration("AOT_PIPELINE_PLAN_TIMEOUT", 2*time.Minute)
)

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// PipelineStage represents the current stage of a spec-driven run.
type PipelineStage string

const (
	PipelineStagePlanning  PipelineStage = "planning"
	PipelineStageExecuting PipelineStage = "executing"
	PipelineStageVerifying PipelineStage = "verifying"
)

// VerificationResult is the structured output of the verification stage.
type VerificationResult struct {
	Pass            bool             `json:"pass"`
	TasksCompleted  int              `json:"tasksCompleted"`
	TasksTotal      int              `json:"tasksTotal"`
	ValidationValid bool             `json:"validationValid"`
	AutomatedChecks []AutomatedCheck `json:"automatedChecks"`
	LLMVerdict      *LLMVerdict      `json:"llmVerdict,omitempty"`
	FailureReport   string           `json:"failureReport,omitempty"`
	ExecutionTimeMs int64            `json:"executionTimeMs"`
}

// AutomatedCheck is a single automated verification check result.
type AutomatedCheck struct {
	Name    string `json:"name"`
	Pass    bool   `json:"pass"`
	Output  string `json:"output,omitempty"`
	Command string `json:"command,omitempty"`
}

// LLMVerdict is the LLM judge's evaluation of semantic criteria.
type LLMVerdict struct {
	Pass     bool              `json:"pass"`
	Criteria []CriterionResult `json:"criteria"`
	Model    string            `json:"model"`
}

// CriterionResult is a single WHEN/THEN criterion evaluation.
type CriterionResult struct {
	Scenario    string `json:"scenario"`
	Pass        bool   `json:"pass"`
	Explanation string `json:"explanation"`
}

// PlanRunInput contains parameters for the planning stage activity.
type PlanRunInput struct {
	AgentRunName string
	Namespace    string
	PodName      string
	PodIP        string
	Prompt       string
	SpecContent  string
	RepoPath     string
}

// PlanRunOutput contains the result of the planning stage.
type PlanRunOutput struct {
	ChangeName string
	TaskCount  int
	SpecsValid bool
}

// VerifyRunInput contains parameters for the verification stage activity.
type VerifyRunInput struct {
	AgentRunName string
	Namespace    string
	PodName      string
	PodIP        string
	ChangeName   string
	RepoPath     string
}

// VerifyRunOutput contains the result of the verification stage.
type VerifyRunOutput struct {
	Result VerificationResult
}

const (
	ActivityPlanRun   = "PlanRun"
	ActivityVerifyRun = "VerifyRun"
)

// runSpecDrivenPipeline executes the Plan → Execute → Verify pipeline
// with retry on verification failure.
func runSpecDrivenPipeline(ctx workflow.Context, input WorkflowInput) error {
	state := &WorkflowState{
		Phase:   "Running",
		Message: "Spec-driven pipeline: starting",
	}

	if err := workflow.SetQueryHandler(ctx, QueryGetState, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return fmt.Errorf("set query handler: %w", err)
	}

	cancelCh := workflow.GetSignalChannel(ctx, SignalCancel)

	// Activity options for pipeline stages.
	planOpts := workflow.ActivityOptions{
		StartToCloseTimeout: defaultPlanTimeout,
		HeartbeatTimeout:    30 * time.Second,
	}
	executeOpts := workflow.ActivityOptions{
		StartToCloseTimeout: activityTimeout,
		HeartbeatTimeout:    heartbeatInterval * 3,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	verifyOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}

	// Compensation: ensure deployment scale-down and LLM key revocation on any exit.
	var podName string
	var deploymentName string
	var llmKey string
	defer func() {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		if llmKey != "" {
			_ = workflow.ExecuteActivity(cleanupCtx, ActivityRevokeLLMKey, RevokeLLMKeyInput{
				Key: llmKey,
			}).Get(cleanupCtx, nil)
		}
		if deploymentName != "" {
			_ = workflow.ExecuteActivity(cleanupCtx, ActivityScaleDownDeployment, ScaleDownDeploymentInput{
				DeploymentName: deploymentName,
				Namespace:      input.Namespace,
			}).Get(cleanupCtx, nil)
		}
	}()

	// --- Provision LLM key ---
	state.Phase = "Running"
	state.Message = "Provisioning LLM key"

	var keyOutput ProvisionLLMKeyOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, executeOpts),
		ActivityProvisionLLMKey, ProvisionLLMKeyInput{
			AgentRunName: input.AgentRunName,
			Namespace:    input.Namespace,
			ModelTier:    input.ModelTier,
			MaxBudget:    input.MaxBudget,
		},
	).Get(ctx, &keyOutput); err != nil {
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

	// --- Create deployment ---
	state.Message = "Creating agent deployment"

	var deployOutput CreateAgentDeploymentOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, executeOpts),
		ActivityCreateAgentDeployment, CreateAgentDeploymentInput{
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
		},
	).Get(ctx, &deployOutput); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during deployment creation"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Failed to create deployment: %v", err)
		return err
	}
	deploymentName = deployOutput.DeploymentName
	podName = deployOutput.DeploymentName
	state.PodName = podName
	state.DeploymentName = deploymentName

	// --- Wait for hydration ---
	state.Message = "Waiting for workspace hydration"

	var hydrationOutput WaitForHydrationOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
		}),
		ActivityWaitForHydration, WaitForHydrationInput{
			PodName:      podName,
			Namespace:    input.Namespace,
			AgentRunName: input.AgentRunName,
		},
	).Get(ctx, &hydrationOutput); err != nil {
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

	// --- Handle cancel signal throughout pipeline ---
	go func() {
		cancelCh.Receive(ctx, nil)
		state.Phase = "Cancelled"
		state.Message = "Cancelled by user"
	}()

	// =============================================
	// STAGE 1: PLAN — Generate OpenSpec change
	// =============================================
	state.Message = "Planning: generating spec from prompt"

	planInput := PlanRunInput{
		AgentRunName: input.AgentRunName,
		Namespace:    input.Namespace,
		PodName:      podName,
		PodIP:        podIP,
		Prompt:       input.Prompt,
		SpecContent:  input.SpecContent,
		RepoPath:     "/workspace",
	}

	var planOutput PlanRunOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, planOpts),
		ActivityPlanRun, planInput,
	).Get(ctx, &planOutput); err != nil {
		if temporal.IsCanceledError(err) {
			state.Phase = "Cancelled"
			state.Message = "Cancelled during planning"
			return err
		}
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Planning failed: %v", err)
		return err
	}

	changeName := planOutput.ChangeName
	maxRetries := defaultMaxRetries

	// =============================================
	// STAGE 2 + 3: EXECUTE → VERIFY (with retry)
	// =============================================
	var lastFailureReport string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if state.Phase == "Cancelled" {
			return fmt.Errorf("cancelled by user")
		}

		// --- EXECUTE ---
		state.Message = fmt.Sprintf("Executing: attempt %d/%d", attempt, maxRetries)

		prompt := input.Prompt
		if lastFailureReport != "" {
			prompt = fmt.Sprintf("PREVIOUS ATTEMPT FAILED VERIFICATION:\n\n%s\n\n---\n\nOriginal task:\n%s",
				lastFailureReport, input.Prompt)
		}

		if err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, executeOpts),
			ActivityStartAgent, StartAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
				PodIP:     podIP,
				Prompt:    prompt,
				RepoPath:  "/workspace",
			},
		).Get(ctx, nil); err != nil {
			if temporal.IsCanceledError(err) {
				state.Phase = "Cancelled"
				state.Message = "Cancelled during execution"
				return err
			}
			state.Phase = "Failed"
			state.Message = fmt.Sprintf("Execution failed: %v", err)
			return err
		}

		// Poll for agent completion (reuse existing polling logic).
		if err := pollAgentStatus(ctx, state, podName, input.Namespace, podIP, input.TTLSeconds, cancelCh); err != nil {
			return err
		}

		if state.Phase == "Cancelled" {
			return fmt.Errorf("cancelled by user")
		}

		// --- VERIFY ---
		state.Message = fmt.Sprintf("Verifying: evaluating against spec (attempt %d/%d)", attempt, maxRetries)

		verifyInput := VerifyRunInput{
			AgentRunName: input.AgentRunName,
			Namespace:    input.Namespace,
			PodName:      podName,
			PodIP:        podIP,
			ChangeName:   changeName,
			RepoPath:     "/workspace",
		}

		var verifyOutput VerifyRunOutput
		if err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, verifyOpts),
			ActivityVerifyRun, verifyInput,
		).Get(ctx, &verifyOutput); err != nil {
			if temporal.IsCanceledError(err) {
				state.Phase = "Cancelled"
				state.Message = "Cancelled during verification"
				return err
			}
			// Verification activity failure is not a run failure — treat as verify fail.
			workflow.GetLogger(ctx).Warn("Verification activity error", "error", err)
			lastFailureReport = fmt.Sprintf("Verification activity error: %v", err)
			continue
		}

		if verifyOutput.Result.Pass {
			state.Phase = "Succeeded"
			state.Message = fmt.Sprintf("Spec-driven pipeline: verified and archived (attempt %d)", attempt)
			return nil
		}

		// Verification failed — prepare retry context.
		lastFailureReport = verifyOutput.Result.FailureReport
		workflow.GetLogger(ctx).Info("Verification failed, will retry",
			"attempt", attempt,
			"maxRetries", maxRetries,
			"failureReport", lastFailureReport,
		)
	}

	// All retries exhausted.
	state.Phase = "Failed"
	state.Message = fmt.Sprintf("Spec-driven pipeline: failed verification after %d attempts. %s",
		maxRetries, lastFailureReport)
	return nil
}

// pollAgentStatus reuses the existing agent status polling logic from the
// single-agent workflow. It blocks until the agent completes, fails, or is cancelled.
func pollAgentStatus(ctx workflow.Context, state *WorkflowState, podName, namespace, podIP string, ttlSeconds int32, cancelCh workflow.ReceiveChannel) error {
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

	var ttlDuration time.Duration
	if ttlSeconds > 0 {
		ttlDuration = time.Duration(ttlSeconds) * time.Second
	} else {
		ttlDuration = time.Hour
	}
	ttlTimer := workflow.NewTimer(ctx, ttlDuration)
	pollTimer := workflow.NewTimer(ctx, statusPollInterval)
	consecutiveErrors := 0

	for {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(cancelCh, func(ch workflow.ReceiveChannel, more bool) {
			ch.Receive(ctx, nil)
			state.Phase = "Cancelled"
			state.Message = "Cancel signal received"
			_ = workflow.ExecuteActivity(actCtx, ActivityStopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: namespace,
				PodIP:     podIP,
			}).Get(ctx, nil)
		})

		selector.AddFuture(ttlTimer, func(f workflow.Future) {
			state.Phase = "Failed"
			state.Message = "Exceeded TTL"
			_ = workflow.ExecuteActivity(actCtx, ActivityStopAgent, StopAgentInput{
				PodName:   podName,
				Namespace: namespace,
				PodIP:     podIP,
			}).Get(ctx, nil)
		})

		selector.AddFuture(pollTimer, func(f workflow.Future) {
			var statusOutput GetAgentStatusOutput
			err := workflow.ExecuteActivity(actCtx, ActivityGetAgentStatus, GetAgentStatusInput{
				PodName:   podName,
				Namespace: namespace,
				PodIP:     podIP,
			}).Get(ctx, &statusOutput)

			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveStatusErrors {
					state.Phase = "Failed"
					state.Message = fmt.Sprintf("Sidecar unreachable after %d errors: %v", consecutiveErrors, err)
				}
			} else {
				consecutiveErrors = 0
				switch statusOutput.State {
				case "AGENT_PROCESS_STATE_COMPLETED":
					state.Phase = "Succeeded"
					state.Message = "Agent completed"
				case "AGENT_PROCESS_STATE_FAILED":
					state.Phase = "Failed"
					state.Message = fmt.Sprintf("Agent failed: %s", statusOutput.Error)
				case "AGENT_PROCESS_STATE_WAITING_FOR_INPUT":
					state.Phase = "WaitingForInput"
					state.Message = "Agent waiting for human input"
				}
			}

			pollTimer = workflow.NewTimer(ctx, statusPollInterval)
		})

		selector.Select(ctx)

		switch state.Phase {
		case "Succeeded", "Failed", "Cancelled":
			return nil
		}
	}
}
