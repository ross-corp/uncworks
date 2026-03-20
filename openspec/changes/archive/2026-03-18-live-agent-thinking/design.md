## Context

The pi-coding-agent writes streaming JSONL events to `.aot/logs/agent.jsonl`. The `message_update` events contain `text_delta` fields with partial text as the LLM generates tokens. The structured logs endpoint only parses `message_end` (completed messages), so in-progress thinking is invisible.

## Goals / Non-Goals

**Goals:**
- Show the agent's current partial text in the activity feed in real-time
- Thinking entry is visually distinct (dimmed, italic, pulsing dot)
- Thinking entry is replaced when the full message arrives
- Minimal server load (read last N lines of JSONL, not full file)

**Non-Goals:**
- Token-by-token streaming via WebSocket (too complex, polling is sufficient)
- Showing thinking for completed runs (only relevant during active runs)

## Decisions

### Decision 1: Tail the JSONL file for latest text_delta

The `/logs/thinking` endpoint reads the last 100 lines of `agent.jsonl`, accumulates `text_delta` events from the most recent `message_start` that hasn't ended yet, and returns the accumulated text.

**Rationale:** Reading the last 100 lines is fast (~1ms). The alternative (streaming via SSE) adds complexity we don't need — 2-second polling is responsive enough.

### Decision 2: Separate endpoint, not part of structured logs

The thinking state is ephemeral — it's replaced by the completed message. Mixing it into the structured logs would require tracking which entries are "thinking" vs "final". A separate endpoint is cleaner.

### Decision 3: ActivityFeed polls thinking only for active runs

The ActivityFeed checks `run.status.phase === "running"` before polling thinking. For completed runs, no thinking poll happens.
