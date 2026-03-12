## 1. Event Bus Core

- [x] 1.1 Create `internal/eventbus/eventbus.go` with `EventBus` interface (`Publish`, `Subscribe`, `Unsubscribe`) and channel-based implementation
- [x] 1.2 Write `internal/eventbus/eventbus_test.go` — single subscriber, multi-subscriber, cross-run isolation, slow-client drop, unsubscribe cleanup, empty topic removal
- [x] 1.3 Add `NoOpEventBus` for tests that don't need event delivery

## 2. Controller Integration

- [x] 2.1 Add `EventBus` field to `AgentRunReconciler` struct and inject it at construction in `cmd/controller/main.go`
- [x] 2.2 Call `bus.Publish()` after each status subresource update in `reconcilePod()` — phase change, TTL expiry
- [x] 2.3 Write tests verifying events are emitted on phase transitions (mock EventBus)

## 3. gRPC WatchAgentRun Streaming

- [x] 3.1 Add `EventBus` field to `GRPCServer` struct and inject at construction
- [x] 3.2 Rewrite `WatchAgentRun` to: send current state, subscribe to bus, stream events, unsubscribe on context done or run completion
- [x] 3.3 Write tests for WatchAgentRun — initial state delivery, event streaming, client disconnect cleanup, stream close on completion

## 4. WebSocket Hub Integration

- [x] 4.1 Refactor `Hub` to accept an `EventBus` and subscribe to it per active topic
- [x] 4.2 Auto-subscribe to bus when first WebSocket client subscribes to a run; auto-unsubscribe when last client leaves
- [x] 4.3 Write tests for WebSocket hub receiving events from bus and broadcasting to clients

## 5. TypeScript WebSocket Reconnection

- [x] 5.1 Add `ReconnectingStream` class to `packages/shared/src/ws/reconnecting-stream.ts` with exponential backoff, jitter, max retries
- [x] 5.2 Track active subscriptions and re-send on reconnect
- [x] 5.3 Emit `connection_failed` event after max retries exceeded
- [x] 5.4 Write tests for reconnection backoff timing, subscription restoration, and max retry cutoff

## 6. Integration Verification

- [ ] 6.1 Add E2E test: create AgentRun, watch via gRPC stream, update status, verify event delivery
- [ ] 6.2 Update web dashboard to use `ReconnectingWebSocket` instead of raw WebSocket
- [ ] 6.3 Manual smoke test: start cluster, create run, verify web dashboard updates in real time
