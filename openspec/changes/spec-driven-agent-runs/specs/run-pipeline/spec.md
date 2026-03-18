## Purpose

Define the multi-stage agent run pipeline that replaces the current single-stage "run until exit" model. The pipeline ensures that agent runs are planned, executed, and verified against structured acceptance criteria before being marked as successful.

## ADDED Requirements

### Requirement: Runs execute as a three-stage pipeline
The AgentRunWorkflow SHALL execute spec-driven runs as a sequential pipeline of Plan → Execute → Verify stages, where each stage is a separate agent invocation in the same workspace.

#### Scenario: Spec-driven run completes all three stages
- **WHEN** a run is created with `orchestrationMode: "spec-driven"`
- **THEN** the workflow executes Stage 1 (Plan), Stage 2 (Execute), and Stage 3 (Verify) in sequence
- **AND** the run phase transitions through Planning → Running → Verifying → Succeeded

#### Scenario: Run with specContent auto-upgrades to spec-driven
- **WHEN** a run is created with non-empty `specContent` and `orchestrationMode` is not explicitly set
- **THEN** the workflow automatically uses the spec-driven pipeline

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
- **AND** the run status message includes the final verification report

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

### Requirement: Each run produces an OpenSpec change as documentation
The pipeline SHALL create an OpenSpec change directory on the workspace PVC for each spec-driven run, containing the plan artifacts (proposal, specs, tasks) and verification results.

#### Scenario: OpenSpec change created during planning
- **WHEN** Stage 1 (Plan) completes
- **THEN** an OpenSpec change exists at `/workspace/.openspec/changes/<run-id>/` with at minimum `proposal.md` and `specs/` directory
- **AND** `openspec validate --json` reports the change as valid

#### Scenario: Succeeded run archives the change
- **WHEN** a spec-driven run succeeds verification
- **THEN** the OpenSpec change is archived via `openspec archive`

### Requirement: Backward compatibility with single-stage runs
Runs with `orchestrationMode: "single"` SHALL continue to use the existing single-stage workflow without Plan or Verify stages.

#### Scenario: Single mode unchanged
- **WHEN** a run is created with `orchestrationMode: "single"`
- **THEN** the workflow executes the agent once without planning or verification
- **AND** the run succeeds when the agent process exits cleanly
