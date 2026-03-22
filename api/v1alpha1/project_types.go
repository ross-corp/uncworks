package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	// DisplayName is the human-readable project name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description is a short summary of the project.
	// +optional
	Description string `json:"description,omitempty"`

	// Repos are the application source code repositories (GitHub).
	Repos []Repository `json:"repos,omitempty"`

	// Devbox defines packages to install in every workspace via devbox.
	// +optional
	Devbox *DevboxConfig `json:"devbox,omitempty"`

	// Defaults are the default run configuration inherited by project runs.
	// +optional
	Defaults *ProjectDefaults `json:"defaults,omitempty"`

	// IDE configures the browser-based IDE for this project.
	// +optional
	IDE *IDEConfig `json:"ide,omitempty"`

	// SSH configures SSH access to the project workspace.
	// +optional
	SSH *SSHConfig `json:"ssh,omitempty"`
}

// DevboxConfig defines devbox packages for the project workspace.
type DevboxConfig struct {
	// Packages is the list of Nix packages to install (e.g., "go@1.22", "nodejs@20").
	Packages []string `json:"packages,omitempty"`
}

// ProjectDefaults are default values inherited by runs that reference this project.
type ProjectDefaults struct {
	// ModelTier is the default model for runs.
	// +optional
	ModelTier string `json:"modelTier,omitempty"`

	// ManageModelTier is the default model for plan/verify stages.
	// +optional
	ManageModelTier string `json:"manageModelTier,omitempty"`

	// ImplementModelTier is the default model for execute stages.
	// +optional
	ImplementModelTier string `json:"implementModelTier,omitempty"`

	// TTLSeconds is the default run timeout.
	// +optional
	TTLSeconds int32 `json:"ttlSeconds,omitempty"`

	// OrchestrationMode is the default orchestration mode.
	// +optional
	OrchestrationMode string `json:"orchestrationMode,omitempty"`

	// AutoPush enables automatic git push after successful runs.
	// +optional
	AutoPush bool `json:"autoPush,omitempty"`

	// AutoPR enables automatic PR creation after successful runs.
	// +optional
	AutoPR bool `json:"autoPR,omitempty"`

	// PRBaseBranch is the target branch for auto-created PRs.
	// +optional
	PRBaseBranch string `json:"prBaseBranch,omitempty"`
}

// IDEConfig defines the browser IDE settings for a project.
type IDEConfig struct {
	// Enabled controls whether IDE pods can be created for this project.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image is the Docker image for the IDE pod.
	// +optional
	Image string `json:"image,omitempty"`

	// IdleTimeoutMinutes is how long the IDE pod runs without activity before scaling to 0.
	// +optional
	IdleTimeoutMinutes int32 `json:"idleTimeoutMinutes,omitempty"`
}

// SSHConfig defines SSH access settings for a project.
type SSHConfig struct {
	// Enabled controls whether SSH access is available.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// AuthorizedKeys is the list of SSH public keys allowed to connect.
	AuthorizedKeys []string `json:"authorizedKeys,omitempty"`
}

// ProjectStatus defines the observed state of a Project.
type ProjectStatus struct {
	// ConfigRepoReady indicates the soft-serve repo has been created and scaffolded.
	ConfigRepoReady bool `json:"configRepoReady,omitempty"`

	// ConfigRepoURL is the in-cluster URL for the project config repo.
	// +optional
	ConfigRepoURL string `json:"configRepoURL,omitempty"`

	// IDEActive indicates whether the IDE pod is currently running.
	IDEActive bool `json:"ideActive,omitempty"`

	// IDEPodName is the name of the IDE pod if active.
	// +optional
	IDEPodName string `json:"idePodName,omitempty"`

	// RunCount is the total number of runs for this project.
	RunCount int32 `json:"runCount,omitempty"`

	// LastRunID is the ID of the most recent run.
	// +optional
	LastRunID string `json:"lastRunId,omitempty"`

	// LastRunAt is when the most recent run was created.
	// +optional
	LastRunAt *metav1.Time `json:"lastRunAt,omitempty"`

	// TotalCost is the aggregated estimated cost across all runs.
	// +optional
	TotalCost string `json:"totalCost,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Repos",type=integer,JSONPath=`.status.runCount`
// +kubebuilder:printcolumn:name="Config Ready",type=boolean,JSONPath=`.status.configRepoReady`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec   `json:"spec,omitempty"`
	Status            ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Projects.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
