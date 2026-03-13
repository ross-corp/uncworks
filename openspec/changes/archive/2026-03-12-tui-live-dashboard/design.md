## Context

The TUI package (`packages/tui/`) has a working ANSI renderer and view components (header, list, detail, dashboard) but no interactive runtime. The renderer outputs static strings. There's no input handling, no screen management, and no connection to live data. The web dashboard has all these features via SolidJS + WebSocket; the TUI needs parity.

## Goals / Non-Goals

**Goals:**
- Interactive TUI that runs in any terminal emulator (xterm-256color compatible).
- Keyboard-driven navigation matching the existing view structure (list + detail).
- Live data via gRPC streaming, same source of truth as web dashboard.
- HITL input from the terminal for `WaitingForInput` agents.

**Non-Goals:**
- Mouse support (terminal mouse events are inconsistent across terminals).
- Split panes or resizable layouts (single column is sufficient for v1).
- Log streaming in the TUI (phase + status is enough; full logs can use `kubectl logs`).
- Custom themes or color configuration.

## Decisions

### 1. Raw stdin + ANSI output, no framework dependency

Use Node.js `process.stdin.setRawMode(true)` for input and direct `process.stdout.write()` with ANSI escape codes for output. The existing `renderToString()` already produces ANSI output — we just need a loop that clears the screen and re-renders on state change.

**Alternative considered:** Ink (React-based TUI framework). Rejected because we already have SolidJS views and a custom renderer; adding React is redundant.

### 2. Signal-based reactivity from SolidJS

Use SolidJS `createSignal` and `createEffect` for state management. When the gRPC stream delivers an update, we update the signal, which triggers a re-render via `createEffect`. This reuses the existing `@aot/shared` store pattern.

### 3. gRPC-Web or direct gRPC via @grpc/grpc-js

Use `@grpc/grpc-js` directly from Node.js (the TUI runs in Node, not a browser). The `@aot/shared` gRPC client already wraps this. The TUI calls `listAgentRuns()` on startup, then `watchAgentRun()` for the selected run.

### 4. Inline text input for HITL

When the user presses Enter on a `WaitingForInput` agent, switch to "input mode": show a `> ` prompt at the bottom, collect keystrokes, submit on Enter, cancel on Escape. This avoids spawning an external editor.

## Risks / Trade-offs

- **[Terminal compatibility]** → ANSI escape codes are standard but some terminals handle cursor movement differently. Mitigated by testing in common terminals (xterm, iTerm2, GNOME Terminal, Windows Terminal).
- **[Node.js gRPC in TUI]** → The TUI inherits Node.js gRPC client startup overhead (~200ms). Acceptable for a long-running dashboard process.
- **[No scrolling for long lists]** → v1 shows as many runs as fit in terminal height. Pagination or virtual scrolling is a follow-up.
