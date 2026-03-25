package v1alpha1

// ChainRunPhase constants for ChainRun.Status.Phase.
const (
	ChainRunPhasePending   = "pending"
	ChainRunPhaseRunning   = "running"
	ChainRunPhaseSucceeded = "succeeded"
	ChainRunPhaseFailed    = "failed"
	ChainRunPhaseCancelled = "cancelled"
)

// ChainRunStepPhase constants for ChainRunStepStatus.Phase.
const (
	ChainRunStepPhasePending   = "pending"
	ChainRunStepPhaseRunning   = "running"
	ChainRunStepPhaseSucceeded = "succeeded"
	ChainRunStepPhaseFailed    = "failed"
	ChainRunStepPhaseSkipped   = "skipped"
)

// ScheduleLastResult constants for Schedule.Status.LastResult.
const (
	ScheduleLastResultRunning   = "running"
	ScheduleLastResultSucceeded = "succeeded"
	ScheduleLastResultFailed    = "failed"
)
