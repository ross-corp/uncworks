## Why

When watching a spec-driven run, the activity feed shows nothing while the agent is thinking. The JSONL file has streaming `message_update` events with partial text (the agent's thinking), but the structured logs endpoint only shows completed messages. Users stare at "Agent started" for minutes with no visibility into what the agent is doing. The activity feed should show real-time thinking — the current partial message that will be replaced when the next completed message arrives.

## What Changes

- **New endpoint** `GET /api/v1/runs/{id}/logs/thinking` — returns the last partial agent message from the JSONL file (the most recent `message_update` with `text_delta` events accumulated)
- **ActivityFeed polls thinking** — every 2 seconds, fetches the thinking endpoint and displays it as a dimmed "thinking..." entry at the bottom of the feed, replaced when a completed message arrives
- **Thinking entry type** — new entry type `thinking` in the activity feed with italic/dimmed styling and a pulsing indicator

## Capabilities

### New Capabilities
- `ui-agent-thinking`: Real-time display of the agent's in-progress thinking in the activity feed.

### Modified Capabilities

None.

## Impact

- `internal/server/files.go` — New `handleThinking` endpoint that reads the JSONL file and extracts the latest partial text
- `web/src/components/ActivityFeed.tsx` — Poll thinking endpoint, render thinking entry
