## ADDED Requirements

### Requirement: Track fix attempts per PR branch
The system SHALL maintain a counter of fix attempts per `aot/*` branch. Each time a CI autofix run is triggered for a branch, the counter for that branch SHALL increment by one. The counter SHALL be stored as an annotation on the most recent AgentRun CRD for that branch or as a field on the AgentRun status.

#### Scenario: First fix attempt for a branch
- **WHEN** a CI failure is detected on `aot/ar-abc123` and no prior fix attempts exist
- **THEN** the fix attempt counter for that branch is set to 1
- **AND** the autofix run is created

#### Scenario: Subsequent fix attempt increments counter
- **WHEN** a CI failure is detected on `aot/ar-abc123` and the counter is currently 2
- **THEN** the counter increments to 3
- **AND** the autofix run is created (if under the max)

### Requirement: Circuit breaker stops after max retries
The system SHALL enforce a maximum number of fix attempts per branch (default: 3, configurable via `CI_AUTOFIX_MAX_RETRIES` environment variable). When the counter reaches the maximum, no further autofix runs SHALL be created for that branch.

#### Scenario: Max retries reached
- **WHEN** a CI failure is detected on `aot/ar-abc123` and the fix attempt counter equals the max (3)
- **THEN** no new autofix run is created
- **AND** the system posts a comment on the PR explaining that autofix has been exhausted

#### Scenario: Max retries not yet reached
- **WHEN** a CI failure is detected and the fix attempt counter is below the max
- **THEN** a new autofix run is created normally

### Requirement: Post PR comment on circuit breaker activation
When the circuit breaker activates (max retries reached), the system SHALL post a comment on the associated GitHub PR. The comment SHALL summarize how many fix attempts were made and that manual intervention is required.

#### Scenario: Circuit breaker comment content
- **WHEN** the circuit breaker activates after 3 failed fix attempts
- **THEN** a comment is posted on the PR containing:
  - The number of fix attempts made
  - A note that automatic fixing has been stopped
  - A suggestion to review the CI logs manually
- **AND** the comment is posted via the GitHub REST API using the existing token provider

#### Scenario: PR comment API failure
- **WHEN** the GitHub API returns an error when posting the circuit breaker comment
- **THEN** the error is logged at warning level
- **AND** the circuit breaker still prevents further autofix runs (the comment is best-effort)

### Requirement: CRD status fields for CI fix tracking
The AgentRun status SHALL include `ciFixAttempts` (int32) tracking how many CI fix runs have been created for the associated PR, and `lastCIStatus` (string) recording the most recent CI check conclusion ("success", "failure", "pending"). These fields SHALL be updated by the webhook handler when processing check_run events.

#### Scenario: Status fields updated on fix run creation
- **WHEN** a new CI autofix run is created
- **THEN** the new run's `status.ciFixAttempts` reflects the current attempt number
- **AND** `status.lastCIStatus` is set to "failure"

#### Scenario: Status fields updated on CI success
- **WHEN** a `check_run` event arrives with `conclusion` = `success` for an `aot/*` branch
- **THEN** the most recent AgentRun for that branch has its `status.lastCIStatus` updated to "success"
