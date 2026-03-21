## ADDED Requirements

### Requirement: Trace ID shared across all spans in a run
The system SHALL generate a single traceId per pipeline run and set it on all spans (root, stage parents, and child spans).

#### Scenario: All spans share traceId
- **WHEN** a pipeline run creates root, PLAN, EXECUTE, VERIFY spans and 50 child spans
- **THEN** all 53+ spans SHALL have the same traceId value

### Requirement: Root span metadata shows pipeline summary
The system SHALL include aggregate pipeline metadata on the root span that summarizes the entire run.

#### Scenario: Root span summary
- **WHEN** the pipeline completes with 3 stages, 2 attempts, 50 tool calls (48 success, 2 error), 16400 total tokens
- **THEN** the root span metadata SHALL include:
  - `pipeline.stages: 3`
  - `pipeline.attempts: 2`
  - `pipeline.result: "succeeded"`
  - `gen_ai.usage.total_tokens: 16400`
  - `tool.count.total: 50`
  - `tool.count.success: 48`
  - `tool.count.error: 2`

### Requirement: Detail panel shows aggregate stats for parent spans
The system SHALL render a summary view in the trace detail panel when a stage parent or root span is selected, showing aggregate metrics instead of individual span details.

#### Scenario: Stage detail shows token breakdown
- **WHEN** the user clicks the EXECUTE stage parent span in the waterfall
- **THEN** the detail panel SHALL show: total duration, token usage (input/output/cache), estimated cost, tool count (success/error), task completion status, attempt number
