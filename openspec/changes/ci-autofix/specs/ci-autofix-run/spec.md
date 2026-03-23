## ADDED Requirements

### Requirement: Create a fix AgentRun targeting the existing PR branch
The system SHALL create a new AgentRun CRD that checks out the existing `aot/*` branch (not the base branch). The AgentRun spec SHALL set the `Branch` field on the repository to the failing PR branch so the agent workspace starts with the code that failed CI.

#### Scenario: Fix run is created for a failing PR branch
- **WHEN** a CI failure is detected on branch `aot/ar-abc123`
- **THEN** a new AgentRun is created with `spec.repos[0].branch` set to `aot/ar-abc123`
- **AND** the run's `specSource` is set to `ci-autofix:{owner}/{repo}#{pr_number}`

#### Scenario: Fix run receives CI error context in the prompt
- **WHEN** the fix AgentRun is created
- **THEN** the `spec.prompt` contains the condensed CI error log
- **AND** the prompt instructs the agent to fix the specific CI failures and commit to the same branch

### Requirement: Fix run skips the planning stage
The fix run SHALL use a workflow variant that skips the Plan stage and proceeds directly to Execute with the CI error context as the prompt. The fix run does not generate new OpenSpec artifacts; it works on the existing code and fixes the specific errors reported by CI.

#### Scenario: Fix run workflow skips plan
- **WHEN** a fix run's Temporal workflow starts
- **THEN** the workflow does not execute the PlanRun activity
- **AND** proceeds directly to the Execute stage with the CI error prompt

#### Scenario: Fix run still runs verification
- **WHEN** the fix run's Execute stage completes
- **THEN** the workflow runs the Verify stage against the existing OpenSpec change
- **AND** uses the same verification logic as a normal spec-driven run

### Requirement: Fix run pushes to the existing branch without creating a new PR
After successful verification, the fix run SHALL push its changes to the same `aot/*` branch. It SHALL NOT create a new PR because the existing PR already tracks that branch and will update automatically when the branch is pushed.

#### Scenario: Fix run pushes to existing branch
- **WHEN** the fix run passes verification
- **THEN** the system calls PushChanges with the existing branch name (e.g., `aot/ar-abc123`)
- **AND** does NOT call CreatePR

#### Scenario: Push triggers CI re-run
- **WHEN** the fix run pushes to the `aot/*` branch
- **THEN** GitHub automatically re-runs CI checks on the updated branch
- **AND** if CI fails again, the webhook triggers another autofix cycle (subject to circuit breaker)

### Requirement: Fix run links to the parent PR
The fix AgentRun status SHALL include a `parentPRUrl` field linking to the PR being fixed. The `parentRunID` field SHALL reference the original AgentRun that created the PR (if known).

#### Scenario: Fix run status shows parent PR
- **WHEN** a fix AgentRun is created
- **THEN** `status.parentPRUrl` contains the URL of the PR being fixed
- **AND** `spec.parentRunID` contains the name of the original AgentRun (if resolvable from the branch name)
