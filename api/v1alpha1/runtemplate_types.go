package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunTemplateSpec defines a reusable, named run configuration.
type RunTemplateSpec struct {
	// DisplayName is the human-readable template name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description explains what this template does.
	// +optional
	Description string `json:"description,omitempty"`

	// ProjectRef references a Project for repo/default inheritance.
	// +optional
	ProjectRef string `json:"projectRef,omitempty"`

	// Repos are the source code repositories.
	// +optional
	Repos []Repository `json:"repos,omitempty"`

	// Prompt is the agent instruction.
	Prompt string `json:"prompt,omitempty"`

	// ModelTier controls LLM model routing.
	// +optional
	ModelTier string `json:"modelTier,omitempty"`

	// ManageModelTier is the model for plan/verify stages.
	// +optional
	ManageModelTier string `json:"manageModelTier,omitempty"`

	// ImplementModelTier is the model for execute stages.
	// +optional
	ImplementModelTier string `json:"implementModelTier,omitempty"`

	// OrchestrationMode controls the execution mode.
	// +optional
	OrchestrationMode OrchestrationMode `json:"orchestrationMode,omitempty"`

	// TTLSeconds is the run timeout.
	// +optional
	TTLSeconds int32 `json:"ttlSeconds,omitempty"`

	// AutoPush enables automatic git push after success.
	// +optional
	AutoPush bool `json:"autoPush,omitempty"`

	// AutoPR enables automatic PR creation after success.
	// +optional
	AutoPR bool `json:"autoPR,omitempty"`

	// PRBaseBranch is the target branch for auto-created PRs.
	// +optional
	PRBaseBranch string `json:"prBaseBranch,omitempty"`

	// SpecRef references a spec in the project's config repo.
	// +optional
	SpecRef string `json:"specRef,omitempty"`
}

// RunTemplateStatus defines the observed state of a RunTemplate.
type RunTemplateStatus struct {
	// LastTriggeredAt is when this template was last used to create a run.
	// +optional
	LastTriggeredAt *metav1.Time `json:"lastTriggeredAt,omitempty"`

	// RunCount is the total number of runs created from this template.
	RunCount int32 `json:"runCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelTier`
// +kubebuilder:printcolumn:name="Runs",type=integer,JSONPath=`.status.runCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RunTemplate is a reusable, named run configuration.
type RunTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RunTemplateSpec   `json:"spec,omitempty"`
	Status            RunTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunTemplateList contains a list of RunTemplates.
type RunTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunTemplate{}, &RunTemplateList{})
}
