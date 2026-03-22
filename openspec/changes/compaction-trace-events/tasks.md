## 1. Install pi-compaxxt in Sidecar Image

- [x] 1.1 Add `@ssweens/pi-compaxxt` to `docker/Dockerfile.sidecar` via `pi install`
- [x] 1.2 Add `session_before_compact` hook to `extensions/aot-determinism.ts` that logs compaction metadata (tokensBefore, tokensAfter, summary length) to the JSONL log

## 2. Add Compaction Detection in `maybeCaptureStreamEvent` (DONE)

- [x] 2.1 Add `context_compaction` and `compaction` cases to the `switch evt.Type` block in `maybeCaptureStreamEvent` (`internal/sidecar/gateway.go`)
- [x] 2.2 Extract pre/post token counts from the event payload; handle missing fields gracefully
- [x] 2.3 Create a zero-duration `TraceSpan` with `Type: "compaction"`, `Name: spanPrefix() + ".compaction"`
- [x] 2.4 Populate metadata: `compaction.tokens_before`, `compaction.tokens_after`, `compaction.tokens_saved`, `compaction.reduction_pct`, `stage`
- [x] 2.5 Set `traceId` and `parentId` using `getTraceID()` and `getParentSpanID()`
- [x] 2.6 Call `appendTraceSpan(span)` to write the span to `spans.jsonl`

## 3. Add Compaction Styling to TraceTimeline (DONE)

- [x] 3.1 Add `"compaction"` to `TraceSpan["type"]` union in `web/src/types/agent-run.ts`
- [x] 3.2 Add `compaction` entry to `OP_COLORS` in `web/src/components/TraceTimeline.tsx`: orange/amber color scheme
- [x] 3.3 Add compaction label formatting: display `"Compaction: Xk -> Yk"` from metadata token counts
- [x] 3.4 Add compaction metadata rendering in the span detail panel (tokens before, after, saved, reduction percentage)

## 4. Add Compaction to Span Type Contract Test (DONE)

- [x] 4.1 Add `"compaction": true` to `validSpanTypes` map in `test/contract/boundary_span_types_test.go`
- [x] 4.2 Add `"compaction"` to `expectedGatewayTypes` slice in `TestBoundary_SpanTypes_GatewayUsesValidTypes`

## 5. Verification

- [ ] 5.1 Build updated sidecar image with pi-compaxxt and verify it starts correctly
- [ ] 5.2 Run a test with a small context model that triggers compaction and verify compaction spans appear in the trace
