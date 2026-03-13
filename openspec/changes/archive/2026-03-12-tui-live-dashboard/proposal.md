## Why

The TUI currently renders static snapshots via `renderToString()` and has no interactive runtime. Users cannot navigate agent runs, view live updates, or send HITL input from the terminal. For engineers who prefer CLI workflows, the TUI should be a fully interactive alternative to the web dashboard.

## What Changes

- Add a terminal application runtime using raw mode stdin/stdout with ANSI escape sequences (no ncurses dependency).
- Implement keyboard navigation (arrow keys, Enter, q to quit) for agent run selection.
- Connect the TUI to the gRPC `WatchAgentRun` streaming API for real-time phase updates.
- Add a HITL input mode: when a selected agent is in `WaitingForInput` phase, pressing Enter opens a text input prompt that calls `SendHumanInput`.
- Add a `task dev:tui` target and `aot dashboard` CLI command to launch the TUI.

## Capabilities

### New Capabilities
- `tui-runtime`: Interactive terminal application with raw-mode input handling, screen clearing, cursor management, and 60fps render loop.
- `tui-grpc-binding`: Connect TUI state to gRPC client for live data — fetch initial runs via `ListAgentRuns`, subscribe to updates via `WatchAgentRun`, send input via `SendHumanInput`.

### Modified Capabilities

## Impact

- `packages/tui/` — New `runtime.ts` and `input.ts` modules alongside existing `renderer.ts` and `views.ts`.
- `packages/shared/` — gRPC client used by TUI (already exists).
- `cmd/aot/main.go` — New `dashboard` subcommand that spawns a Node.js TUI process.
- `Taskfile.yml` — New `dev:tui` target.
