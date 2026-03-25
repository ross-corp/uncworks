## ADDED Requirements

### Requirement: Run structural checks before the manage agent review
The system SHALL run all automated structural checks (task completion, spec validation, file existence, test commands) as a first tier before starting the manage agent review session. If any structural check fails, the system SHALL skip the manage agent review entirely and return a verification failure immediately.

#### Scenario: All structural checks pass, manage agent review proceeds
- **WHEN** task completion, spec validation, file existence, and test command checks all pass
- **THEN** the system proceeds to start the manage agent review session (Tier 2)
- **AND** the structural check results are included in the `AutomatedChecks` array of the verification result

#### Scenario: Task completion check fails, manage agent review is skipped
- **WHEN** the task completion check fails (incomplete tasks, no tasks found, or `openspec list` error)
- **THEN** the verification result is returned immediately with `pass: false`
- **AND** the manage agent review session is NOT started
- **AND** the `FailureReport` describes the specific task completion failure

#### Scenario: Spec validation check fails, manage agent review is skipped
- **WHEN** the spec validation check fails (invalid spec structure, missing required fields)
- **THEN** the verification result is returned immediately with `pass: false`
- **AND** the manage agent review session is NOT started
- **AND** the `FailureReport` describes the validation issues

#### Scenario: File existence check fails, manage agent review is skipped
- **WHEN** a required file specified in a spec scenario does not exist in the workspace
- **THEN** the verification result is returned immediately with `pass: false`
- **AND** the manage agent review session is NOT started
- **AND** the `FailureReport` identifies the missing file and the scenario that requires it

#### Scenario: Test command check fails, manage agent review is skipped
- **WHEN** an automated test command extracted from the specs fails (non-zero exit code)
- **THEN** the verification result is returned immediately with `pass: false`
- **AND** the manage agent review session is NOT started
- **AND** the `FailureReport` includes the failing command and its output

### Requirement: Structural failures short-circuit to save time and cost
The system SHALL return the structural failure to the workflow immediately so that the retry loop can re-execute the implement agent with the structural failure context. The manage agent review (which requires starting an LLM session) SHALL NOT be invoked when the failure is detectable by automated checks alone.

#### Scenario: Structural failure saves LLM cost on obvious errors
- **WHEN** the implement agent produced no file changes (empty diff) and task completion shows 0/N tasks complete
- **THEN** the verify activity returns in under 30 seconds (no LLM call)
- **AND** the workflow retries the implement agent with the structural failure report

#### Scenario: Structural failure context is included in retry prompt
- **WHEN** a structural check fails and the workflow retries the implement agent
- **THEN** the retry prompt includes the specific structural failure (e.g., "test command `go test ./...` failed with: <output>")
- **AND** the implement agent receives actionable information about what to fix

### Requirement: Preserve existing gate ordering within the structural tier
The structural checks SHALL execute in the existing order: (1) task completion via `openspec list`, (2) spec validation via `openspec validate`, (3) file existence checks extracted from scenarios, (4) test commands extracted from specs. Each check SHALL short-circuit on failure (do not run subsequent checks if an earlier one fails).

#### Scenario: Checks run in order and short-circuit
- **WHEN** the spec validation check fails
- **THEN** file existence checks and test commands are NOT executed
- **AND** only the checks that ran are included in the `AutomatedChecks` array

#### Scenario: All four structural check types run when all pass
- **WHEN** task completion, spec validation, file existence, and test commands all pass
- **THEN** the `AutomatedChecks` array contains results for all four check types
- **AND** the total structural check phase completes before the manage agent review begins
