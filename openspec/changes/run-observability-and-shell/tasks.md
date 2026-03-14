## 1. Proto/CRD: New Fields

- [x] 1.1 Add `int32 retain_pod_minutes = 14` to proto `AgentRunSpec` in `api.proto` (default 30 in workflow)
- [x] 1.2 Add `string log_output = 7` to proto `AgentRunStatus` in `api.proto`
- [x] 1.3 Add `RetainPodMinutes int32` and `LogOutput string` to CRD types in `types.go`
- [x] 1.4 Regenerate proto Go + TS code (`buf generate`)
- [x] 1.5 Update `specProtoToCRD` and `crdToProto` in `grpc.go` to pass through new fields
- [x] 1.6 Update shared TS types (`AgentRunSpec.retainPodMinutes`, `AgentRunStatus.logOutput`)
- [x] 1.7 Update `toAgentRun` mapping in shared gRPC client

## 2. Pod Retention

- [x] 2.1 Pass `RetainPodMinutes` through controller → WorkflowInput → workflow
- [x] 2.2 In `AgentRunWorkflow` defer block: after terminal state, sleep for `retainPodMinutes` before calling `CleanupPod`
- [x] 2.3 Before cleanup, execute new `CollectLogs` activity that reads sidecar container logs via K8s API and stores on CRD status `LogOutput` (truncated to 1MB)
- [x] 2.4 Add `CollectLogs` activity to `activities.go`: uses K8s pod log API for `rpc-gateway` container, returns log string
- [x] 2.5 Add `RetainUntil` timestamp to CRD status (set when terminal phase reached + retention duration)
- [x] 2.6 Update workflow tests for retention delay and log collection

## 3. Log Streaming: Sidecar → EventBus → WatchAgentRun

- [x] 3.1 Add `CollectAgentLogs` long-running activity: connects to sidecar StreamOutput RPC, publishes each line to EventBus as `AGENT_RUN_EVENT_TYPE_LOG` event
- [x] 3.2 Start `CollectAgentLogs` activity in workflow after agent starts (parallel with status polling)
- [x] 3.3 Update `WatchAgentRun` gRPC handler to include LOG events from EventBus (verify existing handler already forwards all EventBus events)
- [x] 3.4 Update web `AgentRunEvent` types to handle `log` event type with payload as raw terminal output
- [x] 3.5 Add `useWatchRun` React hook that subscribes to `watchAgentRun()` and buffers log events

## 4. Log Viewer Component (xterm.js)

- [x] 4.1 Install `@xterm/xterm`, `@xterm/addon-fit`, `@xterm/addon-web-links` npm dependencies
- [x] 4.2 Create `LogViewer` component: lazy-loaded xterm.js terminal in read-only mode, accepts log lines array, auto-scrolls, renders ANSI colors
- [x] 4.3 Support both streaming mode (live run → `useWatchRun` hook pushes lines) and static mode (completed run → render persisted `logOutput`)
- [x] 4.4 Add search in logs (xterm.js addon-search or simple Ctrl+F)
- [x] 4.5 Style terminal to match design system (surface-1 background, edge border, rounded corners)

## 5. File Explorer API Endpoints

- [x] 5.1 Create `internal/server/files.go` with `FileHandler` struct holding K8s client
- [x] 5.2 Implement `GET /api/v1/runs/{id}/files?path=...`: look up pod from AgentRun CRD, exec `ls -la --time-style=long-iso` in rpc-gateway container, parse into JSON response `{entries: [{name, type, size, modified}]}`
- [x] 5.3 Implement `GET /api/v1/runs/{id}/files/content?path=...`: exec `cat` in rpc-gateway container, return raw file content with detected Content-Type
- [x] 5.4 Add RBAC: ensure API server service account has `pods/exec` permission
- [x] 5.5 Register file handlers on the API server mux in `cmd/apiserver/main.go`
- [x] 5.6 Add error handling: pod not found (404), path not found (404), permission denied (403)

## 6. File Explorer UI Component

- [x] 6.1 Create `FileTree` component: collapsible tree with directory/file icons, lazy-loads children on directory expand via API calls
- [x] 6.2 Create `FilePreview` component: read-only Monaco editor with auto-detected language from file extension
- [x] 6.3 Create `FileExplorer` component: split pane with FileTree (left) and FilePreview (right), manages selected file state
- [x] 6.4 Add `useFiles` hook: `listDir(runId, path)` and `readFile(runId, path)` using fetch to REST endpoints
- [x] 6.5 Handle pod-not-available state: show "Pod expired" message with last-known directory tree if cached

## 7. Interactive Shell: WebSocket Exec Bridge

- [x] 7.1 Create `internal/server/exec.go` with WebSocket handler for `/api/v1/runs/{id}/exec`
- [x] 7.2 Implement WebSocket-to-SPDY bridge: upgrade HTTP to WebSocket, look up pod, open K8s exec session (`bash -l` in rpc-gateway container), pipe stdin/stdout/stderr
- [x] 7.3 Handle terminal resize messages: parse JSON `{type: "resize", cols, rows}` from WebSocket, send to SPDY resize channel
- [x] 7.4 Handle connection lifecycle: clean up SPDY session on WebSocket close and vice versa
- [x] 7.5 Register exec handler on API server mux
- [x] 7.6 Add `gorilla/websocket` dependency

## 8. Shell Terminal UI Component

- [x] 8.1 Create `ShellTerminal` component: xterm.js terminal with WebSocket attachment, sends keystrokes, renders output, handles resize
- [x] 8.2 Implement WebSocket connection management: connect on mount, reconnect on disconnect, show connection status
- [x] 8.3 Send resize events when terminal or panel dimensions change (xterm.js fit addon)
- [x] 8.4 Handle pod-not-available: show "Pod is no longer available" with retention info
- [x] 8.5 Lazy-load xterm.js (same pattern as Monaco SpecEditor)

## 9. Detail Panel Redesign: Tabs

- [x] 9.1 Refactor `AgentRunDetailPanel` into tabbed layout: tab bar at top (Info | Logs | Files | Shell), content area below
- [x] 9.2 Extract current metadata display into `InfoTab` sub-component (zero behavior change)
- [x] 9.3 Create `LogsTab`: wraps LogViewer with useWatchRun hook for live runs, falls back to persisted logOutput
- [x] 9.4 Create `FilesTab`: wraps FileExplorer, disabled when pod not available
- [x] 9.5 Create `ShellTab`: wraps ShellTerminal, disabled when pod not available
- [x] 9.6 Add tab availability indicators: Logs always enabled, Files/Shell disabled with tooltip when pod expired
- [x] 9.7 Show "Pod expires in X min" countdown for retained pods
- [x] 9.8 Add `data-testid` attributes to all new tab elements

## 10. Web Type & Hook Updates

- [x] 10.1 Add `retainPodMinutes` to web `AgentRunSpec` type, `logOutput` and `retainUntil` to `AgentRunStatus`
- [x] 10.2 Update `mapRun()` in `useClient.ts` to pass through new fields
- [x] 10.3 Update `AgentRunForm` to include "Retain Pod (min)" field (number input, default 30)
- [x] 10.4 Update `handleCreate` in `App.tsx` to pass `retainPodMinutes` to API

## 11. E2E Tests: Go API

- [x] 11.1 Add `e2e/observability_test.go` with `TestE2E_LogStreaming`: create run, subscribe WatchAgentRun, verify LOG events received
- [x] 11.2 Add `TestE2E_FileExplorer_ListDir`: create run, wait for Running, GET `/api/v1/runs/{id}/files?path=/workspace`, verify JSON listing
- [x] 11.3 Add `TestE2E_FileExplorer_ReadFile`: GET file content, verify body
- [x] 11.4 Add `TestE2E_ExecEndpoint`: WebSocket connect, send `echo hello\n`, verify `hello` in response
- [x] 11.5 Add `TestE2E_PodRetention`: create run with retain_pod_minutes=1, wait for Succeeded, verify pod still exists
- [x] 11.6 Add `TestE2E_LogPersistence`: create run, wait for completion + retention expiry, verify logOutput on CRD

## 12. E2E Tests: Playwright

- [x] 12.1 Create `web/e2e/observability.spec.ts` with test: select running run → Logs tab → verify terminal renders with content
- [x] 12.2 Add test: select running run → Files tab → verify tree renders → click file → preview renders
- [x] 12.3 Add test: select running run → Shell tab → verify terminal renders → type `ls` → verify output
- [x] 12.4 Add test: select completed run (pod expired) → Files tab disabled → Logs tab shows persisted content

## 13. Verification

- [x] 13.1 Run `go build ./...` — all Go code compiles
- [x] 13.2 Run `npx tsc --noEmit -p web/tsconfig.json` — web compiles
- [x] 13.3 Run `go test ./internal/... ./test/...` — all existing tests pass
- [x] 13.4 Manually verify: create run → watch logs stream → browse files → open shell → run completes → logs persist
