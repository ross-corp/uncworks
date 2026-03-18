## Purpose

Define how Stage 1 (Plan) converts user input of any fidelity — from a one-line prompt to a detailed specification — into a structured OpenSpec change using the same CLI and workflow that humans use.

## ADDED Requirements

### Requirement: Planning agent uses OpenSpec CLI to scaffold changes
The planning agent SHALL use `openspec new change "<run-id>"` to create the change directory, then generate artifacts using the standard OpenSpec workflow (proposal → specs → tasks).

#### Scenario: CLI scaffolds change
- **WHEN** Stage 1 begins
- **THEN** `openspec new change "<run-id>"` is executed in the workspace
- **AND** the change directory exists at the OpenSpec-configured path with `.openspec.yaml`

#### Scenario: Agent generates all required artifacts
- **WHEN** the planning agent completes
- **THEN** `openspec status --change "<run-id>" --json` reports all `applyRequires` artifacts as `"done"`

### Requirement: Planning agent normalizes any user input into an OpenSpec change
The planning agent SHALL accept user input ranging from a vague prompt to a full specification and produce a valid OpenSpec change with proposal, specs (including WHEN/THEN scenarios), and tasks.

#### Scenario: Vague prompt produces valid spec
- **WHEN** the user input is "fix the login bug"
- **THEN** the planning agent produces an OpenSpec change with at least one spec file containing WHEN/THEN scenarios
- **AND** `openspec validate --json` reports the change as valid

#### Scenario: Detailed spec refines into OpenSpec format
- **WHEN** the user provides specContent with detailed requirements
- **THEN** the planning agent incorporates the user's requirements into WHEN/THEN scenarios
- **AND** the user's original intent is preserved in the spec

#### Scenario: Planning uses repo context
- **WHEN** the planning agent generates a spec
- **THEN** it reads the repository structure and relevant files to produce context-aware acceptance criteria (referencing actual file paths, test commands, build tools)

### Requirement: Generated specs include machine-checkable criteria
The planning agent SHALL produce acceptance criteria that include at least one automated-checkable scenario per spec (file existence, command execution, or pattern match), in addition to any semantic criteria.

#### Scenario: Spec includes command-based check
- **WHEN** the generated spec involves code changes to a project with tests
- **THEN** at least one scenario includes a WHEN/THEN that references running the project's test suite
- **AND** the scenario specifies the exact command to run (e.g., `npm test`, `go test ./...`)

#### Scenario: Spec includes file existence check
- **WHEN** the generated spec involves creating new files
- **THEN** at least one scenario specifies the expected file path in the WHEN/THEN clause

### Requirement: Planning agent validates output via OpenSpec CLI before proceeding
The planning agent's output SHALL be validated through `openspec validate --json` AND `openspec status --json` before the pipeline proceeds to Stage 2.

#### Scenario: Valid and complete spec proceeds
- **WHEN** `openspec validate --json` reports `valid: true`
- **AND** `openspec status --json` reports all `applyRequires` artifacts as `"done"`
- **THEN** the pipeline proceeds to Stage 2

#### Scenario: Invalid spec retries planning
- **WHEN** `openspec validate --json` reports errors
- **THEN** the pipeline retries Stage 1 (up to 2 attempts) with the validation errors as context
- **AND** if retries are exhausted, the run fails with a "Planning validation failed" message

#### Scenario: Incomplete artifacts retry planning
- **WHEN** `openspec status --json` reports artifacts still `"blocked"` or `"ready"` (not `"done"`)
- **THEN** the pipeline retries Stage 1 with the missing artifacts listed as context

### Requirement: Planning stage completes within time budget
The planning stage SHALL complete within 2 minutes. If the planning agent exceeds this time, the stage fails.

#### Scenario: Planning timeout
- **WHEN** the planning agent has been running for more than 2 minutes
- **THEN** the planning stage is terminated
- **AND** the run fails with a "Planning timeout" message
