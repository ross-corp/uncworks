## Why

The `WatchAgentRun` gRPC RPC is currently stubbed — it sends the initial state and immediately closes the stream. The WebSocket hub exists but is disconnected from the controller's reconciliation loop. Users cannot see real-time agent progress without polling. This blocks the core UX promise: watching an AI agent work live.

## What Changes

- Introduce a shared in-process event bus that the controller publishes to on every `AgentRun` status change (phase transition, log line, tool call).
- Rewrite `WatchAgentRun` to subscribe to the event bus and stream updates to the gRPC client until the run completes or the client disconnects.
- Wire the WebSocket hub to consume from the same event bus, so web dashboard clients get identical real-time updates.
- Add reconnection and backoff logic to the `@aot/shared` WebSocket client so the SolidJS dashboard auto-recovers from disconnects.
- Add an `EmitEvent` method on the controller reconciler that publishes structured `AgentRunEvent` messages after each status update.

## Capabilities

### New Capabilities
- `event-bus`: In-process pub/sub event bus with per-run topic channels, fan-out to multiple subscribers, and automatic cleanup on run completion.
- `ws-reconnect`: WebSocket client reconnection with exponential backoff, jitter, and max-retry limits for the TypeScript `@aot/shared` package.

### Modified Capabilities

## Impact

- `internal/server/grpc.go` — `WatchAgentRun` rewritten from stub to event-bus subscriber.
- `internal/server/websocket.go` — Hub subscribes to event bus instead of requiring manual `Broadcast()` calls.
- `internal/controller/agentrun_controller.go` — Reconciler emits events after status updates.
- `packages/shared/src/grpc/client.ts` — `watchAgentRun()` already returns a stream; no API change needed.
- `packages/shared/src/store/agent-store.ts` — May need to handle reconnection state.
- `proto/api.proto` — No changes needed; `AgentRunEvent` message and `WatchAgentRun` RPC already defined.
