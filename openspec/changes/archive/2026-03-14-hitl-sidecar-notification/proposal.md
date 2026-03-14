## Why

The HITL (Human-in-the-Loop) pipeline is completely non-functional. The workflow polls GetAgentStatus expecting `WAITING_FOR_INPUT` state, but the sidecar never reports it. The proto defines `AgentNotificationService.NotifyEvent()` with `EVENT_TYPE_WAITING_FOR_INPUT` but the handler returns Unimplemented. The `pi-aot-extension` sets an in-memory flag but never notifies the sidecar, and stdin writes from `SendInput` are disconnected from the extension's Promise-based input mechanism. Without this fix, the entire human-in-the-loop feature — a core differentiator — cannot work.

## What Changes

- **Sidecar NotifyEvent handler**: Implement the `NotifyEvent` RPC in `gateway.go`. When it receives `EVENT_TYPE_WAITING_FOR_INPUT`, set `proc.state = WAITING_FOR_INPUT`. When it receives events indicating the agent resumed (e.g., after input), transition back to RUNNING.
- **Extension stdin bridge**: In `pi-aot-extension/src/extension.ts`, add a stdin reader that listens for lines and resolves the `waitForHumanInput()` Promise when input arrives. This bridges the sidecar's `SendInput` (which writes to stdin) with the extension's Promise-based API.
- **Extension NotifyEvent calls**: When `waitForHumanInput()` is called, use gRPC to call the sidecar's `NotifyEvent` with `EVENT_TYPE_WAITING_FOR_INPUT`. When input is received and the Promise resolves, call `NotifyEvent` with `EVENT_TYPE_STARTED` to signal resumption.
- **Update contract tests**: Change `TestContract_NotifyEvent_Unimplemented` to verify the handler now works correctly.

## Capabilities

### New Capabilities
- `sidecar-notify-event`: Sidecar accepts NotifyEvent RPCs from the agent process to signal state transitions (waiting for input, resumed)
- `extension-stdin-bridge`: The pi-aot-extension bridges stdin pipe to Promise-based waitForHumanInput, and signals state to the sidecar via NotifyEvent

### Modified Capabilities

## Impact

- `internal/sidecar/gateway.go` — implement NotifyEvent handler, state transitions
- `packages/pi-aot-extension/src/extension.ts` — stdin reader, gRPC NotifyEvent calls
- `test/contract/server_sidecar_test.go` — update NotifyEvent test
- `internal/sidecar/gateway_test.go` — new state transition tests
- `packages/pi-aot-extension/src/hitl.test.ts` — update tests for stdin bridge
