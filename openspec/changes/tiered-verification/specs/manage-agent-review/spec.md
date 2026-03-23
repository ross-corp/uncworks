## ADDED Requirements

### Requirement: Replace the single-prompt LLM judge with a manage agent review session
The system SHALL replace the current Gate 4 LLM judge (a single prompt that outputs a JSON verdict) with a full manage agent session during the verify stage. The manage agent SHALL be started via `StartAgent` with `stage: "verify"` and `role: "manage"`, using the manage model tier. The manage agent SHALL have tool access to read files, run commands, and inspect the workspace — it is a real agent session, not a one-shot prompt.

#### Scenario: Manage agent review session is started after structural checks pass
- **WHEN** all structural pre-screening checks (task completion, spec validation, file existence, test commands) pass
- **THEN** the system starts a manage agent session with `stage: "verify"` and the manage model tier
- **AND** the session has full tool access (read files, list directories, run commands in the workspace)

#### Scenario: Manage agent review uses the manage model, not the implement model
- **WHEN** the manage agent review session is started
- **THEN** the `Model` field on the `StartAgentRequest` is set to the pipeline's `manageModelTier` configuration value
- **AND** this model is independent of the model used for the implement agent's execute stage

#### Scenario: Manage agent receives a structured review prompt
- **WHEN** the manage agent review session is started
- **THEN** the prompt includes: (1) the git diff summary for the change, (2) the full text of all spec scenarios for the change, (3) excerpts from the implement agent's output log showing its reasoning and any questions, and (4) on retries, the previous review's feedback
- **AND** the prompt instructs the agent to evaluate each scenario, check the diff against requirements, and produce a structured verdict

#### Scenario: Manage agent produces a per-scenario verdict
- **WHEN** the manage agent review session completes
- **THEN** its output contains a JSON verdict with `pass` (boolean) and `criteria` (array of `{scenario, pass, explanation}` objects)
- **AND** each scenario from the spec is represented in the criteria array

#### Scenario: Manage agent verdict determines verification pass/fail
- **WHEN** the manage agent's verdict has `pass: true`
- **THEN** the verification result is marked as passed
- **AND** the `LLMVerdict` field on `VerificationResult` is populated from the manage agent's output

#### Scenario: Manage agent verdict failure populates the failure report
- **WHEN** the manage agent's verdict has `pass: false`
- **THEN** the verification result is marked as failed
- **AND** the `FailureReport` contains the manage agent's per-scenario explanations for failed criteria

### Requirement: Manage agent can investigate unclear areas during review
The manage agent SHALL have the ability to read source files, run test commands, and examine the workspace during its review session. It is not limited to reading the diff — it can investigate any file in the workspace to determine whether the implementation meets the spec.

#### Scenario: Manage agent reads source files beyond the diff
- **WHEN** the manage agent is reviewing a change and needs to understand context not shown in the diff
- **THEN** it can use file read tools to inspect any file under `/workspace`
- **AND** its review can reference findings from files that were not modified

#### Scenario: Manage agent runs verification commands
- **WHEN** the manage agent wants to verify a behavioral requirement (e.g., an API returns the correct response)
- **THEN** it can use command execution tools to run commands in the workspace
- **AND** command output informs its verdict

### Requirement: Manage agent session is polled to completion
The system SHALL poll the manage agent session until it reaches a terminal state (done or error), using the same `pollUntilAgentDone` mechanism used for other agent sessions. The verify activity SHALL record heartbeats during polling to prevent Temporal timeout.

#### Scenario: Manage agent session is polled with heartbeats
- **WHEN** the manage agent review session is running
- **THEN** the verify activity polls the sidecar for agent status at regular intervals
- **AND** records a Temporal heartbeat on each poll iteration

#### Scenario: Manage agent session times out
- **WHEN** the manage agent session does not complete within the verify stage timeout
- **THEN** the verify activity returns a failure result with a timeout error in the failure report
- **AND** the workflow treats this as a verification failure (eligible for retry)
