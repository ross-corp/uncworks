## ADDED Requirements

### Requirement: Schedule CRD triggers Chains or RunTemplates on a cron expression
The system SHALL provide a Schedule custom resource that references either a Chain or a RunTemplate and fires on a cron expression. The Schedule SHALL support suspend/resume, concurrency policies, and history limits. The cron format SHALL follow the standard 5-field syntax (minute hour day-of-month month day-of-week).

#### Scenario: Create a Schedule for a RunTemplate
- **WHEN** a user creates a Schedule with cronExpression "0 2 * * 1" and runTemplateRef "weekly-dep-update"
- **THEN** the system persists the Schedule
- **AND** the Schedule status shows phase "Active" with nextFireTime computed from the cron expression

#### Scenario: Create a Schedule for a Chain
- **WHEN** a user creates a Schedule with cronExpression "0 0 * * *" and chainRef "nightly-review-chain"
- **THEN** the system persists the Schedule
- **AND** the Schedule status shows phase "Active" with nextFireTime at midnight

#### Scenario: Schedule validation rejects invalid cron expression
- **WHEN** a user creates a Schedule with cronExpression "not-a-cron"
- **THEN** the API returns a validation error: "invalid cron expression"

#### Scenario: Schedule validation rejects dual reference
- **WHEN** a user creates a Schedule with both runTemplateRef and chainRef set
- **THEN** the API returns a validation error: "exactly one of runTemplateRef or chainRef must be set"

#### Scenario: Schedule validation rejects no reference
- **WHEN** a user creates a Schedule with neither runTemplateRef nor chainRef set
- **THEN** the API returns a validation error: "exactly one of runTemplateRef or chainRef must be set"

### Requirement: Schedule controller fires on cron tick
The schedule controller SHALL evaluate all active Schedules on a periodic tick (default 60 seconds). When the current time passes a Schedule's nextFireTime, the controller SHALL create an AgentRun (for RunTemplate targets) or a ChainRun (for Chain targets).

#### Scenario: Cron tick creates an AgentRun
- **WHEN** the controller ticks and Schedule "weekly-dep-update" has nextFireTime in the past
- **THEN** the controller creates an AgentRun from the referenced RunTemplate
- **AND** updates the Schedule status: lastFireTime = now, nextFireTime = next cron occurrence
- **AND** increments the Schedule status executionCount

#### Scenario: Cron tick creates a ChainRun
- **WHEN** the controller ticks and Schedule "nightly-review" has nextFireTime in the past and chainRef is set
- **THEN** the controller creates a ChainRun from the referenced Chain
- **AND** updates the Schedule status accordingly

#### Scenario: Concurrency policy Forbid skips overlapping runs
- **WHEN** the controller ticks and the Schedule's concurrencyPolicy is "Forbid"
- **AND** the previous triggered run (AgentRun or ChainRun) is still Running
- **THEN** the controller skips this fire
- **AND** records a condition message: "Skipped: previous run still active"
- **AND** advances nextFireTime to the next cron occurrence

#### Scenario: Concurrency policy Replace cancels the active run
- **WHEN** the controller ticks and the Schedule's concurrencyPolicy is "Replace"
- **AND** the previous triggered run is still Running
- **THEN** the controller cancels the active run
- **AND** creates a new run from the referenced template or chain

#### Scenario: Concurrency policy Allow permits parallel runs
- **WHEN** the controller ticks and the Schedule's concurrencyPolicy is "Allow"
- **AND** the previous triggered run is still Running
- **THEN** the controller creates a new run without cancelling the active one

### Requirement: Schedule suspend and resume
The Schedule CRD SHALL support a `suspend` boolean field. When suspend is true, the controller SHALL skip all cron evaluations for this Schedule.

#### Scenario: Suspend a Schedule
- **WHEN** a user sets spec.suspend to true on a Schedule
- **THEN** the controller stops evaluating the cron expression for this Schedule
- **AND** the Schedule status phase transitions to "Suspended"

#### Scenario: Resume a Schedule
- **WHEN** a user sets spec.suspend to false on a previously suspended Schedule
- **THEN** the controller resumes cron evaluation
- **AND** computes the next fire time from the current time (does not retroactively fire missed ticks)
- **AND** the Schedule status phase transitions to "Active"

### Requirement: Schedule history limits
The Schedule CRD SHALL support `successfulRunHistoryLimit` and `failedRunHistoryLimit` fields (defaulting to 3 each). The controller SHALL garbage-collect completed runs that exceed these limits.

#### Scenario: History limit prunes old successful runs
- **WHEN** a Schedule fires and the number of completed successful runs exceeds successfulRunHistoryLimit
- **THEN** the controller deletes the oldest successful runs beyond the limit

#### Scenario: History limit prunes old failed runs
- **WHEN** a Schedule fires and the number of completed failed runs exceeds failedRunHistoryLimit
- **THEN** the controller deletes the oldest failed runs beyond the limit

### Requirement: Schedule CRUD API
The REST API SHALL expose endpoints for creating, listing, getting, updating, and deleting Schedules.

#### Scenario: List Schedules
- **WHEN** a user calls GET /api/v1/schedules
- **THEN** the system returns all Schedules with their cron expressions, targets, and status

#### Scenario: Suspend a Schedule via API
- **WHEN** a user calls POST /api/v1/schedules/{name}/suspend
- **THEN** the system sets spec.suspend to true on the Schedule

#### Scenario: Resume a Schedule via API
- **WHEN** a user calls POST /api/v1/schedules/{name}/resume
- **THEN** the system sets spec.suspend to false on the Schedule

#### Scenario: Trigger a Schedule immediately
- **WHEN** a user calls POST /api/v1/schedules/{name}/trigger
- **THEN** the system creates a run immediately regardless of the cron expression
- **AND** does not affect the regular cron schedule
