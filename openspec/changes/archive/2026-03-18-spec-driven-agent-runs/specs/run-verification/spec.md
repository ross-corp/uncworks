## Purpose

Define the hybrid verification system that uses OpenSpec's built-in CLI tooling as the primary evaluation engine, augmented with automated scenario checks and LLM judgment for semantic criteria.

## ADDED Requirements

### Requirement: Verification runs OpenSpec task completion check first
The verification stage SHALL check task completion via `openspec list --json` as the first gate. If tasks are incomplete, verification fails immediately without further checks.

#### Scenario: All tasks complete
- **WHEN** `openspec list --json` reports the change's `completedTasks` equals `totalTasks` and `status` is `"complete"`
- **THEN** the task completion check passes and verification proceeds to the next gate

#### Scenario: Incomplete tasks fail immediately
- **WHEN** `openspec list --json` reports `completedTasks` less than `totalTasks`
- **THEN** verification fails immediately without running automated checks or LLM judge
- **AND** the failure report lists the specific incomplete tasks from `tasks.md`

### Requirement: Verification validates spec structure via OpenSpec CLI
The verification stage SHALL run `openspec validate --json` to ensure the change's spec structure remains valid after execution.

#### Scenario: Valid spec structure
- **WHEN** `openspec validate --json` reports the change as `valid: true` with no issues
- **THEN** the structural validation check passes

#### Scenario: Invalid spec structure
- **WHEN** `openspec validate --json` reports `valid: false` or returns issues
- **THEN** verification fails with the validation issues included in the failure report

### Requirement: Verification runs automated checks from spec scenarios
The verification stage SHALL execute machine-checkable criteria derived from spec scenarios — commands, file existence, and pattern matches — after OpenSpec CLI checks pass.

#### Scenario: Command-based check succeeds
- **WHEN** a spec scenario references a command (e.g., "WHEN `npm test` is run THEN it exits with code 0")
- **AND** the command exits with code 0
- **THEN** that automated check passes

#### Scenario: Command-based check fails and short-circuits
- **WHEN** a spec scenario references a command
- **AND** the command exits with a non-zero exit code
- **THEN** verification fails without invoking the LLM judge
- **AND** the failure report includes the command, exit code, and stdout/stderr

#### Scenario: File existence check
- **WHEN** a spec scenario references a file path (e.g., "THEN `src/auth.ts` exists")
- **AND** the file does not exist in the workspace
- **THEN** verification fails with the missing file path in the failure report

### Requirement: LLM judge evaluates semantic criteria after automated checks pass
The LLM judge SHALL only be invoked when all prior gates pass (task completion, validation, automated checks). It evaluates WHEN/THEN scenarios that cannot be checked mechanically.

#### Scenario: All automated checks pass, LLM evaluates semantics
- **WHEN** task completion, validation, and automated checks all pass
- **AND** the spec contains semantic WHEN/THEN scenarios
- **THEN** the LLM judge is invoked with the spec, git diff, and structured agent log
- **AND** the LLM returns a per-scenario pass/fail verdict with explanation

#### Scenario: Semantic criterion fails
- **WHEN** the LLM judge evaluates a WHEN/THEN scenario
- **AND** the git diff does not demonstrate the required behavior
- **THEN** the judge reports that criterion as failed with a specific explanation of what's missing

### Requirement: Successful verification triggers OpenSpec archive
On full verification pass, the verification stage SHALL archive the change via `openspec archive --yes`, which merges delta specs into the main spec tree.

#### Scenario: Archive on success
- **WHEN** all verification gates pass (tasks complete, valid, automated pass, LLM pass)
- **THEN** `openspec archive --yes` is executed for the run's change
- **AND** the change is moved to the archive directory
- **AND** delta specs are merged into `openspec/specs/`

#### Scenario: Archive failure is non-fatal
- **WHEN** all verification gates pass but `openspec archive` fails (e.g., filesystem error)
- **THEN** the run is still marked as Succeeded
- **AND** a warning is logged about the archive failure

### Requirement: Verification produces structured results
The verification stage SHALL produce a structured JSON result persisted to the workspace and returned to the Temporal workflow.

#### Scenario: Verification result structure
- **WHEN** verification completes (pass or fail)
- **THEN** the result includes: overall verdict (pass/fail), task completion status (completed/total), validation result, list of automated check results (each with name, pass/fail, output), LLM judge verdict if invoked (per-scenario breakdown), and total execution time

#### Scenario: Verification result persisted to workspace
- **WHEN** verification completes
- **THEN** the result is written to the change directory as `verification-result.json`

### Requirement: Verification failure report enables targeted retry
The failure report SHALL be formatted to give the execution agent actionable guidance when prepended to its retry context.

#### Scenario: Failure report as retry context
- **WHEN** verification fails
- **THEN** the failure report includes: which specific gate failed (tasks/validation/automated/semantic), which criteria failed, what the agent's code is missing, and concrete suggestions for fixing each failure
