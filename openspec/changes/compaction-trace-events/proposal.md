## Why

When pi's context window fills up, it compacts (summarizes) earlier messages to free token space. These compaction events are emitted in pi's JSONL stream but are completely invisible in the current trace timeline. Engineers have no way to see where compactions happen, which makes it difficult to:

- Understand context loss -- when the agent "forgets" earlier instructions or file contents, it is often because a compaction discarded that context.
- Debug regressions -- a tool call that worked earlier may fail after compaction because the agent lost the earlier result.
- Optimize prompt sizes -- without visibility into compaction frequency and token counts, there is no feedback loop for reducing prompt bloat.

The trace waterfall already surfaces LLM turns, tool calls, and thinking blocks. Adding compaction events as a distinct span type fills the last major observability gap in the agent's inner loop.

## What Changes

- Install `@ssweens/pi-compaxxt` in the sidecar image for enhanced compaction (session context, LLM-judged important files).
- Add a `session_before_compact` hook to the AOT extension that emits rich compaction metadata to the JSONL log.
- Detect compaction events in pi's JSONL output stream (the sidecar's `maybeCaptureStreamEvent` function).
- Create a new "compaction" span type with pre/post token counts stored in metadata.
- Render compaction spans in the TraceTimeline waterfall with a distinct visual treatment (orange/amber color) so they stand out from regular tool/thought spans.
- Add "compaction" to the frontend's `TraceSpan` type union and the backend's `validSpanTypes` contract test.

## Capabilities

### New Capabilities
- `compaction-events`: Detection of pi context compaction events in the JSONL stream, creation of compaction trace spans with pre/post token metadata, and distinct waterfall rendering.
- `pi-compaxxt`: Enhanced compaction quality via pi-compaxxt extension with session context and LLM-judged important files.

### Modified Capabilities
- None

## Impact

- `docker/Dockerfile.sidecar` -- Install `@ssweens/pi-compaxxt` alongside pi-coding-agent.
- `extensions/aot-determinism.ts` -- Add `session_before_compact` hook to emit compaction metadata to JSONL log.
- `internal/sidecar/gateway.go` -- New case in `maybeCaptureStreamEvent` switch for compaction event type; new span with `Type: "compaction"`.
- `web/src/types/agent-run.ts` -- Add `"compaction"` to `TraceSpan["type"]` union.
- `web/src/components/TraceTimeline.tsx` -- Add compaction styling to `OP_COLORS`, token reduction labels, detail panel.
- `test/contract/boundary_span_types_test.go` -- Add `"compaction"` to `validSpanTypes` map and `expectedGatewayTypes` list.
