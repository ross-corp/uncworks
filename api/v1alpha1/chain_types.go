package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChainStep defines a single step in a chain DAG.
type ChainStep struct {
	// Name is the unique identifier for this step within the chain.
	Name string `json:"name"`

	// TemplateRef references a RunTemplate by name.
	TemplateRef string `json:"templateRef"`

	// DependsOn lists step names that must complete before this step starts.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// ContextFrom names a step whose output summary is injected into this step's prompt.
	// +optional
	ContextFrom string `json:"contextFrom,omitempty"`

	// BranchFrom names a step whose git branch is used as this step's source branch.
	// +optional
	BranchFrom string `json:"branchFrom,omitempty"`

	// Condition is a CEL expression evaluated against parent step results.
	// If empty, the step runs unconditionally when dependencies are met.
	// +optional
	Condition string `json:"condition,omitempty"`
}

// ChainSpec defines a DAG of steps.
type ChainSpec struct {
	// DisplayName is the human-readable chain name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description explains what this chain does.
	// +optional
	Description string `json:"description,omitempty"`

	// ProjectRef references a Project for context.
	// +optional
	ProjectRef string `json:"projectRef,omitempty"`

	// Steps defines the DAG of execution steps.
	Steps []ChainStep `json:"steps"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Steps",type=integer,JSONPath=`.spec.steps`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Chain defines a DAG of run steps.
type Chain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ChainSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ChainList contains a list of Chains.
type ChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chain `json:"items"`
}

// ChainRunStepStatus tracks the execution status of a single chain step.
type ChainRunStepStatus struct {
	// Name matches the ChainStep.Name.
	Name string `json:"name"`

	// Phase is the step's current state.
	Phase string `json:"phase"` // pending, running, succeeded, failed, skipped

	// RunID is the AgentRun created for this step.
	// +optional
	RunID string `json:"runId,omitempty"`

	// StartedAt is when this step's run was created.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is when this step's run finished.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Message provides context about the step's state.
	// +optional
	Message string `json:"message,omitempty"`
}

// ChainRunSpec defines the parameters for a chain execution.
type ChainRunSpec struct {
	// ChainRef references the Chain definition by name.
	ChainRef string `json:"chainRef"`

	// TriggeredBy describes what triggered this chain run.
	// +optional
	TriggeredBy string `json:"triggeredBy,omitempty"` // "schedule:weekly-review", "manual", "webhook"
}

// ChainRunStatus defines the observed state of a ChainRun.
type ChainRunStatus struct {
	// Phase is the overall chain execution state.
	Phase string `json:"phase,omitempty"` // pending, running, succeeded, failed, cancelled

	// Steps tracks per-step execution status.
	// +optional
	Steps []ChainRunStepStatus `json:"steps,omitempty"`

	// StartedAt is when the chain run started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is when the chain run finished.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Message provides context about the chain run state.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Chain",type=string,JSONPath=`.spec.chainRef`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ChainRun is an instance of a Chain execution.
type ChainRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ChainRunSpec   `json:"spec,omitempty"`
	Status            ChainRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChainRunList contains a list of ChainRuns.
type ChainRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChainRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chain{}, &ChainList{})
	SchemeBuilder.Register(&ChainRun{}, &ChainRunList{})
}
