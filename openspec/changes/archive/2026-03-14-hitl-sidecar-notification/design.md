## Context

The HITL data flow should be:

```
Agent process (extension)                Sidecar (gateway.go)              Workflow
       │                                        │                            │
       │ waitForHumanInput("question")           │                            │
       │ → NotifyEvent(WAITING_FOR_INPUT)  ────▶ │                            │
       │                                        │ state = WAITING_FOR_INPUT   │
       │                                        │ ◀── GetAgentStatus ──────── │
       │                                        │ ──▶ state=WAITING ────────▶ │
       │                                        │                            │ phase=WaitingForInput
       │                                        │                            │
       │                                        │ ◀── SendInput(data) ─────── │ (from signal)
       │ ◀── stdin.Write(data) ─────────────────│                            │
       │ Promise resolves with input             │                            │
       │ → NotifyEvent(STARTED) ───────────────▶│                            │
       │                                        │ state = RUNNING             │
```

Currently broken at every arrow between agent process and sidecar.

## Goals / Non-Goals

**Goals:**
- Sidecar accepts NotifyEvent RPCs and transitions agent process state accordingly
- Extension calls NotifyEvent when entering/exiting WAITING_FOR_INPUT
- Extension reads stdin to receive forwarded human input, bridging stdin ↔ Promise
- Full HITL cycle works: agent waits → user sees WaitingForInput → user sends input → agent resumes

**Non-Goals:**
- Structured question/answer protocol (the question text is just logged, not displayed specially)
- Multiple concurrent HITL sessions within one agent run
- Timeout on waiting for input (workflow TTL handles this)

## Decisions

### 1. NotifyEvent sets state directly

The sidecar's NotifyEvent handler sets `proc.state` based on `event_type`:
- `EVENT_TYPE_WAITING_FOR_INPUT` → `AGENT_PROCESS_STATE_WAITING_FOR_INPUT`
- `EVENT_TYPE_STARTED` → `AGENT_PROCESS_STATE_RUNNING` (agent resumed)
- Other event types: no state change (just log the event)

**Why not use stdout parsing?** Fragile, requires parsing agent output format. NotifyEvent is the designed protocol — it exists in the proto for exactly this purpose.

### 2. Extension calls sidecar via gRPC (localhost)

The extension runs inside the agent container. The sidecar runs in the same pod. The extension calls `localhost:50052` (sidecar port) to send NotifyEvent. This requires adding a gRPC client to the extension.

**Why localhost?** Same pod, shared network namespace. The sidecar port (50052) is already exposed.

### 3. Stdin bridge using readline

The extension adds a stdin reader (Node.js `readline` on `process.stdin`) that:
- Listens for lines
- When `waitForHumanInput()` is active (Promise pending), resolves the Promise with the line
- When not waiting, ignores or buffers the input

This bridges `SendInput` (sidecar writes to agent stdin) → `waitForHumanInput()` (extension Promise).

### 4. Extension uses @connectrpc/connect-node for gRPC

The extension already has protobuf dependencies. Add `@connectrpc/connect-node` for making ConnectRPC calls to the sidecar. Use the generated TS client from `gen/ts/aot/agent/v1/`.

## Risks / Trade-offs

- **[Risk] Agent process may not use the extension** → If the agent process doesn't import pi-aot-extension, HITL won't work. Mitigation: document that the extension is required for HITL support.
- **[Risk] Stdin race condition** → If SendInput writes before waitForHumanInput is called, the line is lost. Mitigation: buffer stdin lines and check buffer when waitForHumanInput is called.
- **[Risk] gRPC connection from agent container** → Agent container needs network access to sidecar. This works because they share a pod network namespace (localhost).
