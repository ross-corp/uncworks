## Context

AOT's control plane has three real-time delivery surfaces: gRPC `WatchAgentRun` server-streaming, WebSocket hub for browser clients, and `kubectl get -w` via the K8s watch API. The gRPC stream is currently stubbed (returns initial state, then closes). The WebSocket hub has broadcast infrastructure but nothing publishes to it. The controller reconciles AgentRun CRDs but does not emit events when status changes. These three systems need a shared event source.

## Goals / Non-Goals

**Goals:**
- Single source of truth for AgentRun events that feeds both gRPC streaming and WebSocket delivery.
- Zero-polling real-time updates for all clients.
- Clean subscriber lifecycle — no goroutine leaks, no blocked channels.
- WebSocket clients auto-reconnect on network interruptions.

**Non-Goals:**
- Durable event storage or replay (events are ephemeral, in-process only).
- Cross-process event bus (no NATS, Redis Streams, etc. — single binary for now).
- Push notifications or webhooks to external systems.
- Changes to the protobuf schema (existing `AgentRunEvent` message is sufficient).

## Decisions

### 1. In-process channel-based event bus

Use a Go struct with `sync.RWMutex`-protected subscriber map. Each subscriber gets a buffered channel. The bus fans out events to all subscribers for a given run ID.

**Alternative considered:** Using K8s watch API directly from gRPC server. Rejected because it couples the API server to a K8s client and doesn't cover the WebSocket path.

**Alternative considered:** Redis Pub/Sub. Rejected as premature — adds an external dependency for a single-process system.

### 2. Per-run topic channels with buffered subscribers

Subscribers register for a specific `agentRunID`. The bus maintains a `map[string][]chan *AgentRunEvent`. Channels are buffered (capacity 64). On publish, if a subscriber's channel is full, the event is dropped (non-blocking send) — this prevents a slow client from blocking the controller.

**Alternative considered:** Unbuffered channels with goroutine-per-send. Rejected because goroutine fan-out is harder to manage and can leak.

### 3. Controller emits events via injected EventBus interface

The `AgentRunReconciler` receives an `EventBus` interface at construction. After each status subresource update, it calls `bus.Publish(runID, event)`. This keeps the controller testable — tests inject a mock or no-op bus.

### 4. Exponential backoff with jitter for WebSocket reconnection

The TypeScript client uses `min(baseDelay * 2^attempt + jitter, maxDelay)` with base=1s, max=30s, jitter=0-1s. Reconnection resets on successful message receipt.

**Alternative considered:** Linear backoff. Rejected because exponential is standard practice and prevents thundering herd on server restart.

## Risks / Trade-offs

- **[Dropped events on slow clients]** → Acceptable for dashboard use. Clients see latest state on next event. Could add sequence numbers later if replay is needed.
- **[Memory from idle subscribers]** → Subscribers are cleaned up when the gRPC stream ends or WebSocket disconnects. The bus removes empty topic entries.
- **[Controller becomes event publisher]** → Tight coupling risk. Mitigated by using an interface — the bus is injected, not imported.
- **[No persistence]** → If the API server restarts, streaming clients lose history. Acceptable for v1; clients re-fetch current state on reconnect.
