# run-pipeline Specification

## Purpose
TBD - created by archiving change spec-driven-agent-runs. Update Purpose after archive.
## Requirements
### Requirement: Runs execute as a three-stage pipeline
The AgentRunWorkflow SHALL execute spec-driven runs as a sequential pipeline of Plan → Execute → Verify stages, where each stage is a separate agent invocation in the same workspace.

#### Scenario: Spec-driven run completes all three stages
- **WHEN** a run is created with `orchestrationMode: "spec-driven"`
- **THEN** the workflow executes Stage 1 (Plan), Stage 2 (Execute), and Stage 3 (Verify) in sequence
- **AND** the run phase transitions through Planning → Running → Verifying → Succeeded

#### Scenario: Run with specContent auto-upgrades to spec-driven
- **WHEN** a run is created with non-empty `specContent` and `orchestrationMode` is not explicitly set
- **THEN** the workflow automatically uses the spec-driven pipeline

### Requirement: Plan stage creates an OpenSpec change via CLI
The planning agent SHALL use the OpenSpec CLI and standard OpenSpec workflow to create a structured change from user input.

#### Scenario: Planning agent scaffolds change
- **WHEN** Stage 1 begins
- **THEN** the sidecar runs `openspec new change "<run-id>"` to scaffold the change directory
- **AND** the planning agent generates proposal.md, specs/*.md, and tasks.md

#### Scenario: Planning output validated
- **WHEN** the planning agent completes
- **THEN** `openspec validate --json` is run on the change
- **AND** `openspec status --json` confirms all required artifacts are complete
- **AND** the pipeline only proceeds to Stage 2 if both checks pass

### Requirement: Execute stage uses `/opsx:apply` pattern
The execution agent SHALL use the OpenSpec apply workflow to implement the change, tracking progress via tasks.md checkboxes.

#### Scenario: Agent implements tasks
- **WHEN** Stage 2 begins
- **THEN** the execution agent receives the change's tasks.md as its work plan
- **AND** the agent marks tasks as `[x]` in tasks.md as it completes them

#### Scenario: Task progress trackable during execution
- **WHEN** Stage 2 is running
- **THEN** `openspec list --json` returns the current `completedTasks/totalTasks` for the change

### Requirement: Verify stage uses OpenSpec CLI as primary gate
The verification stage SHALL use `openspec list --json` for task completion, `openspec validate --json` for structural validity, automated scenario checks, and LLM judge for semantic criteria — in that order.

#### Scenario: Verification succeeds and archives
- **WHEN** all verification gates pass
- **THEN** `openspec archive --yes` is executed to archive the change and merge specs
- **AND** the run is marked as Succeeded

#### Scenario: Verification fails on incomplete tasks
- **WHEN** `openspec list --json` reports the change as `"in-progress"` (tasks incomplete)
- **THEN** verification fails immediately
- **AND** Stage 2 is retried with the list of incomplete tasks as context

### Requirement: Failed verification triggers retry with context
The workflow SHALL retry Stage 2 (Execute) when Stage 3 (Verify) reports failure, up to a configurable maximum number of retries.

#### Scenario: Retry on verification failure
- **WHEN** Stage 3 reports verification failure
- **AND** the retry count is below the maximum (default: 3)
- **THEN** Stage 2 is re-executed with the verification failure report prepended to the agent's context
- **AND** the retry count is incremented

#### Scenario: Max retries exceeded
- **WHEN** Stage 3 reports verification failure
- **AND** the retry count has reached the maximum
- **THEN** the run is marked as Failed
- **AND** the run status includes the final verification report
- **AND** the OpenSpec change remains unarchived as the failure artifact

### Requirement: Run status reflects current stage
The AgentRunStatus SHALL include the current pipeline stage and retry count so that the API and UI can display progress accurately.

#### Scenario: Status during planning
- **WHEN** Stage 1 (Plan) is executing
- **THEN** the run phase is "Running" and the status message indicates "Planning: generating spec"

#### Scenario: Status during verification
- **WHEN** Stage 3 (Verify) is executing
- **THEN** the run phase is "Running" and the status message indicates "Verifying: evaluating against spec"

#### Scenario: Status includes retry count
- **WHEN** Stage 2 is retrying after a verification failure
- **THEN** the status message includes the current attempt number (e.g., "Executing: attempt 2/3")

### Requirement: Backward compatibility with single-stage runs
Runs with `orchestrationMode: "single"` SHALL continue to use the existing single-stage workflow without Plan or Verify stages.

#### Scenario: Single mode unchanged
- **WHEN** a run is created with `orchestrationMode: "single"`
- **THEN** the workflow executes the agent once without planning or verification
- **AND** the run succeeds when the agent process exits cleanly

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

