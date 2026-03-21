## 1. Proto + Types

- [x] 1.1 Add `parentSpanId` and `traceId` fields to `StartAgentRequest` in agent proto
- [x] 1.2 Add `TraceID` field to `TraceSpan` struct in sidecar (`gateway.go`) and server (`traces.go`)
- [x] 1.3 Add `Status` field ("ok", "error", "unset") to `TraceSpan` struct
- [x] 1.4 Add `TraceID` and `Status` to frontend `TraceSpan` type (`agent-run.ts`)

## 2. Workflow Stage Spans

- [x] 2.1 Create `WriteTraceSpan` Temporal activity that writes a span JSON line to `.aot/traces/spans.jsonl` via ExecCommand
- [x] 2.2 In `workflow_spec_driven.go`: generate traceID at pipeline start, create root "pipeline" span
- [x] 2.3 Before PlanRun: create PLAN stage span (type "stage", parentId = root span ID)
- [x] 2.4 After PlanRun: update PLAN span with endTime via closing span write
- [x] 2.5 Before each Execute attempt: create EXECUTE stage span with attempt number
- [x] 2.6 After each Execute: close EXECUTE span
- [x] 2.7 Before each Verify attempt: create VERIFY stage span with attempt number
- [x] 2.8 After each Verify: close VERIFY span with result status (ok/error)
- [x] 2.9 On pipeline completion: close root span with aggregate metadata
- [x] 2.10 Pass stage span ID as `parentSpanId` in StartAgentRequest

## 3. Sidecar Child Span Linking

- [x] 3.1 Read `parentSpanId` from StartAgentRequest, store as package-level state
- [x] 3.2 Set `parentId` on all child spans (thought, tool, started) to the current stage span ID
- [x] 3.3 Set `traceId` on all child spans from StartAgentRequest

## 4. Token Usage Extraction

- [x] 4.1 Parse `usage` from pi's `message_end` events in `maybeCaptureStreamEvent`
- [x] 4.2 Add token fields to thought span metadata: `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`, `gen_ai.usage.cache_read_tokens`
- [x] 4.3 Add model name to thought span metadata: `gen_ai.request.model`
- [x] 4.4 Compute context utilization percentage

## 5. Cost Estimation

- [x] 5.1 Create `internal/temporal/pricing.go` with per-model pricing table
- [x] 5.2 Compute cost in frontend from token counts using mirrored pricing
- [x] 5.3 Add `gen_ai.cost.stage_usd` computation (client-side aggregation)
- [x] 5.4 Add `gen_ai.cost.total_usd` computation (client-side aggregation)

## 6. Stage Span Aggregation

- [x] 6.1 Frontend aggregates child span metadata when rendering stage parent
- [x] 6.2 computeStageAggregates sums token counts and tool counts
- [x] 6.3 Root span aggregates all descendant spans

## 7. Frontend Hierarchy

- [x] 7.1 Add "stage" span type to styling (bold, taller bar, amber)
- [x] 7.2 Add collapse/expand toggle for stage parent rows
- [x] 7.3 Show aggregate stats in stage row label
- [x] 7.4 Detail panel for stage spans: tokens, cost, tools, attempt
- [x] 7.5 Detail panel for root span: pipeline summary
- [x] 7.6 Remove CSS stage separator hack

## 8. Tests

- [x] 8.1 Unit test: TraceSpanData JSON marshal/unmarshal
- [x] 8.2 Unit test: extractToolFromEvent with all formats
- [x] 8.3 Unit test: Token extraction from message_end (via extractToolFromEvent tests)
- [x] 8.4 Unit test: Cost estimation with pricing table
- [x] 8.5 Contract test: StartAgentInput with parentSpanId and traceId
- [x] 8.6 Fix: Add "stage" to validSpanTypes in contract test
