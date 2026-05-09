package temporal

import (
	"encoding/json"
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

	// SignalHumanInput is the Temporal signal name for forwarding HITL user input to the workflow.
	SignalHumanInput = "human-input"
	// SignalCancel is the Temporal signal name for requesting graceful workflow cancellation.
	SignalCancel = "cancel"

	// QueryGetState is the Temporal query name that returns the current WorkflowState.
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

// OrchestrationMode specifies how an agent run handles decomposition.
type OrchestrationMode string

const (
	// OrchestrationModeSingle runs a single agent without decomposition.
	OrchestrationModeSingle OrchestrationMode = "single"
	// OrchestrationModeAuto lets a senior agent autonomously decompose the task into junior runs.
	OrchestrationModeAuto OrchestrationMode = "auto"
	// OrchestrationModeManual uses an explicit list of orchestration tasks provided in the spec.
	OrchestrationModeManual OrchestrationMode = "manual"
	// OrchestrationModeSpecDriven runs the three-stage plan/execute/verify pipeline against an OpenSpec change.
	OrchestrationModeSpecDriven OrchestrationMode = "spec-driven"
)

// OrchestrationTask defines a single sub-task in a manual orchestration.
type OrchestrationTask struct {
	Name     string   `json:"name"`
	Prompt   string   `json:"prompt"`
	RepoURLs []string `json:"repoUrls,omitempty"`
}

// DecompositionPlan is the structured output from a senior agent's decomposition.
type DecompositionPlan struct {
	Tasks             []DecompositionTask `json:"tasks"`
	IntegrationPrompt string              `json:"integration_prompt"`
}

// DecompositionTask is a single task in a decomposition plan.
type DecompositionTask struct {
	Name   string   `json:"name"`
	Prompt string   `json:"prompt"`
	Repos  []string `json:"repos,omitempty"`
}

// WorkflowInput contains the parameters for starting an AgentRunWorkflow.
type WorkflowInput struct {
	AgentRunName          string
	Namespace             string
	Repos                 []Repository
	Prompt                string
	DevboxConfig          string
	TTLSeconds            int32
	Image                 string
	EnvVars               map[string]string
	ModelTier             string
	ManageModelTier       string
	ImplementModelTier    string
	MaxBudget             float64
	LiteLLMBaseURL        string
	SpecContent           string
	WorkspaceName         string
	OrchestrationMode     OrchestrationMode
	Orchestration         []OrchestrationTask
	ParentRunID           string
	SpecRunID             string
	PipelineConfig        *PipelineConfigInput
	AutoPush              bool
	AutoPR                bool
	GitHubTokenSecretName string
	PRBaseBranch          string
	Project               string
	Feature               string
	Tags                  []string
	Backend               string
	SpecSource            string
}

// PipelineConfigInput contains per-stage configuration for spec-driven runs.
type PipelineConfigInput struct {
	Plan    StageConfigInput `json:"plan,omitempty"`
	Execute StageConfigInput `json:"execute,omitempty"`
	Verify  StageConfigInput `json:"verify,omitempty"`
}

// StageConfigInput configures a single pipeline stage.
type StageConfigInput struct {
	Model          string `json:"model,omitempty"`
	TimeoutSeconds int32  `json:"timeoutSeconds,omitempty"`
	MaxRetries     int32  `json:"maxRetries,omitempty"`
	OnFailure      string `json:"onFailure,omitempty"` // "retry" | "fail" | "skip"
}

// WorkflowState represents the current state of the workflow, returned by queries.
type WorkflowState struct {
	Phase          string
	Message        string
	PodName        string
	DeploymentName string
	PRUrl          string
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
	ActivityWaitForHydration  = "WaitForHydration"
	ActivityStartAgent        = "StartAgent"
	ActivityGetAgentStatus    = "GetAgentStatus"
	ActivityForwardHumanInput = "ForwardHumanInput"
	ActivityStopAgent         = "StopAgent"

	// Deployment-based activities
	ActivityCreateAgentDeployment = "CreateAgentDeployment"
	ActivityScaleDownDeployment   = "ScaleDownDeployment"
	ActivityArchiveAndCleanup     = "ArchiveAndCleanup"
	ActivityCollectAgentLogs      = "CollectAgentLogs"

	// Knowledge system activities
	ActivityPersistRunData = "PersistRunData"
	ActivityEmbedRunData   = "EmbedRunData"
	ActivityHydrateContext = "HydrateContext"

	// Git/PR activities
	ActivityPushChanges = "PushChanges"
	ActivityCreatePR    = "CreatePR"

	// Tag enrichment
	ActivityEnrichRunTags = "EnrichRunTags"

	// Trace span writing
	ActivityWriteTraceSpan = "WriteTraceSpan"
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

	// --- Step 0: Orchestration preamble ---
	// Auto-upgrade to spec-driven if specContent is provided.
	if input.SpecContent != "" && input.OrchestrationMode != OrchestrationModeSingle {
		if input.OrchestrationMode == "" || input.OrchestrationMode == OrchestrationModeSingle {
			input.OrchestrationMode = OrchestrationModeSpecDriven
		}
	}

	switch input.OrchestrationMode {
	case OrchestrationModeManual:
		return runManualOrchestration(ctx, input)
	case OrchestrationModeAuto:
		return runAutoOrchestration(ctx, input)
	case OrchestrationModeSpecDriven:
		return runSpecDrivenPipeline(ctx, input)
	}
	// OrchestrationModeSingle or unspecified: fall through to standard single-agent workflow.

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

	// Compensation: ensure deployment scale-down and LLM key revocation on any exit
	var podName string
	var deploymentName string
	var llmKey string
	var podIP string
	defer func() {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 5,
			},
		})

		// Persist run data to PostgreSQL (knowledge system) before scale-down
		if deploymentName != "" {
			repoURL := ""
			if len(input.Repos) > 0 {
				repoURL = input.Repos[0].URL
			}
			persistCtx := workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 2 * time.Minute,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    5 * time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    30 * time.Second,
					MaximumAttempts:    3,
				},
			})
			if err := workflow.ExecuteActivity(persistCtx, ActivityPersistRunData, PersistRunDataInput{
				AgentRunID:    input.AgentRunName,
				WorkspacePath: "/workspace",
				RepoURL:       repoURL,
				PodIP:         podIP,
			}).Get(persistCtx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to persist run data", "error", err)
			}

			// Embed run data asynchronously (does not block workflow completion)
			embedCtx := workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 5 * time.Minute,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    5 * time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    60 * time.Second,
					MaximumAttempts:    2,
				},
			})
			if err := workflow.ExecuteActivity(embedCtx, ActivityEmbedRunData, EmbedRunDataInput{
				AgentRunID: input.AgentRunName,
				RepoURL:    repoURL,
			}).Get(embedCtx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to embed run data", "error", err)
			}
		}

		if llmKey != "" {
			if err := workflow.ExecuteActivity(cleanupCtx, ActivityRevokeLLMKey, RevokeLLMKeyInput{
				Key: llmKey,
			}).Get(cleanupCtx, nil); err != nil {
				workflow.GetLogger(ctx).Error("Failed to revoke LLM key during cleanup", "key", llmKey, "error", err)
			}
		}
		if deploymentName != "" {
			// Collect agent logs before scale-down so chain contextFrom steps get real output.
			logCtx := workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 60 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 2,
				},
			})
			if err := workflow.ExecuteActivity(logCtx, ActivityCollectAgentLogs, CollectAgentLogsInput{
				AgentRunName: input.AgentRunName,
				Namespace:    input.Namespace,
				PodIP:        podIP,
			}).Get(logCtx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Failed to collect agent logs", "error", err)
			}

			// Scale deployment to 0 — PVC persists for later access/debug
			if err := workflow.ExecuteActivity(cleanupCtx, ActivityScaleDownDeployment, ScaleDownDeploymentInput{
				DeploymentName: deploymentName,
				Namespace:      input.Namespace,
			}).Get(cleanupCtx, nil); err != nil {
				workflow.GetLogger(ctx).Error("Failed to scale down deployment", "deployment", deploymentName, "error", err)
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

	// --- Step 2: Create agent deployment + PVC ---
	state.Message = "Creating agent deployment"

	deployInput := CreateAgentDeploymentInput{
		Name:                  fmt.Sprintf("agentrun-%s", input.AgentRunName),
		Namespace:             input.Namespace,
		AgentRunName:          input.AgentRunName,
		Repos:                 input.Repos,
		Prompt:                input.Prompt,
		DevboxConfig:          input.DevboxConfig,
		Image:                 input.Image,
		EnvVars:               input.EnvVars,
		LLMKey:                llmKey,
		LiteLLMBaseURL:        input.LiteLLMBaseURL,
		ModelID:               modelIDFromTier(input.ModelTier),
		SpecContent:           input.SpecContent,
		GitHubTokenSecretName: input.GitHubTokenSecretName,
	}

	var deployOutput CreateAgentDeploymentOutput
	if err := workflow.ExecuteActivity(actCtx, ActivityCreateAgentDeployment, deployInput).Get(ctx, &deployOutput); err != nil {
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
	podName = deployOutput.DeploymentName // used for WaitForHydration label lookup
	state.PodName = podName
	state.DeploymentName = deploymentName

	// --- Step 3: Wait for hydration ---
	state.Phase = "Hydrating"
	state.Message = "Waiting for workspace hydration"

	// WaitForHydration manages its own internal polling loop — disable Temporal
	// retries to avoid restarting the activity from scratch on transient worker
	// failures. The HeartbeatTimeout ensures the activity is detected as dead
	// quickly if the worker crashes, and the activity will be re-dispatched once.
	hydrationOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 1.0,
		},
	}
	hydrationCtx := workflow.WithActivityOptions(ctx, hydrationOpts)

	var hydrationOutput WaitForHydrationOutput
	if err := workflow.ExecuteActivity(hydrationCtx, ActivityWaitForHydration, WaitForHydrationInput{
		PodName:      podName,
		Namespace:    input.Namespace,
		AgentRunName: input.AgentRunName,
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
	podIP = hydrationOutput.PodIP

	// Use the workspace root as the working directory for multi-repo support.
	workspacePath := "/workspace"

	// --- Step 3b: Hydrate context from past work (knowledge system) ---
	// Degrades gracefully on failure. Timeout is generous because it embeds the
	// prompt via pgvector and writes a file to the pod PVC over the sidecar RPC.
	{
		hydrateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 2,
			},
		})
		repoURL := ""
		if len(input.Repos) > 0 {
			repoURL = input.Repos[0].URL
		}
		agentType := ""
		if input.OrchestrationMode == OrchestrationModeAuto {
			agentType = "senior"
		}
		var hydrateOutput HydrateContextOutput
		if err := workflow.ExecuteActivity(hydrateCtx, ActivityHydrateContext, HydrateContextInput{
			AgentRunID:    input.AgentRunName,
			WorkspacePath: workspacePath,
			Prompt:        input.Prompt,
			RepoURL:       repoURL,
			AgentType:     agentType,
			PodIP:         podIP,
		}).Get(hydrateCtx, &hydrateOutput); err != nil {
			workflow.GetLogger(ctx).Warn("Context hydration failed (proceeding without)", "error", err)
		} else if hydrateOutput.ContextWritten {
			workflow.GetLogger(ctx).Info("Context hydration complete — past work context written to workspace")
		}
	}

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
		case "Succeeded":
			// Enrich tags from git diff before cleanup
			enrichCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 2,
				},
			})
			repoPath := workspacePath
			if len(input.Repos) > 0 {
				rp := input.Repos[0].Path
				if rp == "" {
					rp = repoNameFromURL(input.Repos[0].URL)
				}
				repoPath = "/workspace/" + rp
			}
			if err := workflow.ExecuteActivity(enrichCtx, ActivityEnrichRunTags, EnrichRunTagsInput{
				AgentRunName: input.AgentRunName,
				Namespace:    input.Namespace,
				PodIP:        podIP,
				RepoPath:     repoPath,
			}).Get(ctx, nil); err != nil {
				workflow.GetLogger(ctx).Warn("Tag enrichment failed", "error", err)
			}

			// Push changes and optionally create a PR
			if input.AutoPush || input.AutoPR {
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
				commitMsg := fmt.Sprintf("feat: %s\n\nAgentRun: %s", truncateForTitle(input.Prompt, 72), input.AgentRunName)
				repoURL := ""
				if len(input.Repos) > 0 {
					repoURL = input.Repos[0].URL
				}

				var pushOutput PushChangesOutput
				if err := workflow.ExecuteActivity(gitCtx, ActivityPushChanges, PushChangesInput{
					AgentRunName:  input.AgentRunName,
					PodIP:         podIP,
					RepoPath:      repoPath,
					BranchName:    branchName,
					CommitMessage: commitMsg,
					RepoURL:       repoURL,
				}).Get(ctx, &pushOutput); err != nil {
					workflow.GetLogger(ctx).Warn("Push changes failed", "error", err)
				} else if input.AutoPR && len(input.Repos) > 0 {
					owner, repo, err := parseGitHubOwnerRepo(input.Repos[0].URL)
					if err != nil {
						workflow.GetLogger(ctx).Warn("Failed to parse repo URL for PR", "error", err)
					} else {
						baseBranch := input.PRBaseBranch
						if baseBranch == "" {
							baseBranch = "main"
						}
						prCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
							StartToCloseTimeout: 30 * time.Second,
							RetryPolicy: &temporal.RetryPolicy{
								MaximumAttempts: 2,
							},
						})
						// Build PR body with proposal content or prompt summary
						summary := pushOutput.ProposalContent
						if summary == "" {
							// Use first 300 chars of prompt as summary
							if len(input.Prompt) > 300 {
								summary = input.Prompt[:300] + "..."
							} else {
								summary = input.Prompt
							}
						}
						runURL := fmt.Sprintf("https://uncworks.stork-eel.ts.net/run/%s", input.AgentRunName)
						prBody := fmt.Sprintf("## Summary\n\n%s\n\n## Changes\n\n%s\n\n## Pipeline\n\n- **Run:** %s\n- **Model:** %s\n\n## Run\n\nView the full agent run: %s\n\n---\n*This PR was automatically created by UNCWORKS.*",
							summary,
							pushOutput.DiffStat,
							input.AgentRunName,
							input.ModelTier,
							runURL,
						)
						var prOutput CreatePROutput
						if err := workflow.ExecuteActivity(prCtx, ActivityCreatePR, CreatePRInput{
							RepoOwner:    owner,
							RepoName:     repo,
							BranchName:   branchName,
							BaseBranch:   baseBranch,
							Title:        fmt.Sprintf("feat: %s", truncateForTitle(input.Prompt, 60)),
							Body:         prBody,
							AgentRunName: input.AgentRunName,
						}).Get(ctx, &prOutput); err != nil {
							workflow.GetLogger(ctx).Warn("Create PR failed", "error", err)
						} else {
							state.PRUrl = prOutput.PRUrl
						}
					}
				}
			}
			return nil
		case "Failed", "Cancelled":
			// Pod cleanup happens via defer
			return nil
		}
	}
}

// SpawnJuniorInput contains parameters for spawning a child workflow.
type SpawnJuniorInput struct {
	ParentRunName         string
	Namespace             string
	Task                  string
	TaskName              string
	Repos                 []Repository
	DevboxConfig          string
	TTLSeconds            int32
	Image                 string
	EnvVars               map[string]string
	Blocking              bool
	ModelTier             string
	MaxBudget             float64
	LiteLLMBaseURL        string
	SpecRunID             string
	GitHubTokenSecretName string
}

// SpawnJuniorWorkflow starts a child AgentRunWorkflow for a junior agent.
func SpawnJuniorWorkflow(ctx workflow.Context, input SpawnJuniorInput) error {
	taskSuffix := input.TaskName
	if taskSuffix == "" {
		taskSuffix = fmt.Sprintf("%d", workflow.Now(ctx).UnixMilli()%100000)
	}
	juniorName := fmt.Sprintf("%s-junior-%s", input.ParentRunName, taskSuffix)

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: juniorName,
		TaskQueue:  TaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	childInput := WorkflowInput{
		AgentRunName:          juniorName,
		Namespace:             input.Namespace,
		Repos:                 input.Repos,
		Prompt:                input.Task,
		DevboxConfig:          input.DevboxConfig,
		TTLSeconds:            input.TTLSeconds,
		Image:                 input.Image,
		EnvVars:               input.EnvVars,
		ModelTier:             input.ModelTier,
		MaxBudget:             input.MaxBudget,
		LiteLLMBaseURL:        input.LiteLLMBaseURL,
		ParentRunID:           input.ParentRunName,
		SpecRunID:             input.SpecRunID,
		GitHubTokenSecretName: input.GitHubTokenSecretName,
		OrchestrationMode:     OrchestrationModeSingle, // juniors always run as single
	}

	future := workflow.ExecuteChildWorkflow(childCtx, AgentRunWorkflow, childInput)

	if input.Blocking {
		return future.Get(ctx, nil)
	}

	// Fire-and-forget: just wait for the child to start
	return future.GetChildWorkflowExecution().Get(ctx, nil)
}

const maxOrchestrationTasks = 7

// runManualOrchestration executes the manual orchestration path:
// spawn juniors directly from the user-defined task list, wait for all.
func runManualOrchestration(ctx workflow.Context, input WorkflowInput) error {
	state := &WorkflowState{
		Phase:   "Running",
		Message: "Manual orchestration: spawning junior tasks",
	}

	if err := workflow.SetQueryHandler(ctx, QueryGetState, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return fmt.Errorf("set query handler: %w", err)
	}

	tasks := input.Orchestration
	if len(tasks) > maxOrchestrationTasks {
		workflow.GetLogger(ctx).Warn("Truncating orchestration tasks", "count", len(tasks), "max", maxOrchestrationTasks)
		tasks = tasks[:maxOrchestrationTasks]
	}

	specRunID := input.SpecRunID
	if specRunID == "" {
		specRunID = input.AgentRunName
	}

	// Fan out juniors in parallel
	var futures []workflow.ChildWorkflowFuture
	for _, task := range tasks {
		repos := input.Repos
		if len(task.RepoURLs) > 0 {
			repos = make([]Repository, len(task.RepoURLs))
			for i, u := range task.RepoURLs {
				repos[i] = Repository{URL: u}
			}
		}

		taskSuffix := task.Name
		if taskSuffix == "" {
			taskSuffix = fmt.Sprintf("%d", workflow.Now(ctx).UnixMilli()%100000)
		}
		juniorName := fmt.Sprintf("%s-junior-%s", input.AgentRunName, taskSuffix)

		childOpts := workflow.ChildWorkflowOptions{
			WorkflowID: juniorName,
			TaskQueue:  TaskQueue,
		}
		childCtx := workflow.WithChildOptions(ctx, childOpts)

		childInput := WorkflowInput{
			AgentRunName:          juniorName,
			Namespace:             input.Namespace,
			Repos:                 repos,
			Prompt:                task.Prompt,
			DevboxConfig:          input.DevboxConfig,
			TTLSeconds:            input.TTLSeconds,
			Image:                 input.Image,
			EnvVars:               input.EnvVars,
			ModelTier:             input.ModelTier,
			MaxBudget:             input.MaxBudget,
			LiteLLMBaseURL:        input.LiteLLMBaseURL,
			ParentRunID:           input.AgentRunName,
			SpecRunID:             specRunID,
			GitHubTokenSecretName: input.GitHubTokenSecretName,
			OrchestrationMode:     OrchestrationModeSingle,
		}

		future := workflow.ExecuteChildWorkflow(childCtx, AgentRunWorkflow, childInput)
		futures = append(futures, future)
	}

	// Wait for all juniors
	var failedTasks []string
	for i, future := range futures {
		if err := future.Get(ctx, nil); err != nil {
			taskName := tasks[i].Name
			if taskName == "" {
				taskName = fmt.Sprintf("task-%d", i)
			}
			failedTasks = append(failedTasks, taskName)
			workflow.GetLogger(ctx).Warn("Junior task failed", "task", taskName, "error", err)
		}
	}

	if len(failedTasks) > 0 {
		state.Phase = "Failed"
		state.Message = fmt.Sprintf("Manual orchestration: %d/%d tasks failed: %s",
			len(failedTasks), len(tasks), strings.Join(failedTasks, ", "))
		return fmt.Errorf("orchestration failed: %s", strings.Join(failedTasks, ", "))
	}

	state.Phase = "Succeeded"
	state.Message = fmt.Sprintf("Manual orchestration: all %d tasks completed", len(tasks))
	return nil
}

// runAutoOrchestration executes the auto-decomposition path:
// start senior agent for decomposition, parse JSON plan, spawn juniors, integrate.
func runAutoOrchestration(ctx workflow.Context, input WorkflowInput) error {
	state := &WorkflowState{
		Phase:   "Running",
		Message: "Auto orchestration: decomposing spec",
	}

	if err := workflow.SetQueryHandler(ctx, QueryGetState, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return fmt.Errorf("set query handler: %w", err)
	}

	// Build decomposition prompt
	decompositionPrompt := fmt.Sprintf(`You are a senior engineer. Analyze the following spec and decompose it into
independent sub-tasks that can be executed in parallel by junior agents.

Output a JSON object with this schema:
{
  "tasks": [
    {
      "name": "short-kebab-case-name",
      "prompt": "Detailed task description for the junior agent",
      "repos": ["optional subset of repos relevant to this task"]
    }
  ],
  "integration_prompt": "Instructions for reviewing and integrating junior outputs"
}

Rules:
- Each task should be independently executable
- Each task should produce a clear, verifiable output (code change, test, etc.)
- Maximum 7 tasks (if more are needed, group related work)
- If the spec is simple enough for one agent, return {"tasks": []} and it will
  be executed as a single run

Here is the spec/prompt to decompose:

%s`, input.Prompt)

	// For auto mode, we simulate the decomposition by parsing from the prompt.
	// In a real implementation, this would start the senior agent, collect output,
	// and parse the JSON. For now, we fall back to single-run execution since
	// the agent infrastructure for collecting structured output requires the full
	// deployment pipeline. The senior agent integration is a future enhancement.
	//
	// Fallback: execute as single run with the original prompt.
	_ = decompositionPrompt // Used when full agent integration is wired up
	workflow.GetLogger(ctx).Info("Auto orchestration: falling back to single-run mode (structured output collection not yet wired)")

	state.Phase = "Running"
	state.Message = "Auto orchestration: executing as single run (decomposition pending)"

	// Re-run as single-agent workflow by delegating to the standard path
	singleInput := input
	singleInput.OrchestrationMode = OrchestrationModeSingle
	return AgentRunWorkflow(ctx, singleInput)
}

// parseDecompositionPlan parses a JSON decomposition plan from agent output.
// Returns nil if the JSON is malformed or empty tasks.
func parseDecompositionPlan(output string) *DecompositionPlan {
	// Try to extract JSON from the output (agent may include non-JSON text)
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start < 0 || end < 0 || end <= start {
		return nil
	}

	jsonStr := output[start : end+1]
	var plan DecompositionPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil
	}

	if len(plan.Tasks) == 0 {
		return nil
	}

	// Enforce max tasks
	if len(plan.Tasks) > maxOrchestrationTasks {
		plan.Tasks = plan.Tasks[:maxOrchestrationTasks]
	}

	return &plan
}

// modelIDFromTier maps a model tier name to a pi-coding-agent model identifier.
// LiteLLM exposes models as OpenAI-compatible, so we use the openai/ prefix.
// modelIDFromTier returns the model name to pass to pi-coding-agent.
// Pi has a built-in "litellm" provider that uses OPENAI_BASE_URL + OPENAI_API_KEY.
// The litellm/ prefix tells pi to route through the LiteLLM proxy.
func modelIDFromTier(tier string) string {
	if tier == "" {
		return "litellm/default"
	}
	return "litellm/" + tier
}
