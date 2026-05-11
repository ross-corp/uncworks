## ADDED Requirements

### Requirement: OpenSpec change name passed from template to run
Agent runs created from templates SHALL carry an `openspecChange` field in their spec when the template has an associated openspec change. This field is passed to the verification activity so `openspec list` can check the correct change.

#### Scenario: Run created from template with openspec context
- **WHEN** an agent run is submitted from a template that specifies an openspec change name
- **THEN** the AgentRun CRD's spec SHALL include `openspecChange: <change-name>`
- **AND** the verification stage SHALL use this value when calling `openspec list --change <name> --json`

#### Scenario: Run without openspecChange skips task gate
- **WHEN** a run's spec has no `openspecChange` field (ad-hoc run)
- **THEN** the verification stage skips the `openspec list` task completion gate
- **AND** verification proceeds with the remaining gates (validate, automated checks, LLM judge)

### Requirement: LLM judge produces salvageability verdict
The LLM judge SHALL assess not only pass/fail but also whether a failed run is "salvageable" — i.e., can meaningfully continue with an additional implement stage rather than being discarded.

#### Scenario: Judge marks run as salvageable
- **WHEN** the LLM judge evaluates a failed run and determines the core work is partially complete and the gap is small and well-defined
- **THEN** the judge sets `salvageable: true` in the verification result
- **AND** includes a `salvageGuidance` string describing what additional work is needed
- **AND** sets a `confidenceScore` between 0 and 1 indicating confidence that retry will succeed

#### Scenario: Judge marks run as not salvageable
- **WHEN** the LLM judge evaluates a failed run and determines the implementation is fundamentally wrong, incomplete in a large way, or that the required changes were not made
- **THEN** the judge sets `salvageable: false` in the verification result
- **AND** the run is marked as Failed with no retry scheduled

### Requirement: Salvageable runs can trigger a retry implement stage
When a run is marked salvageable, the workflow SHALL schedule an additional implement stage if the retry count has not been exhausted, using the judge's salvage guidance as the new execute prompt prefix.

#### Scenario: Retry within policy
- **WHEN** a run is marked salvageable
- **AND** the run's `retryCount` is less than `maxRetries` (default: 2)
- **THEN** the workflow schedules an additional implement stage
- **AND** prepends the `salvageGuidance` to the agent's next execute prompt
- **AND** increments `retryCount` on the AgentRun CRD

#### Scenario: Retry limit exceeded
- **WHEN** a run is marked salvageable
- **AND** the run's `retryCount` has reached `maxRetries`
- **THEN** the run is marked as Failed (retry limit exceeded)
- **AND** no further implement stages are scheduled

#### Scenario: Retry uses prior run context
- **WHEN** a retry implement stage is scheduled
- **THEN** the agent receives the judge's salvage guidance, the original task description, and a summary of what was done in prior attempts (from the run's logOutput)
- **AND** the agent does NOT start from scratch (existing code changes are preserved)

### Requirement: LLM judge is visible as a named stage in traces and logs
The LLM judge's execution SHALL appear as a named span in the trace (not "unknown stage") and its output SHALL be written to the run's logs.

#### Scenario: Judge appears as named trace span
- **WHEN** the LLM judge is invoked during verification
- **THEN** a trace span with name `verification.llm-judge` is recorded with start time, end time, and the judge's verdict in metadata

#### Scenario: Judge output written to run logs
- **WHEN** the LLM judge produces a verdict
- **THEN** the per-criterion results (each WHEN/THEN scenario with pass/fail/explanation) are written to the run's log output
- **AND** the overall verdict is written as the final log line before the run status transitions
