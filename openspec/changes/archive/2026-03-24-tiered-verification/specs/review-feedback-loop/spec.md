## ADDED Requirements

### Requirement: Manage agent review feedback replaces the generic retry template
The system SHALL use the manage agent's detailed review feedback as the retry prompt for the implement agent, replacing the current generic template ("PREVIOUS ATTEMPT FAILED: ... Fix the issues and complete the OpenSpec change"). The feedback SHALL contain specific, actionable comments tied to individual spec scenarios.

#### Scenario: Retry prompt contains manage agent review comments
- **WHEN** the manage agent review fails and the workflow retries the implement agent
- **THEN** the retry prompt includes the manage agent's per-scenario feedback (which scenarios failed, why they failed, what to fix)
- **AND** the retry prompt does NOT use the generic "Fix the issues" template

#### Scenario: Retry prompt includes both structural and review failures
- **WHEN** a previous attempt failed structural checks and a subsequent attempt fails manage agent review
- **THEN** the retry prompt for the next attempt includes only the most recent failure's feedback (the manage agent review comments)
- **AND** the structural check results from the passing tier are not re-reported as failures

#### Scenario: Manage agent feedback is actionable
- **WHEN** the manage agent identifies a failing scenario
- **THEN** the feedback for that scenario includes: the scenario name, what the spec requires (WHEN/THEN), what the implementation actually does (from the diff or file inspection), and a specific instruction for what to change
- **AND** the implement agent can act on the feedback without re-reading the full spec

### Requirement: Review feedback is stored on the VerificationResult
The `VerificationResult` struct SHALL include a `ReviewFeedback` field (string) containing the manage agent's full review commentary. This field is distinct from `FailureReport` — the failure report is a concise summary, while the review feedback is the detailed per-scenario analysis.

#### Scenario: ReviewFeedback is populated on manage agent review failure
- **WHEN** the manage agent review produces a failing verdict
- **THEN** `VerificationResult.ReviewFeedback` contains the manage agent's full per-scenario analysis
- **AND** `VerificationResult.FailureReport` contains a concise summary of which scenarios failed

#### Scenario: ReviewFeedback is populated on manage agent review success
- **WHEN** the manage agent review produces a passing verdict
- **THEN** `VerificationResult.ReviewFeedback` contains the manage agent's analysis (for audit/observability)
- **AND** the feedback is available in the verification result JSON written to disk

### Requirement: Workflow passes review feedback as the retry prompt
The workflow's execute-verify retry loop SHALL read `VerificationResult.ReviewFeedback` (when present) and use it as the `lastFailureReport` variable that constructs the implement agent's retry prompt. When `ReviewFeedback` is empty (structural failure), the workflow SHALL fall back to `FailureReport`.

#### Scenario: ReviewFeedback is preferred over FailureReport for retry prompt
- **WHEN** the verification fails with both `FailureReport` and `ReviewFeedback` populated
- **THEN** the workflow uses `ReviewFeedback` as the retry prompt content
- **AND** the implement agent receives the detailed per-scenario feedback

#### Scenario: FailureReport is used when ReviewFeedback is empty
- **WHEN** the verification fails with only `FailureReport` populated (structural failure, no manage agent review ran)
- **THEN** the workflow uses `FailureReport` as the retry prompt content
- **AND** the implement agent receives the structural failure details

### Requirement: Previous review feedback is included in subsequent reviews
On retries, the manage agent's review prompt SHALL include the previous attempt's review feedback. This gives the reviewer context about what was already flagged, so it can check whether the implement agent addressed the prior feedback.

#### Scenario: Second review sees first review's feedback
- **WHEN** the first verification attempt's manage agent review failed with specific feedback
- **AND** the second attempt's manage agent review starts
- **THEN** the second review prompt includes a section titled "Previous review feedback" with the first review's commentary
- **AND** the manage agent can assess whether the implement agent addressed the prior issues

#### Scenario: First attempt has no previous feedback
- **WHEN** the first verification attempt's manage agent review starts
- **THEN** the review prompt does NOT include a "Previous review feedback" section
- **AND** the manage agent reviews the implementation fresh
