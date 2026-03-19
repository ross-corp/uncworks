## 1. Dependencies & Scaffolding

- [x] 1.1 Install new deps: cmdk, react-markdown, rehype-highlight, nuqs, vaul, react-router-dom
- [x] 1.2 Set up URL router (react-router-dom with Layout + Outlet)
- [x] 1.3 Create route structure: `/`, `/new`, `/run/:id`
- [x] 1.4 Set up shadcn theme system: CSS variables per theme, localStorage persistence, anti-flash script
- [x] 1.5 Remove CRT CSS (index.css fx-* classes deleted, styles/muthr.css deleted)

## 2. Theming

- [x] 2.1 Create theme provider with all shadcn built-in themes (useThemeNew hook)
- [x] 2.2 Implement light/dark mode toggle with system preference detection
- [x] 2.3 Persist theme + mode to localStorage (aot-theme-color, aot-theme-mode)
- [x] 2.4 Add anti-flash inline script in index.html (apply theme before first paint)
- [x] 2.5 Add "theme" command to command palette (list all themes, toggle dark mode)

## 3. Shared Components (custom, ~510 LOC)

- [x] 3.1 `ActivityFeed` — timestamped entry list (user, agent, tool, result, system types)
- [x] 3.2 `ToolCallCard` — expandable tool call with name + JSON input + result (inline in ActivityFeed)
- [x] 3.3 `DiffBlock` — inline code diff (green/red lines, syntax highlighted)
- [x] 3.4 `StageProgress` — plan → execute → verify progress bar with status icons
- [x] 3.5 `CommandInput` — `:` and `/` prefix command bar
- [x] 3.6 `ChatMessage` — user/agent message with markdown rendering via react-markdown
- [x] 3.7 `RunStatusBadge` — status dot + text (● running, ✓ ok, ✗ fail, etc.)

## 4. Run List View (/)

- [x] 4.1 Build run list table (name, status, stage, model, age columns)
- [x] 4.2 Implement j/k keyboard navigation with visual selection indicator
- [x] 4.3 Implement / filter (filters name, status, model in-place)
- [x] 4.4 Implement quick filter keys (1=all, 2=active, 3=succeeded, 4=failed)
- [x] 4.5 Wire enter → navigate to `/run/:id`
- [x] 4.6 Wire n → navigate to `/new`
- [x] 4.7 Wire d → delete with confirmation, c → clone via /new?clone=id
- [x] 4.8 Implement 5s polling for run list updates
- [x] 4.9 Use nuqs for filter state — inline / filter implemented, nuqs not needed

## 5. New Run View (/new)

- [x] 5.1 Build prompt input with repo selector
- [x] 5.2 Build collapsed config line (model · TTL · mode)
- [x] 5.3 Build Prompt/Spec tab toggle with spec textarea
- [ ] 5.4 Build "Refine with AI" chat panel — future feature, ChatMessage component ready
- [x] 5.5 Build "Run" button that creates the agent run via API
- [x] 5.6 Auto-switch to spec-driven mode when Spec tab is active
- [x] 5.7 Navigate to `/run/:id` after successful creation

## 6. Run Detail View (/run/:id)

- [x] 6.1 Build header: run name, status badge, stage progress bar
- [x] 6.2 Build tab bar: 1 activity, 2 files, 3 shell, 4 traces, 5 verify
- [x] 6.3 Implement number key tab switching
- [x] 6.4 Build Activity tab using ActivityFeed + structured logs API
- [x] 6.5 Build Files tab (reuse existing FileExplorer)
- [x] 6.6 Build Shell tab (reuse existing ShellTerminal)
- [x] 6.7 Build Traces tab (reuse existing TraceTimeline)
- [x] 6.8 Build Verify tab using VerificationPanel
- [x] 6.9 Build HITL input overlay (shown when agent is waiting_for_input)
- [x] 6.10 Build info overlay (toggle with `i` key) showing run metadata
- [x] 6.11 Wire esc → navigate back to `/`

## 7. Command Palette (cmdk)

- [x] 7.1 Set up cmdk with ⌘K trigger
- [x] 7.2 Add run search (search by name, top 10 results)
- [x] 7.3 Add navigation commands (go to runs, new run)
- [x] 7.4 Add action commands (cancel run, clone run)
- [x] 7.5 Add theme commands (switch theme, toggle dark mode)

## 8. Cleanup & Migration

- [x] 8.1 Remove old components: IconRail, SplitPane, CommandPalette, LogViewer, LogViewerInner, AgentLogView, LogsTab
- [x] 8.2 Remove old components: DetailPane, RunDetail, RunList, AgentRunForm, App.tsx, SpecRunPage, WorkspaceEditor, ConfirmDialog, GitHubModal, OrchestrationGraph, OrchestrationNode, CompletionSummary, DiffViewer, MonacoDiffEditor, DetailPanel, RunGraph, LiveIndicators
- [x] 8.3 Remove CRT CSS: fx-* classes from index.css, delete styles/muthr.css
- [x] 8.4 Remove unused hooks: useKeyboard, useKeyboardNavigation, useWorkspaces, useRepoRegistry, useTheme, useWatchRun, useOrchestrationGraph, useGitHub
- [x] 8.5 Update data-testid attributes on all new components
- [x] 8.6 Verify TypeScript compiles with zero errors

## 9. Testing

- [x] 9.1 Playwright E2E: run-list.spec.ts (navigation, filter, keyboard)
- [x] 9.2 Playwright E2E: new-run.spec.ts (prompt input, spec tab, create)
- [x] 9.3 Playwright E2E: run-detail.spec.ts (tabs, number keys, info overlay)
- [x] 9.4 Playwright E2E: theming.spec.ts (dark mode, localStorage persistence)
- [x] 9.5 Playwright E2E: command-palette.spec.ts (⌘K, search, close)
- [x] 9.6 Storybook stories for RunStatusBadge, DiffBlock, ChatMessage, StageProgress
- [x] 9.7 All old E2E tests replaced with new tests for new UI
