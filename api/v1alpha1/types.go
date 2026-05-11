// Package v1alpha1 contains the AgentRun CRD types for the AOT orchestrator.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrchestrationMode specifies how an agent run handles decomposition.
// +kubebuilder:validation:Enum=single;auto;manual;spec-driven
type OrchestrationMode string

const (
	OrchestrationModeSingle     OrchestrationMode = "single"
	OrchestrationModeAuto       OrchestrationMode = "auto"
	OrchestrationModeManual     OrchestrationMode = "manual"
	OrchestrationModeSpecDriven OrchestrationMode = "spec-driven"
)

// OrchestrationTask defines a single sub-task in a manual orchestration.
type OrchestrationTask struct {
	// Name is a short kebab-case identifier for the task.
	Name string `json:"name"`
	// Prompt is the task description for the junior agent.
	Prompt string `json:"prompt"`
	// RepoURLs optionally restricts which repos are cloned for this task.
	// +optional
	RepoURLs []string `json:"repoUrls,omitempty"`
}

// Orchestration contains the task list for manual orchestration mode.
type Orchestration struct {
	// Tasks is the list of sub-tasks to execute.
	Tasks []OrchestrationTask `json:"tasks"`
}

// BackendType specifies the execution backend for an AgentRun.
// +kubebuilder:validation:Enum=Pod
type BackendType string

const (
	BackendPod BackendType = "Pod"
)

// AgentRunPhase represents the lifecycle phase of an AgentRun.
// +kubebuilder:validation:Enum=Pending;Running;WaitingForInput;Succeeded;Failed;Cancelled
type AgentRunPhase string

const (
	AgentRunPhasePending         AgentRunPhase = "Pending"
	AgentRunPhaseRunning         AgentRunPhase = "Running"
	AgentRunPhaseWaitingForInput AgentRunPhase = "WaitingForInput"
	AgentRunPhaseSucceeded       AgentRunPhase = "Succeeded"
	AgentRunPhaseFailed          AgentRunPhase = "Failed"
	AgentRunPhaseCancelled       AgentRunPhase = "Cancelled"
)

// Repository specifies a git repository to clone into the agent workspace.
type Repository struct {
	// URL is the git repository URL.
	URL string `json:"url"`
	// Branch is the git branch to check out.
	// +optional
	Branch string `json:"branch,omitempty"`
	// Path is the directory name under /workspace/<repo>/. Derived from repo URL if empty.
	// +optional
	Path string `json:"path,omitempty"`
}

// AgentRunSpec defines the desired state of an AgentRun.
type AgentRunSpec struct {
	// Backend specifies the execution backend.
	// +kubebuilder:default=Pod
	Backend BackendType `json:"backend"`

	// Repos is the list of git repositories to clone into the workspace.
	Repos []Repository `json:"repos"`

	// Prompt is the task description for the agent.
	Prompt string `json:"prompt"`

	// DevboxConfig is the path to the devbox.json configuration.
	// +optional
	DevboxConfig string `json:"devboxConfig,omitempty"`

	// TTLSeconds is the maximum lifetime in seconds for this agent run.
	// +kubebuilder:default=3600
	// +optional
	TTLSeconds int32 `json:"ttlSeconds,omitempty"`

	// EnvVars are additional environment variables for the agent.
	// +optional
	EnvVars map[string]string `json:"envVars,omitempty"`

	// Image overrides the default agent container image.
	// +optional
	Image string `json:"image,omitempty"`

	// ModelTier controls LLM model routing through LiteLLM.
	// Options: "default" (Ollama local), "default-cloud" (OpenRouter free), "premium" (Anthropic/OpenAI).
	// +kubebuilder:default=default
	// +optional
	ModelTier string `json:"modelTier,omitempty"`

	// ManageModelTier is the model for plan/verify stages in spec-driven runs.
	// Falls back to ModelTier if empty.
	// +optional
	ManageModelTier string `json:"manageModelTier,omitempty"`

	// ImplementModelTier is the model for execute stages in spec-driven runs.
	// Falls back to ManageModelTier (then ModelTier) if empty.
	// +optional
	ImplementModelTier string `json:"implementModelTier,omitempty"`

	// SpecContent is the CodeSpeak .cs.md spec body (markdown).
	// +optional
	SpecContent string `json:"specContent,omitempty"`

	// SpecSource tracks where the spec came from: "editor", "github:<owner/repo/path>", etc.
	// +optional
	SpecSource string `json:"specSource,omitempty"`

	// ProjectRef is the name of the Project CRD this run belongs to.
	// When set, empty run fields are inherited from the project's defaults.
	// +optional
	ProjectRef string `json:"projectRef,omitempty"`

	// SpecRef is the name of a spec in the project's config repo (e.g., "add-comments").
	// Resolves to openspec/specs/{specRef}/spec.md in the project's soft-serve repo.
	// Requires ProjectRef to be set.
	// +optional
	SpecRef string `json:"specRef,omitempty"`

	// WorkspaceName is the name of the workspace preset used for this run.
	// +optional
	WorkspaceName string `json:"workspaceName,omitempty"`

	// ParentRunID links this junior run to its parent senior run.
	// +optional
	ParentRunID string `json:"parentRunID,omitempty"`

	// OrchestrationMode controls decomposition behavior: single (default), auto, or manual.
	// +kubebuilder:default=single
	// +optional
	OrchestrationMode OrchestrationMode `json:"orchestrationMode,omitempty"`

	// Orchestration defines the manual orchestration task list.
	// +optional
	Orchestration *Orchestration `json:"orchestration,omitempty"`

	// SpecRunID groups all runs from a single spec execution.
	// +optional
	SpecRunID string `json:"specRunID,omitempty"`

	// DisplayName is a human-readable name generated from the prompt by the LLM.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// PipelineConfig provides per-stage configuration for spec-driven runs.
	// +optional
	PipelineConfig *PipelineConfig `json:"pipelineConfig,omitempty"`

	// MaxBudget is the maximum LLM spend budget in USD.
	// +optional
	MaxBudget float64 `json:"maxBudget,omitempty"`

	// AutoPush controls whether changes are pushed to a feature branch after successful verification.
	// +optional
	AutoPush bool `json:"autoPush,omitempty"`

	// AutoPR controls whether a GitHub PR is created after pushing changes.
	// Requires AutoPush to be true.
	// +optional
	AutoPR bool `json:"autoPR,omitempty"`

	// PRBaseBranch is the base branch for the PR (default: "main").
	// +optional
	PRBaseBranch string `json:"prBaseBranch,omitempty"`

	// Project is the project this run belongs to.
	// +optional
	Project string `json:"project,omitempty"`

	// Feature is the feature/unit-of-work this run contributes to.
	// +optional
	Feature string `json:"feature,omitempty"`

	// Tags are freeform labels for cross-cutting filtering.
	// +optional
	Tags []string `json:"tags,omitempty"`

	// ApprovalMode controls what approval is required before a run is marked Succeeded.
	// "": no approval required (auto-succeed when agent exits 0)
	// "hitl": pause and wait for explicit human approval before marking Succeeded
	// "llm-judge": require LLM-based review of completed work to pass (future)
	// "hybrid": require both LLM review AND human approval (future)
	// +optional
	ApprovalMode string `json:"approvalMode,omitempty"`
}

// PipelineConfig provides per-stage configuration for the spec-driven pipeline.
type PipelineConfig struct {
	// Plan configures the planning stage.
	// +optional
	Plan StageConfig `json:"plan,omitempty"`
	// Execute configures the execution stage.
	// +optional
	Execute StageConfig `json:"execute,omitempty"`
	// Verify configures the verification stage.
	// +optional
	Verify StageConfig `json:"verify,omitempty"`
}

// StageConfig configures a single pipeline stage.
type StageConfig struct {
	// Model is the LiteLLM model name for this stage.
	// +optional
	Model string `json:"model,omitempty"`
	// TimeoutSeconds is the stage timeout.
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
	// MaxRetries is the max retries for this stage.
	// +optional
	MaxRetries int32 `json:"maxRetries,omitempty"`
	// OnFailure controls behavior when retries are exhausted: "retry", "fail", or "skip".
	// +optional
	OnFailure string `json:"onFailure,omitempty"`
}

// AgentRunStatus defines the observed state of an AgentRun.
type AgentRunStatus struct {
	// Phase is the current lifecycle phase.
	Phase AgentRunPhase `json:"phase,omitempty"`

	// Message provides human-readable status information.
	// +optional
	Message string `json:"message,omitempty"`

	// PodName is the name of the provisioned Pod (for Pod backend).
	// +optional
	PodName string `json:"podName,omitempty"`

	// TraceID is the OpenTelemetry trace ID for this run.
	// +optional
	TraceID string `json:"traceID,omitempty"`

	// WorktreePath is the path to the git worktree on the agent.
	// +optional
	WorktreePath string `json:"worktreePath,omitempty"`

	// StartedAt is when the agent started running.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is when the agent finished.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Conditions represent the latest available observations of the AgentRun's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LogOutput is the persisted agent log output (up to 1MB), collected before pod deletion.
	// +optional
	LogOutput string `json:"logOutput,omitempty"`

	// AgentJSONL is the full pi-coding-agent JSONL log (capped at 512 KB), collected before pod deletion.
	// Used as fallback for the structured-logs endpoint when the PVC has been deleted.
	// +optional
	AgentJSONL string `json:"agentJSONL,omitempty"`

	// RetainUntil is when the pod retention expires and cleanup will run.
	// +optional
	RetainUntil *metav1.Time `json:"retainUntil,omitempty"`

	// DeploymentName is the name of the Deployment managing the agent pod.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// DebugActive indicates whether a debug session is currently active.
	// +optional
	DebugActive bool `json:"debugActive,omitempty"`

	// Stage is the current pipeline stage for spec-driven runs (planning, executing, verifying).
	// Empty for non-spec-driven runs.
	// +optional
	Stage string `json:"stage,omitempty"`

	// RetryCount is the number of execute→verify retry attempts completed.
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// VerificationResult is the JSON-encoded verdict from the verification stage.
	// +optional
	VerificationResult string `json:"verificationResult,omitempty"`

	// PRUrl is the URL of the GitHub PR created by the pipeline.
	// +optional
	PRUrl string `json:"prUrl,omitempty"`

	// Archived indicates whether this run has been archived (hidden from default list).
	// +optional
	Archived bool `json:"archived,omitempty"`

	// TotalCost is the estimated total cost of this run (e.g., "$0.12").
	// +optional
	TotalCost string `json:"totalCost,omitempty"`

	// TotalAdditions is the aggregate number of lines added across all diffs.
	// +optional
	TotalAdditions int32 `json:"totalAdditions,omitempty"`

	// TotalDeletions is the aggregate number of lines deleted across all diffs.
	// +optional
	TotalDeletions int32 `json:"totalDeletions,omitempty"`

	// CIFixAttempts is the number of CI autofix attempts for this run's PR.
	// +optional
	CIFixAttempts int32 `json:"ciFixAttempts,omitempty"`

	// LastCIStatus is the most recent CI check status ("success", "failure").
	// +optional
	LastCIStatus string `json:"lastCIStatus,omitempty"`

	// ParentPRUrl is the URL of the PR this fix run is targeting.
	// +optional
	ParentPRUrl string `json:"parentPRUrl,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.backend`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentRun is the Schema for the agentruns API.
type AgentRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentRunSpec   `json:"spec,omitempty"`
	Status AgentRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentRunList contains a list of AgentRun.
type AgentRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentRun `json:"items"`
}
