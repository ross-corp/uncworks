package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScheduleSpec defines a cron-triggered run or chain.
type ScheduleSpec struct {
	// DisplayName is the human-readable schedule name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Cron is the cron expression (standard 5-field format).
	Cron string `json:"cron"`

	// Timezone is the IANA timezone for the cron expression.
	// +optional
	// +kubebuilder:default="UTC"
	Timezone string `json:"timezone,omitempty"`

	// Suspend stops future executions when true.
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// ConcurrencyPolicy controls behavior when a previous run is still active.
	// +optional
	// +kubebuilder:default="Forbid"
	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`

	// ChainRef triggers a Chain on schedule. Mutually exclusive with TemplateRef.
	// +optional
	ChainRef string `json:"chainRef,omitempty"`

	// TemplateRef triggers a single RunTemplate on schedule. Mutually exclusive with ChainRef.
	// +optional
	TemplateRef string `json:"templateRef,omitempty"`

	// SuccessfulRunsHistoryLimit is how many successful runs to keep.
	// +optional
	// +kubebuilder:default=5
	SuccessfulRunsHistoryLimit int32 `json:"successfulRunsHistoryLimit,omitempty"`

	// FailedRunsHistoryLimit is how many failed runs to keep.
	// +optional
	// +kubebuilder:default=3
	FailedRunsHistoryLimit int32 `json:"failedRunsHistoryLimit,omitempty"`
}

// ScheduleStatus defines the observed state of a Schedule.
type ScheduleStatus struct {
	// LastScheduledTime is when the schedule last fired.
	// +optional
	LastScheduledTime *metav1.Time `json:"lastScheduledTime,omitempty"`

	// LastRunID is the ID of the most recent run or chain run.
	// +optional
	LastRunID string `json:"lastRunId,omitempty"`

	// LastResult is the outcome of the most recent run.
	// +optional
	LastResult string `json:"lastResult,omitempty"` // succeeded, failed, running

	// NextScheduleTime is when the schedule will next fire.
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// Active lists currently running run/chain-run IDs.
	// +optional
	Active []string `json:"active,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cron",type=string,JSONPath=`.spec.cron`
// +kubebuilder:printcolumn:name="Suspended",type=boolean,JSONPath=`.spec.suspend`
// +kubebuilder:printcolumn:name="Last Result",type=string,JSONPath=`.status.lastResult`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Schedule triggers runs or chains on a cron schedule.
type Schedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScheduleSpec   `json:"spec,omitempty"`
	Status            ScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScheduleList contains a list of Schedules.
type ScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Schedule{}, &ScheduleList{})
}
