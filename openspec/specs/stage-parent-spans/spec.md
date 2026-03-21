# stage-parent-spans Specification

## Purpose
TBD - created by archiving change hierarchical-trace-spans. Update Purpose after archive.
## Requirements
### Requirement: Stage parent spans created by Temporal workflow
The system SHALL create a parent trace span for each pipeline stage (PLAN, EXECUTE, VERIFY) with start/end times and a link to the root span.

#### Scenario: Plan stage parent span
- **WHEN** the spec-driven workflow starts the PlanRun activity
- **THEN** a span with name "PLAN", type "stage", and parentId set to the root span SHALL be written to spans.jsonl
- **AND** when PlanRun completes, the span's endTime SHALL be set

#### Scenario: Execute stage parent span with attempt number
- **WHEN** the workflow starts execute attempt 2
- **THEN** a span with name "EXECUTE", type "stage", and metadata `{"attempt": 2}` SHALL be written
- **AND** its parentId SHALL be the root pipeline span

#### Scenario: Verify stage parent span with result
- **WHEN** the workflow completes verification and it fails
- **THEN** the VERIFY span's status SHALL be "error" and metadata SHALL include `{"result": "failed"}`

### Requirement: Child spans linked to stage parent via parentId
The system SHALL set parentId on all sidecar-created spans to link them to their stage parent span.

#### Scenario: Sidecar spans have parentId
- **WHEN** the sidecar creates a manage.thought span during the plan stage
- **THEN** the span's parentId SHALL be the PLAN stage span's ID

#### Scenario: parentSpanId passed via StartAgent
- **WHEN** the Temporal workflow calls StartAgent for the execute stage
- **THEN** the StartAgentRequest SHALL include a parentSpanId field
- **AND** the sidecar SHALL use this as the parentId for all child spans created during that stage

### Requirement: Retry cycles create separate stage spans
The system SHALL create a new EXECUTE and VERIFY parent span for each retry attempt, with incrementing attempt numbers.

#### Scenario: Two execute attempts
- **WHEN** execute attempt 1 fails verification and is retried
- **THEN** two EXECUTE stage spans SHALL exist: one with attempt=1 and one with attempt=2
- **AND** each SHALL have its own child spans

### Requirement: Root pipeline span
The system SHALL create a single root span per run representing the entire pipeline, with aggregate metadata rolled up from all stages.

#### Scenario: Root span aggregates tokens
- **WHEN** the pipeline completes with PLAN using 2000 tokens and EXECUTE using 5000 tokens
- **THEN** the root span's metadata SHALL include `gen_ai.usage.total_tokens: 7000`

#### Scenario: Root span shows cost
- **WHEN** the pipeline uses deepseek-v3.1 with 10000 input tokens and 3000 output tokens
- **THEN** the root span's metadata SHALL include an estimated cost based on model pricing

### Requirement: Collapsible stage rows in waterfall
The system SHALL render stage parent spans as collapsible rows that show/hide their child spans.

#### Scenario: Collapse stage hides children
- **WHEN** the user clicks the collapse toggle on a PLAN stage row
- **THEN** all child spans of that stage SHALL be hidden
- **AND** the PLAN row SHALL continue showing its duration bar and aggregate stats

