## 1. Dependencies & Scaffolding

- [ ] 1.1 Install new deps: `cmdk`, `react-markdown`, `rehype-highlight`, `nuqs`, `vaul`
- [ ] 1.2 Set up URL router (react-router or lightweight alternative with nuqs)
- [ ] 1.3 Create route structure: `/`, `/new`, `/run/:id`, `/run/:id/:tab`
- [ ] 1.4 Set up shadcn theme system: CSS variables per theme, localStorage persistence, anti-flash script
- [ ] 1.5 Remove CRT CSS (index.css fx-* classes, styles/muthr.css)

## 2. Theming

- [ ] 2.1 Create theme provider with all shadcn built-in themes (zinc, slate, stone, gray, neutral, red, rose, orange, green, blue, yellow, violet)
- [ ] 2.2 Implement light/dark mode toggle with system preference detection
- [ ] 2.3 Persist theme + mode to localStorage (`aot-theme`, `aot-mode`)
- [ ] 2.4 Add anti-flash inline script in index.html (apply theme before first paint)
- [ ] 2.5 Add "theme" command to command palette (list all themes, instant preview)

## 3. Shared Components (custom, ~510 LOC)

- [ ] 3.1 `ActivityFeed` — timestamped entry list (user, agent, tool, result, system types)
- [ ] 3.2 `ToolCallCard` — expandable tool call with name + JSON input + result
- [ ] 3.3 `DiffBlock` — inline code diff (green/red lines, syntax highlighted)
- [ ] 3.4 `StageProgress` — plan → execute → verify progress bar with status icons
- [ ] 3.5 `CommandInput` — `:` and `/` prefix command bar (integrates with cmdk)
- [ ] 3.6 `ChatMessage` — user/agent message with markdown rendering
- [ ] 3.7 `RunStatusBadge` — status dot + text (● running, ✓ ok, ✗ fail, etc.)

## 4. Run List View (/)

- [ ] 4.1 Build run list table (name, status, stage, model, age columns)
- [ ] 4.2 Implement j/k keyboard navigation with visual selection indicator
- [ ] 4.3 Implement / filter (filters name, status, model in-place)
- [ ] 4.4 Implement quick filter keys (1=all, 2=active, 3=succeeded, 4=failed)
- [ ] 4.5 Wire enter → navigate to `/run/:id`
- [ ] 4.6 Wire n → navigate to `/new`
- [ ] 4.7 Wire d → delete with confirmation, c → clone
- [ ] 4.8 Implement 5s polling for run list updates
- [ ] 4.9 Use nuqs for filter state in URL params (`?status=failed`)

## 5. New Run View (/new)

- [ ] 5.1 Build prompt input with repo selector
- [ ] 5.2 Build collapsed config line (model · TTL · mode) with expand toggle
- [ ] 5.3 Build Prompt/Spec tab toggle (text input vs Monaco editor)
- [ ] 5.4 Build "Refine with AI" chat panel (calls LiteLLM proxy for conversation)
- [ ] 5.5 Build "Run" button that creates the agent run via API
- [ ] 5.6 Auto-switch to spec-driven mode when Spec tab is active
- [ ] 5.7 Navigate to `/run/:id` after successful creation

## 6. Run Detail View (/run/:id)

- [ ] 6.1 Build header: run name, status badge, stage progress bar
- [ ] 6.2 Build tab bar: 1 activity, 2 files, 3 shell, 4 traces, 5 verify
- [ ] 6.3 Implement number key tab switching
- [ ] 6.4 Build Activity tab using ActivityFeed + structured logs API
- [ ] 6.5 Build Files tab (reuse existing FileTree/FileExplorer)
- [ ] 6.6 Build Shell tab (reuse existing ShellTerminal/xterm)
- [ ] 6.7 Build Traces tab (reuse existing TraceTimeline)
- [ ] 6.8 Build Verify tab using VerificationPanel
- [ ] 6.9 Build HITL input overlay (shown when agent is waiting_for_input)
- [ ] 6.10 Build info overlay (toggle with `i` key) showing run metadata
- [ ] 6.11 Wire esc → navigate back to `/`

## 7. Command Palette (cmdk)

- [ ] 7.1 Set up cmdk with ⌘K trigger
- [ ] 7.2 Add run search (search by name, filter by status)
- [ ] 7.3 Add navigation commands (go to runs, new run, settings)
- [ ] 7.4 Add action commands (cancel run, clone run, delete run)
- [ ] 7.5 Add theme commands (switch theme, toggle dark mode)

## 8. Cleanup & Migration

- [ ] 8.1 Remove old components: IconRail, SplitPane, custom CommandPalette, LogViewer, LogViewerInner
- [ ] 8.2 Remove old components: DetailPane, AgentRunForm (modal version), old App.tsx dashboard
- [ ] 8.3 Remove CRT CSS: fx-* classes from index.css, delete styles/muthr.css
- [ ] 8.4 Remove unused hooks: useKeyboard (replaced by new keyboard system)
- [ ] 8.5 Update data-testid attributes on all new components
- [ ] 8.6 Verify TypeScript compiles with zero errors

## 9. Testing

- [ ] 9.1 Update Playwright E2E tests for new navigation patterns (j/k, enter, esc)
- [ ] 9.2 Update Playwright tests for new run creation flow (prompt → run)
- [ ] 9.3 Update Playwright tests for run detail (activity feed, tabs)
- [ ] 9.4 Add Playwright test for theme switching and persistence
- [ ] 9.5 Add Playwright test for command palette (cmdk)
- [ ] 9.6 Add Storybook stories for all new custom components
- [ ] 9.7 Verify all existing E2E tests pass or are updated
