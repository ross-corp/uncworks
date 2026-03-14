// Package v1alpha1 contains the AgentRun CRD types for the AOT orchestrator.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackendType specifies the execution backend for an AgentRun.
// +kubebuilder:validation:Enum=Pod;KubeVirt;External
type BackendType string

const (
	BackendPod      BackendType = "Pod"
	BackendKubeVirt BackendType = "KubeVirt"
	BackendExternal BackendType = "External"
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
	// Path is the directory name under /workspace/src/. Derived from repo URL if empty.
	// +optional
	Path string `json:"path,omitempty"`
}

// AgentRunSpec defines the desired state of an AgentRun.
type AgentRunSpec struct {
	// Backend specifies the execution backend (Pod, KubeVirt, or External).
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

	// ExternalConfig is the configuration for External backend (SSH/Lima).
	// +optional
	ExternalConfig *ExternalBackendConfig `json:"externalConfig,omitempty"`

	// KubeVirtConfig is the configuration for KubeVirt backend.
	// +optional
	KubeVirtConfig *KubeVirtBackendConfig `json:"kubeVirtConfig,omitempty"`

	// ModelTier controls LLM model routing through LiteLLM.
	// Options: "default" (Ollama local), "default-cloud" (OpenRouter free), "premium" (Anthropic/OpenAI).
	// +kubebuilder:default=default
	// +optional
	ModelTier string `json:"modelTier,omitempty"`

	// SpecContent is the CodeSpeak .cs.md spec body (markdown).
	// +optional
	SpecContent string `json:"specContent,omitempty"`

	// SpecSource tracks where the spec came from: "editor", "github:<owner/repo/path>", etc.
	// +optional
	SpecSource string `json:"specSource,omitempty"`

	// WorkspaceName is the name of the workspace preset used for this run.
	// +optional
	WorkspaceName string `json:"workspaceName,omitempty"`
}

// ExternalBackendConfig holds configuration for the External (SSH/Lima) backend.
type ExternalBackendConfig struct {
	// Host is the SSH host address.
	Host string `json:"host"`
	// Port is the SSH port.
	// +kubebuilder:default=22
	Port int32 `json:"port,omitempty"`
	// User is the SSH user.
	User string `json:"user"`
	// SSHKeySecret references a Secret containing the SSH private key.
	SSHKeySecret string `json:"sshKeySecret"`
}

// KubeVirtBackendConfig holds configuration for the KubeVirt backend.
type KubeVirtBackendConfig struct {
	// CPUs is the number of vCPUs for the VM.
	// +kubebuilder:default=2
	CPUs int32 `json:"cpus,omitempty"`
	// MemoryMB is the memory in MB for the VM.
	// +kubebuilder:default=4096
	MemoryMB int32 `json:"memoryMB,omitempty"`
	// DiskGB is the disk size in GB for the VM.
	// +kubebuilder:default=20
	DiskGB int32 `json:"diskGB,omitempty"`
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

	// RetainUntil is when the pod retention expires and cleanup will run.
	// +optional
	RetainUntil *metav1.Time `json:"retainUntil,omitempty"`

	// DeploymentName is the name of the Deployment managing the agent pod.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// DebugActive indicates whether a debug session is currently active.
	// +optional
	DebugActive bool `json:"debugActive,omitempty"`
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
