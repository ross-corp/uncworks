## Why

When an agent run executes, there's no way to see what it's doing. The sidecar captures pi-coding-agent's stdout/stderr and has a `StreamOutput` RPC, the EventBus publishes phase changes, and the `WatchAgentRun` streaming RPC is defined in proto with a client method ready — but none of it is connected. The web UI polls every 5 seconds and shows a phase badge. Users can't see logs, can't browse the workspace the agent is modifying, can't drop into a shell to debug, and completed pods are deleted immediately so there's nothing to inspect after the fact. For a platform that orchestrates autonomous coding agents, this is like flying blind.

## What Changes

- **Live log streaming** — wire the sidecar's `StreamOutput` (stdout/stderr) through the control plane EventBus into `WatchAgentRun`, render in the web UI with xterm.js for full ANSI color/terminal rendering. Real-time output as the agent works.
- **File explorer** — new REST API endpoints that exec into the agent pod to list directories and read files, rendered as a tree view with Monaco read-only editor for file preview. Browse `/workspace/src/` while the agent is running or after completion.
- **Interactive shell** — WebSocket endpoint that bridges browser ↔ K8s pod exec, rendered with xterm.js as a full interactive terminal. Drop into `bash` in the agent pod from the browser.
- **Pod retention** — keep agent pods alive for a configurable duration after workflow completion (default 30 minutes) so logs, files, and shell remain accessible. Persist log output to the AgentRun CRD status for permanent availability after pod deletion.
- **Detail panel redesign** — transform the detail panel from a metadata display into a tabbed workspace: Info | Logs | Files | Shell. The panel becomes the primary way to interact with running and completed agent runs.
- **E2E test coverage** — Playwright tests for log streaming, file explorer navigation, and shell interaction. Go E2E tests for the new API endpoints and log persistence.

## Capabilities

### New Capabilities
- `log-streaming`: Real-time agent log streaming from sidecar through control plane to web UI — wiring StreamOutput → EventBus → WatchAgentRun → xterm.js, plus log persistence for completed runs
- `file-explorer`: REST API for browsing agent pod filesystems and viewing file contents, with tree component and Monaco preview in the web UI
- `interactive-shell`: WebSocket-based interactive terminal into agent pods, bridging browser xterm.js to K8s pod exec via SPDY
- `pod-retention`: Configurable post-completion pod retention with log persistence, replacing immediate pod deletion
- `detail-panel-tabs`: Redesigned detail panel with tabbed interface (Info, Logs, Files, Shell) replacing the current flat metadata layout

### Modified Capabilities
<!-- No existing spec-level requirements change -->

## Impact

- **Proto** (`api.proto`): New `StreamLogs` or enhanced `WatchAgentRun` event types, new REST endpoints for files/exec
- **Sidecar** (`internal/sidecar/gateway.go`): Bridge `StreamOutput` to control plane, possibly new file listing RPC
- **API server** (`cmd/apiserver/main.go`, `internal/server/`): New REST endpoints for file listing, file content, WebSocket exec proxy
- **Workflow** (`internal/temporal/workflow.go`): Pod retention timer, log persistence activity before cleanup
- **Activities** (`internal/temporal/activities.go`): New activities for log collection, delayed cleanup
- **Controller** (`internal/controller/`): Bridge sidecar log stream into EventBus for WatchAgentRun subscribers
- **Web UI**: New components — LogViewer (xterm.js), FileExplorer (tree + Monaco), ShellTerminal (xterm.js + WebSocket), TabPanel redesign of AgentRunDetailPanel
- **Web dependencies**: `xterm` + `@xterm/addon-fit` + `@xterm/addon-web-links` npm packages
- **CRD types** (`api/v1alpha1/types.go`): Add `LogOutput` field to AgentRunStatus for persisted logs, `RetainPodMinutes` to AgentRunSpec
- **E2E tests** (`e2e/`, `web/e2e/`): New Go tests for file/exec/log endpoints, Playwright tests for log viewer, file tree, shell terminal
