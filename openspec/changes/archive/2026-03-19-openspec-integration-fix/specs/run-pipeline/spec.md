## MODIFIED Requirements

### Requirement: Plan stage uses full OpenSpec lifecycle
The plan stage SHALL initialize OpenSpec in the workspace, create a change via `openspec new change`, and validate the output via `openspec validate --json` and `openspec status --json` before proceeding to execute.

#### Scenario: OpenSpec initialized in workspace
- **WHEN** the plan stage starts
- **THEN** `openspec init --tools pi --force` is run in the workspace if no `openspec/config.yaml` exists

#### Scenario: Change created via CLI
- **WHEN** the planning agent completes
- **THEN** `openspec status --change <run-id> --json` is run
- **AND** all `applyRequires` artifacts have `status: "done"`
- **AND** if any artifact is not done, planning retries or fails

#### Scenario: Change validated after planning
- **WHEN** the planning agent completes
- **THEN** `openspec validate <run-id> --json` is run
- **AND** the change is reported as `valid: true`
- **AND** if invalid, planning retries with the validation errors as context

### Requirement: Pre-execute artifact check
The pipeline SHALL verify OpenSpec change artifacts exist before starting the execute stage.

#### Scenario: Artifacts verified before execute
- **WHEN** the plan stage completes successfully
- **THEN** the pipeline verifies that `proposal.md`, at least one `specs/*/spec.md`, and `tasks.md` exist
- **AND** if any artifact is missing, the pipeline fails with a clear error
