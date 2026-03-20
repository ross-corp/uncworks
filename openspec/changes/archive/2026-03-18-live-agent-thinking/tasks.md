## 1. Backend: Thinking Endpoint

- [x] 1.1 Add `GET /api/v1/runs/{id}/logs/thinking` handler in files.go
- [x] 1.2 Read last 100 lines of `.aot/logs/agent.jsonl` from workspace PVC
- [x] 1.3 Parse JSONL backwards to find the last `message_start` without a corresponding `message_end`
- [x] 1.4 Accumulate `text_delta` events from `message_update` entries after that `message_start`
- [x] 1.5 Return JSON: `{"thinking": true, "text": "accumulated partial text", "toolName": "if tool call in progress"}`
- [x] 1.6 If no in-progress message, return `{"thinking": false}`

## 2. Frontend: ActivityFeed Thinking Display

- [x] 2.1 Add `thinking` entry type to ActivityFeed with dimmed italic styling and pulsing dot
- [x] 2.2 Poll `/logs/thinking` every 2 seconds when run phase is "running"
- [x] 2.3 Show thinking entry at bottom of feed (after all completed entries)
- [x] 2.4 Replace thinking entry when a new completed entry arrives from structured logs
- [x] 2.5 Stop polling when run phase changes to succeeded/failed/cancelled

## 3. Testing

- [x] 3.1 Unit test: thinking parser handles in-progress message, completed message, empty file
- [ ] 3.2 Playwright test: thinking indicator visible during active run
