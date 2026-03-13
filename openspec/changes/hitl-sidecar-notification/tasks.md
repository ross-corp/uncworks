## 1. Sidecar NotifyEvent Handler

- [x] 1.1 Implement `NotifyEvent` method on Gateway in `internal/sidecar/gateway.go`: set `proc.state` to `WAITING_FOR_INPUT` on `EVENT_TYPE_WAITING_FOR_INPUT`, set to `RUNNING` on `EVENT_TYPE_STARTED`, return `acknowledged: true`
- [x] 1.2 Return `FailedPrecondition` error when no process is running
- [x] 1.3 For other event types (LOG, TOOL_CALL, etc.), acknowledge without state change
- [x] 1.4 Update `TestContract_NotifyEvent_Unimplemented` in `test/contract/server_sidecar_test.go` to test the implemented handler (rename, verify acknowledged=true, verify state transitions)
- [x] 1.5 Add contract test: NotifyEvent with no process returns FailedPrecondition

## 2. Extension Stdin Bridge

- [x] 2.1 Add stdin line reader to `AOTExtension` in `packages/pi-aot-extension/src/extension.ts` using Node.js readline on `process.stdin`
- [x] 2.2 Buffer stdin lines when not waiting; when `waitForHumanInput()` is called with buffered input, resolve immediately
- [x] 2.3 When `waitForHumanInput()` is pending and a stdin line arrives, resolve the Promise with the line
- [x] 2.4 On Promise resolution (input received), call `provideHumanInput()` to clear paused/waiting state

## 3. Extension NotifyEvent Integration

- [x] 3.1 Add `@connectrpc/connect` and `@connectrpc/connect-node` dependencies to `packages/pi-aot-extension/package.json`
- [x] 3.2 Generate or import the TS client for `AgentNotificationService` from `gen/ts/aot/agent/v1/`
- [x] 3.3 Create a ConnectRPC client in `AOTExtension` targeting `http://localhost:50052` (configurable via `sidecarAddress` in config)
- [x] 3.4 In `waitForHumanInput()`: after setting paused state, call `NotifyEvent` with `EVENT_TYPE_WAITING_FOR_INPUT` and question as payload
- [x] 3.5 When input Promise resolves: call `NotifyEvent` with `EVENT_TYPE_STARTED` to signal agent resumed
- [x] 3.6 Handle NotifyEvent call failures gracefully (log warning, don't fail the agent)

## 4. Update Tests

- [x] 4.1 Update `packages/pi-aot-extension/src/hitl.test.ts` to test stdin bridge: simulate stdin line, verify Promise resolves
- [x] 4.2 Update `packages/pi-aot-extension/src/hitl.test.ts` to test buffered stdin: write line before waitForHumanInput, verify immediate resolution
- [x] 4.3 Run `npx tsc --noEmit -p packages/pi-aot-extension/tsconfig.json` — extension compiles

## 5. Verification

- [x] 5.1 Run `go test ./internal/sidecar/... ./test/contract/...` — sidecar and contract tests pass
- [x] 5.2 Run `npx tsc --noEmit` for all TS packages — type checks pass
- [ ] 5.3 Commit and push
