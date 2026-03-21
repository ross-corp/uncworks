# trace-regression Specification

## Purpose

Regression test specification for the trace/observability system. Codifies invariants that have broken in the past so they are automatically verified on every change.

## ADDED Requirements

### Requirement: Child spans MUST link to their stage parent via parentId

The system SHALL set `parentId` on every child span (thought, tool, input) to point to the current stage parent span ID.

#### Scenario: Thought span links to PLAN stage parent
- **WHEN** a `manage.thought` span is emitted during the PLAN stage
- **THEN** the span's `parentId` field SHALL equal the PLAN stage span's `id`
- **AND** the span's `traceId` field SHALL be non-empty and match the stage span's `traceId`

#### Scenario: Tool span links to EXECUTE stage parent
- **WHEN** an `implement.bash` span is emitted during the EXECUTE stage
- **THEN** the span's `parentId` field SHALL equal the EXECUTE stage span's `id`

### Requirement: Token usage MUST use gen_ai semantic convention field names

The system SHALL store token counts using `gen_ai.usage.input_tokens` and `gen_ai.usage.output_tokens` in thought span metadata, matching the OTel GenAI semantic conventions that pi emits.

#### Scenario: Thought span contains non-zero token counts
- **WHEN** a `*.thought` span is recorded after a pi `message_end` event with usage data
- **THEN** `metadata["gen_ai.usage.input_tokens"]` SHALL be a number greater than zero
- **AND** `metadata["gen_ai.usage.output_tokens"]` SHALL be a number greater than zero

#### Scenario: Token fields use correct names
- **WHEN** the thought span metadata is serialized to JSON
- **THEN** the keys SHALL be `gen_ai.usage.input_tokens` and `gen_ai.usage.output_tokens`
- **AND** the keys SHALL NOT be `input_tokens`, `output_tokens`, `inputTokens`, or `outputTokens`

### Requirement: Span names MUST include the actual tool kind, not a generic label

The system SHALL name tool spans as `{role}.{toolName}` (e.g., `manage.write`, `implement.bash`) rather than a generic `{role}.tool`.

#### Scenario: Write tool produces manage.write span name
- **WHEN** the agent invokes the `write` tool during the PLAN stage
- **THEN** the emitted span's `name` SHALL be `manage.write`
- **AND** the span's `name` SHALL NOT be `manage.tool`

#### Scenario: Bash tool produces implement.bash span name
- **WHEN** the agent invokes the `bash` tool during the EXECUTE stage
- **THEN** the emitted span's `name` SHALL be `implement.bash`
- **AND** the span's `name` SHALL NOT be `implement.tool`

### Requirement: Tool input MUST be captured in span metadata

The system SHALL store the tool's JSON arguments in `metadata.toolInput` for every tool span.

#### Scenario: Bash tool input captured
- **WHEN** a `toolcall_start` event is received with `{"name":"bash","arguments":{"command":"npm test"}}`
- **THEN** the resulting tool span's `metadata["toolInput"]` SHALL contain a JSON string with `"command":"npm test"`

#### Scenario: Write tool input captured
- **WHEN** a tool event is received for the `write` tool with a file path argument
- **THEN** the resulting tool span's `metadata["toolInput"]` SHALL contain a JSON string with the `path` field

### Requirement: Spans with hasDiff=true MUST return non-empty files from the diff endpoint

The system SHALL return at least one `FileDiff` entry with a non-empty `path` and `patch` when the `/api/v1/runs/{id}/traces/{spanId}/diff` endpoint is called for a span with `hasDiff=true`.

#### Scenario: Diff endpoint returns files for hasDiff span
- **WHEN** a GET request is made to `/api/v1/runs/{id}/traces/{spanId}/diff` for a span with `hasDiff=true`
- **THEN** the response status SHALL be 200
- **AND** the response body SHALL contain a `files` array with at least one entry
- **AND** each file entry SHALL have a non-empty `path` and `patch`

#### Scenario: Diff endpoint returns empty for non-diff span
- **WHEN** a GET request is made to `/api/v1/runs/{id}/traces/{spanId}/diff` for a span with `hasDiff=false`
- **THEN** the response status SHALL be 200
- **AND** the `files` array SHALL be empty

### Requirement: Stage parent spans MUST exist for PLAN, EXECUTE, and VERIFY

The system SHALL emit stage parent spans with `type="stage"` for each pipeline stage, and these spans MUST appear in the trace span list.

#### Scenario: PLAN stage span exists
- **WHEN** the trace spans are listed for a spec-driven pipeline run
- **THEN** at least one span SHALL have `name` containing "PLAN" and `type` equal to `"stage"`

#### Scenario: EXECUTE stage span exists
- **WHEN** the trace spans are listed for a spec-driven pipeline run
- **THEN** at least one span SHALL have `name` containing "EXECUTE" and `type` equal to `"stage"`

#### Scenario: VERIFY stage span exists
- **WHEN** the trace spans are listed for a spec-driven pipeline run
- **THEN** at least one span SHALL have `name` containing "VERIFY" and `type` equal to `"stage"`

### Requirement: Clicking a stage row MUST toggle children visibility in the waterfall

The system SHALL collapse and expand child spans when the user clicks the toggle on a stage parent row.

#### Scenario: Collapse stage hides children
- **WHEN** the user clicks the collapse toggle on an expanded PLAN stage row
- **THEN** all child spans (thought, tool) nested under PLAN SHALL be hidden
- **AND** the stage row itself SHALL remain visible with aggregate stats

#### Scenario: Expand stage shows children
- **WHEN** the user clicks the expand toggle on a collapsed EXECUTE stage row
- **THEN** all child spans nested under EXECUTE SHALL become visible

### Requirement: TraceSpan JSON MUST include traceId and status fields

The system SHALL serialize `traceId` and `status` in the TraceSpan JSON when they have non-empty values, and omit them when empty (via `omitempty`).

#### Scenario: TraceSpan with traceId and status serializes correctly
- **WHEN** a TraceSpan with `traceId="abc"` and `status="ok"` is marshaled to JSON
- **THEN** the JSON output SHALL contain `"traceId":"abc"` and `"status":"ok"`

#### Scenario: TraceSpan with empty traceId omits the field
- **WHEN** a TraceSpan with an empty `traceId` is marshaled to JSON
- **THEN** the JSON output SHALL NOT contain the key `"traceId"`
