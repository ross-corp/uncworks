## 1. Terminal Runtime

- [ ] 1.1 Create `packages/tui/src/runtime.ts` with raw mode setup, screen clear, cursor hide/show, graceful exit on q/Ctrl-C
- [ ] 1.2 Create `packages/tui/src/input.ts` with keypress parser (arrow keys, Enter, Escape, printable chars, Ctrl-C)
- [ ] 1.3 Implement render loop: `createEffect` on state signals → `renderToTerminal(dashboardView(...))` on change
- [ ] 1.4 Write tests for input parser (key sequence → action mapping)

## 2. State Management

- [ ] 2.1 Create `packages/tui/src/state.ts` with SolidJS signals for runs list, selected index, selected run detail, input mode flag
- [ ] 2.2 Wire keyboard actions to state mutations (Up/Down → selection, Enter → detail/input mode, Escape → back, q → exit)
- [ ] 2.3 Write tests for state transitions (navigation wrapping, mode switching)

## 3. gRPC Data Binding

- [ ] 3.1 Create `packages/tui/src/data.ts` that uses `@aot/shared` gRPC client to fetch and stream data
- [ ] 3.2 Call `listAgentRuns()` on startup and populate runs signal
- [ ] 3.3 Call `watchAgentRun()` when selection changes; cancel previous watch
- [ ] 3.4 Implement `sendHumanInput()` call from input mode submit
- [ ] 3.5 Write tests with mock gRPC client (startup fetch, watch subscribe/cancel, input send)

## 4. HITL Input Mode

- [ ] 4.1 Add input line rendering to `views.ts` — show `> ` prompt with typed text when in input mode
- [ ] 4.2 Collect keystrokes in input mode, handle backspace, submit on Enter, cancel on Escape
- [ ] 4.3 Show success/error feedback after `SendHumanInput` response
- [ ] 4.4 Write tests for input mode rendering and keystroke handling

## 5. CLI Integration

- [ ] 5.1 Add `dashboard` subcommand to `cmd/aot/main.go` that execs `npx tsx packages/tui/src/main.ts`
- [ ] 5.2 Create `packages/tui/src/main.ts` entry point that initializes runtime + data binding
- [ ] 5.3 Add `dev:tui` task to `Taskfile.yml`
- [ ] 5.4 Smoke test: run `task dev:tui` against live cluster, navigate runs, verify live updates
