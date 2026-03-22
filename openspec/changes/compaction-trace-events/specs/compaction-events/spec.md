## ADDED Requirements

### Requirement: Sidecar detects compaction events in pi's JSONL output
The sidecar's `maybeCaptureStreamEvent` function SHALL recognize pi's compaction event type and create a trace span when one is detected.

#### Scenario: Compaction event is detected in the JSONL stream
- **WHEN** pi emits a JSON line with a compaction event type (e.g. `"type": "context_compaction"` or equivalent)
- **THEN** the sidecar creates a `TraceSpan` with `Type: "compaction"` and appends it to the trace spans file

#### Scenario: Compaction event includes token count metadata
- **WHEN** a compaction event contains pre-compaction and post-compaction token counts
- **THEN** the resulting span's metadata includes `compaction.tokens_before`, `compaction.tokens_after`, and `compaction.tokens_saved`

#### Scenario: Non-compaction events are unaffected
- **WHEN** pi emits a standard event type (e.g. `message_start`, `tool_execution_start`)
- **THEN** the existing span creation logic handles it and no compaction span is created

### Requirement: Compaction spans have correct trace hierarchy
Each compaction span SHALL be linked to the current trace and stage via `traceId` and `parentId`, consistent with how other spans are parented.

#### Scenario: Compaction span is parented to current stage
- **WHEN** a compaction event occurs during the EXECUTE stage
- **THEN** the compaction span's `parentId` is the current stage span ID AND `traceId` matches the active trace

#### Scenario: Compaction span timing reflects instant event
- **WHEN** a compaction event is captured
- **THEN** the span's `startTime` and `endTime` are both set to the time the event was received (zero-duration span)

### Requirement: Compaction spans render distinctly in the trace waterfall
The TraceTimeline component SHALL render compaction spans with a unique visual style that distinguishes them from tool, thought, and LLM spans.

#### Scenario: Compaction span has distinct color
- **WHEN** a compaction span is rendered in the waterfall
- **THEN** it uses a unique color scheme (e.g. orange/amber warning tones) that is not used by any other span type

#### Scenario: Compaction span label shows token reduction
- **WHEN** a compaction span with token metadata is rendered
- **THEN** the label displays the token reduction (e.g. "Compaction: 48k -> 24k tokens")

#### Scenario: Compaction detail panel shows full metadata
- **WHEN** an engineer clicks on a compaction span
- **THEN** the detail panel shows: tokens before, tokens after, tokens saved, percentage reduced, and the stage in which compaction occurred

### Requirement: Compaction type is included in span type contracts
The `"compaction"` string SHALL appear in both the frontend TypeScript type union and the backend Go `validSpanTypes` map.

#### Scenario: Frontend type union includes compaction
- **WHEN** the `TraceSpan` interface in `agent-run.ts` is inspected
- **THEN** the `type` field's union includes `"compaction"`

#### Scenario: Contract test validates compaction type
- **WHEN** the `TestBoundary_SpanTypes_GatewayUsesValidTypes` contract test runs
- **THEN** `"compaction"` is present in `validSpanTypes` AND in `expectedGatewayTypes`
