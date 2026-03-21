package temporal

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

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
	Model        string
}

// PlanRunOutput contains the result of the planning stage.
type PlanRunOutput struct {
	ChangeName       string
	TaskCount        int
	SpecsValid       bool
	ValidationErrors []string // errors from openspec validate/status (for retry context)
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

// resolveStageConfig returns the stage config with defaults applied.
func resolveStageConfig(cfg *PipelineConfigInput, stage string) StageConfigInput {
	defaults := map[string]StageConfigInput{
		"plan":    {Model: "default-cloud", TimeoutSeconds: 300, MaxRetries: 2, OnFailure: "fail"},
		"execute": {Model: "default-cloud", TimeoutSeconds: 900, MaxRetries: 3, OnFailure: "retry"},
		"verify":  {Model: "default-cloud", TimeoutSeconds: 180, MaxRetries: 1, OnFailure: "fail"},
	}
	def := defaults[stage]

	var sc StageConfigInput
	if cfg != nil {
		switch stage {
		case "plan":
			sc = cfg.Plan
		case "execute":
			sc = cfg.Execute
		case "verify":
			sc = cfg.Verify
		}
	}

	if sc.Model == "" {
		sc.Model = def.Model
	}
	if sc.TimeoutSeconds == 0 {
		sc.TimeoutSeconds = def.TimeoutSeconds
	}
	if sc.MaxRetries == 0 {
		sc.MaxRetries = def.MaxRetries
	}
	if sc.OnFailure == "" {
		sc.OnFailure = def.OnFailure
	}
	return sc
}

// runSpecDrivenPipeline executes the Plan → Execute → Verify pipeline
// with retry on verification failure.
func runSpecDrivenPipeline(ctx workflow.Context, input WorkflowInput) error {
	planCfg := resolveStageConfig(input.PipelineConfig, "plan")
	execCfg := resolveStageConfig(input.PipelineConfig, "execute")
	verifyCfg := resolveStageConfig(input.PipelineConfig, "verify")

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

	// Activity options driven by per-stage config.
	planOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(planCfg.TimeoutSeconds) * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    60 * time.Second,
			MaximumAttempts:    3,
		},
	}
	executeOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(execCfg.TimeoutSeconds) * time.Second,
		HeartbeatTimeout:    heartbeatInterval * 3,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	verifyOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(verifyCfg.TimeoutSeconds) * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    60 * time.Second,
			MaximumAttempts:    3,
		},
	}

	// Compensation: ensure deployment scale-down and LLM key revocation on any exit.
	var podName string
	var deploymentName string
	var llmKey string
	defer func() {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 5,
			},
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
	workflow.Go(ctx, func(gCtx workflow.Context) {
		cancelCh.Receive(gCtx, nil)
		state.Phase = "Cancelled"
		state.Message = "Cancelled by user"
	})

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
		Model:        planCfg.Model,
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
	maxRetries := int(execCfg.MaxRetries)

	// Fix 1: Check plan validation result (was ignoring SpecsValid)
	if !planOutput.SpecsValid {
		state.Phase = "Failed"
		errMsg := "Planning produced invalid OpenSpec change"
		if len(planOutput.ValidationErrors) > 0 {
			errMsg += ": " + strings.Join(planOutput.ValidationErrors, "; ")
		}
		state.Message = errMsg
		return fmt.Errorf("%s", errMsg)
	}

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

		prompt := fmt.Sprintf("Implement the OpenSpec change '%s'.\n\nRead specs at /workspace/openspec/changes/%s/ for requirements.\nRead tasks.md for your checklist. Mark each task [x] as you complete it.\n\nOriginal task: %s",
			changeName, changeName, input.Prompt)
		if lastFailureReport != "" {
			prompt = fmt.Sprintf("PREVIOUS ATTEMPT FAILED:\n%s\n\nFix the issues and complete the OpenSpec change '%s'.\nRead specs at /workspace/openspec/changes/%s/\nMark ALL tasks [x] in tasks.md when complete.\n\nOriginal task: %s",
				lastFailureReport, changeName, changeName, input.Prompt)
		}

		if err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, executeOpts),
			ActivityStartAgent, StartAgentInput{
				PodName:   podName,
				Namespace: input.Namespace,
				PodIP:     podIP,
				Prompt:    prompt,
				RepoPath:  "/workspace",
				Model:     execCfg.Model,
				Stage:     "execute",
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
		state.Message = fmt.Sprintf("manage: verifying against spec (attempt %d/%d)", attempt, maxRetries)

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
			// --- POST-VERIFY: Enrich tags from diff ---
			enrichCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
			})
			repoPath := "/workspace"
			if len(input.Repos) > 0 {
				rp := input.Repos[0].Path
				if rp == "" {
					rp = repoNameFromURL(input.Repos[0].URL)
				}
				repoPath = "/workspace/" + rp
			}
			if enrichErr := workflow.ExecuteActivity(enrichCtx, ActivityEnrichRunTags, EnrichRunTagsInput{
				AgentRunName: input.AgentRunName,
				Namespace:    input.Namespace,
				PodIP:        podIP,
				RepoPath:     repoPath,
			}).Get(ctx, nil); enrichErr != nil {
				workflow.GetLogger(ctx).Warn("Tag enrichment failed", "error", enrichErr)
			}

			// --- POST-VERIFY: Push and PR ---
			if err := postVerifyPushAndPR(ctx, input, state, podIP, changeName, attempt); err != nil {
				workflow.GetLogger(ctx).Warn("Post-verify push/PR failed", "error", err)
				// Push/PR failure is not a pipeline failure
			}

			state.Phase = "Succeeded"
			state.Message = fmt.Sprintf("Spec-driven pipeline: verified and archived (attempt %d)", attempt)
			if state.PRUrl != "" {
				state.Message += fmt.Sprintf(", PR: %s", state.PRUrl)
			}
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

// postVerifyPushAndPR handles the post-verification push and PR creation steps.
// It is a no-op if neither AutoPush nor AutoPR is enabled.
func postVerifyPushAndPR(ctx workflow.Context, input WorkflowInput, state *WorkflowState, podIP, changeName string, attempt int) error {
	if !input.AutoPush && !input.AutoPR {
		return nil
	}

	gitOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	gitCtx := workflow.WithActivityOptions(ctx, gitOpts)

	branchName := fmt.Sprintf("aot/%s", input.AgentRunName)
	commitMsg := fmt.Sprintf("feat(%s): implement spec-driven change\n\nAgentRun: %s\nChange: %s\nAttempt: %d",
		changeName, input.AgentRunName, changeName, attempt)

	// Determine repo path — use first repo if available
	repoPath := "/workspace"
	if len(input.Repos) > 0 {
		rp := input.Repos[0].Path
		if rp == "" {
			rp = repoNameFromURL(input.Repos[0].URL)
		}
		repoPath = "/workspace/" + rp
	}

	state.Message = "Pushing changes to feature branch"

	var pushOutput PushChangesOutput
	if err := workflow.ExecuteActivity(gitCtx, ActivityPushChanges, PushChangesInput{
		AgentRunName:  input.AgentRunName,
		PodIP:         podIP,
		RepoPath:      repoPath,
		BranchName:    branchName,
		CommitMessage: commitMsg,
	}).Get(ctx, &pushOutput); err != nil {
		return fmt.Errorf("push changes: %w", err)
	}

	if !input.AutoPR {
		return nil
	}

	// Parse owner/repo from the first repo URL
	if len(input.Repos) == 0 {
		return fmt.Errorf("no repos configured, cannot create PR")
	}
	owner, repo, err := parseGitHubOwnerRepo(input.Repos[0].URL)
	if err != nil {
		return fmt.Errorf("parse repo URL for PR: %w", err)
	}

	baseBranch := input.PRBaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	state.Message = "Creating GitHub PR"

	prOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	prCtx := workflow.WithActivityOptions(ctx, prOpts)

	var prOutput CreatePROutput
	if err := workflow.ExecuteActivity(prCtx, ActivityCreatePR, CreatePRInput{
		RepoOwner:    owner,
		RepoName:     repo,
		BranchName:   branchName,
		BaseBranch:   baseBranch,
		Title:        fmt.Sprintf("feat(%s): %s", changeName, truncateForTitle(input.Prompt, 50)),
		Body:         fmt.Sprintf("## Automated PR from AOT Pipeline\n\n**AgentRun:** `%s`\n**Change:** `%s`\n**Commit:** `%s`\n\nThis PR was automatically created by the spec-driven pipeline after verification passed.", input.AgentRunName, changeName, pushOutput.CommitSHA),
		AgentRunName: input.AgentRunName,
	}).Get(ctx, &prOutput); err != nil {
		return fmt.Errorf("create PR: %w", err)
	}

	state.PRUrl = prOutput.PRUrl
	return nil
}

// parseGitHubOwnerRepo extracts owner and repo from a GitHub URL.
// Supports both HTTPS and SSH formats:
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
func parseGitHubOwnerRepo(repoURL string) (owner, repo string, err error) {
	// Handle SSH format
	if strings.HasPrefix(repoURL, "git@") {
		// git@github.com:owner/repo.git
		parts := strings.SplitN(repoURL, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL: %s", repoURL)
		}
		pathStr := strings.TrimSuffix(parts[1], ".git")
		segments := strings.SplitN(pathStr, "/", 2)
		if len(segments) != 2 {
			return "", "", fmt.Errorf("cannot parse owner/repo from SSH URL: %s", repoURL)
		}
		return segments[0], segments[1], nil
	}

	// Handle HTTPS format
	u, parseErr := url.Parse(repoURL)
	if parseErr != nil {
		return "", "", fmt.Errorf("parse URL: %w", parseErr)
	}
	pathStr := strings.TrimSuffix(u.Path, ".git")
	pathStr = strings.TrimPrefix(pathStr, "/")
	segments := strings.SplitN(pathStr, "/", 3)
	if len(segments) < 2 {
		return "", "", fmt.Errorf("cannot parse owner/repo from URL path: %s", u.Path)
	}
	return segments[0], segments[1], nil
}

// truncateForTitle truncates a string for use in a PR title, breaking at word boundaries.
func truncateForTitle(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// Break at last space before max
	truncated := s[:max]
	if idx := strings.LastIndex(truncated, " "); idx > 0 {
		truncated = truncated[:idx]
	}
	return truncated + "..."
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
					state.Message = "implement: completed, starting verification"
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
