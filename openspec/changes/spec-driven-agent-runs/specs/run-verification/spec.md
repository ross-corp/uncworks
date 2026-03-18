## Purpose

Define the hybrid verification system that evaluates agent work against the spec's acceptance criteria using both automated checks and LLM judgment.

## ADDED Requirements

### Requirement: Automated checks run before LLM evaluation
The verification stage SHALL first execute all machine-checkable criteria from the spec scenarios. The LLM judge is only invoked if all automated checks pass.

#### Scenario: Automated check failure short-circuits
- **WHEN** a spec scenario specifies a command-based check (e.g., `npm test`)
- **AND** the command exits with a non-zero exit code
- **THEN** the verification reports failure immediately without invoking the LLM judge
- **AND** the failure report includes the command, exit code, and stdout/stderr

#### Scenario: All automated checks pass, LLM evaluates semantic criteria
- **WHEN** all command-based and file-existence checks pass
- **AND** the spec contains semantic criteria (WHEN/THEN scenarios without automated checks)
- **THEN** the LLM judge is invoked with the spec, git diff, and agent log
- **AND** the LLM returns a structured verdict (pass/fail) with explanation for each criterion

### Requirement: Verification checks task completion via OpenSpec CLI
The verification stage SHALL check that all tasks in `tasks.md` are marked complete using `openspec list --json`.

#### Scenario: All tasks complete
- **WHEN** `openspec list --json` reports the change's `completedTasks` equals `totalTasks`
- **THEN** the task completion check passes

#### Scenario: Incomplete tasks
- **WHEN** `openspec list --json` reports `completedTasks` less than `totalTasks`
- **THEN** the verification reports failure
- **AND** the failure report lists the incomplete tasks

### Requirement: Verification produces structured results
The verification stage SHALL produce a structured result containing the verdict, automated check results, LLM evaluation, and the overall pass/fail status.

#### Scenario: Verification result structure
- **WHEN** verification completes (pass or fail)
- **THEN** the result includes: overall verdict (pass/fail), list of automated check results (each with name, pass/fail, output), LLM judge verdict if invoked (pass/fail with per-criterion breakdown), and total execution time

#### Scenario: Verification result persisted to workspace
- **WHEN** verification completes
- **THEN** the result is written to `/workspace/.openspec/changes/<run-id>/verification-result.json`
- **AND** the result is returned to the Temporal workflow for status updates

### Requirement: LLM judge evaluates against WHEN/THEN scenarios
The LLM judge SHALL be given the spec's WHEN/THEN scenarios, the git diff of all changes, and the structured agent log, and SHALL return a per-scenario pass/fail verdict.

#### Scenario: Semantic criterion passes
- **WHEN** the LLM judge evaluates a scenario like "WHEN a request is made without auth THEN it returns 401"
- **AND** the git diff shows auth middleware was added to the route
- **THEN** the judge reports this criterion as passed with explanation

#### Scenario: Semantic criterion fails
- **WHEN** the LLM judge evaluates a scenario
- **AND** the git diff does not demonstrate the required behavior
- **THEN** the judge reports this criterion as failed with a specific explanation of what's missing

### Requirement: Verification failure report enables retry
The verification failure report SHALL be formatted such that prepending it to the execution agent's context gives the agent actionable guidance on what to fix.

#### Scenario: Failure report as retry context
- **WHEN** verification fails
- **THEN** the failure report includes: which specific criteria failed, what the agent's code is missing, and concrete suggestions for fixing each failure
- **AND** prepending this report to the Stage 2 agent prompt is sufficient context for a targeted fix attempt
