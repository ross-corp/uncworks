## ADDED Requirements

### Requirement: Controller logs and requeues on status update failure
The agent run controller SHALL log an error and return the error (triggering automatic requeue) when a `status.Update` call fails, rather than silently discarding the failure.

#### Scenario: Status update failure triggers requeue
- **WHEN** `status.Update` returns an error (e.g., conflict, connection refused)
- **THEN** the error is logged with the resource name and error message
- **THEN** the reconcile function returns the error, causing controller-runtime to requeue

### Requirement: Controller requeues on transient errors
The agent run and chain run controllers SHALL return `ctrl.Result{RequeueAfter: 10 * time.Second}` when encountering transient errors (resource not found for dependencies, network errors) rather than returning nil and dropping the item.

#### Scenario: Dependency not found triggers requeue
- **WHEN** a referenced RunTemplate or Project is not found during reconcile
- **THEN** the controller returns a result with RequeueAfter set
- **THEN** reconcile retries after the backoff period

### Requirement: Schedule active list is reconciled
The schedule controller SHALL remove references to completed or failed `ChainRun` resources from `Schedule.status.active` during each reconcile pass, keeping the active list accurate.

#### Scenario: Completed chain run removed from active list
- **WHEN** a `ChainRun` in `Schedule.status.active` has phase `succeeded` or `failed`
- **THEN** the schedule controller removes that reference from `.status.active`
- **THEN** the updated status is persisted

### Requirement: Embedding failures are propagated
The knowledge activities SHALL return an error when embedding fails rather than returning empty output with a nil error. This allows Temporal to retry the activity according to its retry policy.

#### Scenario: Embedding failure causes activity to fail
- **WHEN** the embedder returns an error
- **THEN** the activity returns that error
- **THEN** Temporal retries the activity per the configured retry policy
