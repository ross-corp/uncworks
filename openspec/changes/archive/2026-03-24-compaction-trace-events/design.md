## Context

The trace timeline surfaces every meaningful event in an agent run: LLM turns (`message_start`/`message_end`), tool executions (`tool_execution_start`/`tool_execution_end`), and thinking blocks (`thinking_delta`/`text_end`). But when pi's context window fills up, it compacts (summarizes) earlier conversation turns to free token space. This is a critical event -- it means the agent has lost access to earlier context -- yet it is invisible in the waterfall.

Pi emits all events as JSON lines on stdout. The sidecar's `maybeCaptureStdoutSpan` parses each line into a `piEvent` struct and routes it to `maybeCaptureStreamEvent`, which switches on `evt.Type`. The known event types today are: `tool_execution_start`, `tool_execution_end`, `message_start`, `message_update`, `message_end`. Compaction events need to be added as a new case.

## Goals / Non-Goals

**Goals:**
- Detect compaction events in pi's JSONL output stream.
- Record compaction spans with pre/post token counts as metadata.
- Render compaction spans in the waterfall with a distinct visual (different from tool/thought/LLM colors).
- Keep compaction spans correctly parented in the trace hierarchy.

**Non-Goals:**
- Preventing compaction (that is a prompt engineering concern, not an observability one).
- Replaying compacted content (the original messages are already gone).
- Changing pi's compaction behavior or thresholds.

## Decisions

### 1. Identify pi's compaction event format

Pi emits compaction events in its JSONL stream. The exact event type string needs to be confirmed by inspecting pi's source or running a session that triggers compaction. Likely candidates based on pi's event naming conventions:

- `"type": "context_compaction"` -- most explicit
- `"type": "compaction"` -- shorter variant
- A `message_update` with an inner `assistantMessageEvent` of type `"compaction"` or `"context_trimmed"`

The implementation task should run pi with a small context window model and observe the actual event. The `piEvent.Type` field or `piAssistantEvent.Type` field will contain the compaction indicator.

**Expected payload fields:** Pre-compaction token count, post-compaction token count, and possibly a summary of what was compacted.

### 2. Detection in `maybeCaptureStreamEvent`

Add a new case to the `switch evt.Type` block in `maybeCaptureStreamEvent` (`internal/sidecar/gateway.go`, line ~1234). The handler should:

1. Extract pre/post token counts from the event payload.
2. Create a zero-duration `TraceSpan` (startTime == endTime == now) since compaction is an instant event, not a range.
3. Set `Type: "compaction"`, `Name: "<role>.compaction"` following the existing `spanPrefix() + ".suffix"` convention.
4. Store metadata: `compaction.tokens_before`, `compaction.tokens_after`, `compaction.tokens_saved`, `compaction.reduction_pct`.
5. Call `appendTraceSpan(span)`.

The token extraction should be resilient -- if the payload does not contain token counts, create the span anyway with whatever metadata is available.

### 3. Span type: "compaction" (new type, not reusing "system")

A new `"compaction"` type is cleaner than overloading `"lifecycle"` or `"tool"`. Compaction is semantically different from all existing types:

- It is not an LLM turn (no model invocation).
- It is not a tool call (no tool execution).
- It is not a thought (no reasoning content).
- It is a system-level context management event.

This requires adding `"compaction"` to:
- `TraceSpan.Type` in `gateway.go` (Go struct -- string field, no enum, so just use it).
- `TraceSpan["type"]` union in `web/src/types/agent-run.ts`.
- `validSpanTypes` in `test/contract/boundary_span_types_test.go`.

### 4. Waterfall rendering: horizontal marker with warning color

Compaction spans should stand out visually because they represent a significant event (context loss). The design:

- **Color:** Orange/amber (`bg-orange-500/30 border-l-2 border-orange-500`, `text-orange-400`) -- warm warning tone, not used by any existing span type. Distinct from the amber used for `started` spans which uses `amber-500`.
- **Bar style:** Zero-duration spans already render as thin vertical lines. For compaction, use the same thin bar but with the orange color to make it a visual marker.
- **Label:** Show token reduction inline: `"Compaction: 48k -> 24k"` computed from metadata.
- **Detail panel:** On click, show full token counts, reduction percentage, and stage context.

Add entry to `OP_COLORS` in `TraceTimeline.tsx`:
```
compaction: { bar: "bg-orange-500/30 border-l-2 border-orange-500", text: "text-orange-400" }
```

Also handle in `resolveOperation` -- compaction spans have name `"<role>.compaction"`, so the `.pop()` will return `"compaction"` which maps to the new OP_COLORS entry.

## Risks / Trade-offs

- **[Pi event format unknown]** -- The exact compaction event type string has not been confirmed. The first task explicitly addresses this by running pi with a small context window. If the format differs from expectations, the detection code adjusts to match.
- **[Compaction may not always include token counts]** -- Some pi versions or configurations may emit compaction events without detailed token metadata. The span should be created regardless, with whatever metadata is available. Missing fields render as "--" in the detail panel.
- **[New span type adds to contract surface]** -- Adding `"compaction"` to the type union means the contract test must be updated in lockstep with frontend and backend. This is a one-time cost and is handled by an explicit task.
- **[Compaction frequency is model-dependent]** -- Small context window models (qwen3:8b, 8k context) may compact frequently, creating many spans. Large context models may never compact. The waterfall handles this naturally -- more spans means more visible markers, which is the desired behavior.
