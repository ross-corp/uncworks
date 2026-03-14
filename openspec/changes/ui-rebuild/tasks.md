## 1. Foundation: CSS Grid Layout and Theme

- [ ] 1.1 Define MU-TH-UR CSS custom properties in `web/src/index.css`: background colors (`--mu-bg-primary`, `--mu-bg-surface`, `--mu-bg-elevated`), text colors (`--mu-text-primary`, `--mu-text-secondary`, `--mu-text-muted`), accent (`--mu-accent`), error (`--mu-error`), glow (`--mu-glow`), scanline gradient (`--mu-scanline`), monospace font stack (`--mu-font-mono`)
- [ ] 1.2 Add `@font-face` or Google Fonts import for JetBrains Mono (primary) and Fira Code (fallback) in `web/src/index.css`
- [ ] 1.3 Set up root CSS Grid in `web/src/index.css` on `#root`: `grid-template-areas: "nav workspace detail" "nav logs logs"`, `grid-template-columns: 280px 1fr 320px`, `grid-template-rows: 1fr 200px`, full viewport height
- [ ] 1.4 Add scanline `::after` pseudo-element on `#root` with `--mu-scanline` background, `pointer-events: none`, `position: fixed`, covering full viewport
- [ ] 1.5 Install `react-resizable-panels` (or equivalent) in `web/package.json` for drag-to-resize zone boundaries
- [ ] 1.6 Rewrite `web/src/App.tsx`: remove all existing layout. Render four zone containers (`nav`, `workspace`, `detail`, `logs`) inside the CSS Grid with resizable splitters between them. Each zone is a placeholder `<div>` for now.
- [ ] 1.7 Add CSS for zone containers: `grid-area` assignments, `overflow: hidden`, MU-TH-UR surface backgrounds, 1px border between zones using `--mu-text-muted`
- [ ] 1.8 Add responsive breakpoint at 1024-1439px: redefine grid to `grid-template-columns: 48px 1fr` and `grid-template-areas: "nav workspace" "nav logs"` (detail panel hidden)
- [ ] 1.9 Add responsive breakpoint below 1024px: display centered "Use a wider screen" message, hide grid entirely

## 2. Workspace View System

- [ ] 2.1 Create `web/src/components/WorkspaceTabs.tsx`: tab bar component with tabs for Graph, Timeline, Diff, Files, Shell. Each tab shows an icon + label. Active tab has accent color bottom border with glow.
- [ ] 2.2 Create `web/src/components/WorkspaceView.tsx`: container component that renders ALL five views simultaneously but shows only the active one (`display: none` on inactive). Accepts `activeView` prop from selection state.
- [ ] 2.3 Wire `WorkspaceTabs` + `WorkspaceView` into the workspace zone in `App.tsx`. Tab clicks update `activeView` in the store.
- [ ] 2.4 Create placeholder components for each view: `OrchestrationGraphView.tsx`, `TimelineView.tsx`, `DiffView.tsx`, `FilesView.tsx`, `ShellView.tsx` — each renders a MU-TH-UR-styled placeholder with the view name and instructions
- [ ] 2.5 Add view-switching logic: selecting a spec sets `activeView: 'graph'`, double-clicking a run node sets `activeView: 'timeline'`, clicking a span sets `activeView: 'diff'`

## 3. Selection State and Store

- [ ] 3.1 Define `SelectionState` interface in `web/src/types/selection.ts`: `specId`, `runId`, `spanId`, `filePath` (all nullable strings), `activeView` (union type of view names)
- [ ] 3.2 Extend the existing agent store (or create new `createCommandCenterStore`) with selection state, spec list, and panel visibility flags (navigatorExpanded, detailVisible, logStreamMinimized)
- [ ] 3.3 Add selection cascade logic: `selectSpec(id)` clears runId/spanId and sets activeView to 'graph'; `selectRun(id)` clears spanId; `selectSpan(id)` sets activeView to 'diff'
- [ ] 3.4 Add panel state actions: `toggleNavigator()`, `toggleDetail()`, `toggleLogStream()`, `setLogStreamHeight(px)`
- [ ] 3.5 Provide the store via SolidJS context in `App.tsx` so all components can access it without prop drilling
- [ ] 3.6 Update URL routing: `/specs/:specId` sets specId selection, `/specs/:specId/runs/:runId` sets both. Navigation updates the URL; direct URL access hydrates the store.

## 4. Spec Navigator

- [ ] 4.1 Create `web/src/components/SpecNavigator.tsx`: full navigator component with header (search bar + "New Run" button), scrollable tree body, and footer
- [ ] 4.2 Create `web/src/components/SpecTreeNode.tsx`: single spec tree node showing name, run count summary, expand/collapse arrow, and status aggregate indicator
- [ ] 4.3 Create `web/src/components/RunTreeNode.tsx`: single run node showing run ID (truncated), phase status indicator (animated for Running), and expand arrow for child runs
- [ ] 4.4 Implement tree expand/collapse: clicking the arrow on a spec loads its runs (if not cached) and toggles visibility of child nodes. Clicking the arrow on a run with children toggles child visibility.
- [ ] 4.5 Add status indicators to `RunTreeNode`: pulsing green circle (CSS animation, `@keyframes pulse`) for Running, steady amber circle for Succeeded, red circle with `box-shadow` glow for Failed, gray circle for Pending
- [ ] 4.6 Wire spec click to store: `onClick` calls `store.selectSpec(id)`, which updates selection state and triggers workspace view change
- [ ] 4.7 Wire run click to store: `onClick` calls `store.selectRun(id)`, which updates detail panel and log stream
- [ ] 4.8 Add search bar at top of navigator: input with `--mu-bg-surface` background, phosphor green text, magnifying glass icon. Filters spec list by name on each keystroke (client-side filter)
- [ ] 4.9 Add "New Run" button in navigator header: phosphor green border, amber text, opens RunCreator modal on click
- [ ] 4.10 Implement virtual scrolling for large spec lists using `@tanstack/solid-virtual`: only render visible tree nodes. Install dependency in `web/package.json`.
- [ ] 4.11 Add live status updates: subscribe to run status changes (from watchAgentRun streams) and update tree node indicators in real-time without re-fetching the entire list
- [ ] 4.12 Style navigator with MU-TH-UR theme: dark surface background, phosphor green text, subtle hover glow on nodes, selected node has accent left border

## 5. Orchestration Graph View

- [ ] 5.1 Create `web/src/components/OrchestrationGraph.tsx`: wraps the DAG graph component from graph-visualization work. Receives `specId` from selection state, fetches the run graph for that spec, and renders it.
- [ ] 5.2 Style graph nodes with MU-TH-UR theme: dark node backgrounds, phosphor green borders for active nodes with animated glow (`@keyframes glow`), amber borders for succeeded, red for failed
- [ ] 5.3 Add click handler on graph nodes: clicking a run node calls `store.selectRun(id)`, updates detail panel and log stream
- [ ] 5.4 Add double-click handler on graph nodes: double-clicking calls `store.selectRun(id)` and sets `activeView: 'timeline'` to show that run's trace timeline
- [ ] 5.5 Add animated edges: edges between nodes use SVG paths with animated dash-offset for "data flowing" effect between active parent and child runs
- [ ] 5.6 Ensure graph preserves scroll position and zoom level when hidden/shown via the workspace view system (view is hidden, not unmounted)

## 6. Trace Timeline View

- [ ] 6.1 Reskin `web/src/components/TraceTimeline.tsx` with MU-TH-UR theme: dark background, phosphor green span bars, CRT scanline overlay on the timeline area, amber highlight on selected span
- [ ] 6.2 Add span click handler: clicking a span bar calls `store.selectSpan(id)` and switches workspace to Diff view if the span has file changes
- [ ] 6.3 Add phosphor glow effect on active (in-progress) spans: `box-shadow` with green glow that pulses
- [ ] 6.4 Add time axis with MU-TH-UR styling: monospace font, muted green tick marks, relative time labels
- [ ] 6.5 Ensure timeline receives `runId` from selection state and fetches/displays trace data for the selected run

## 7. Diff View

- [ ] 7.1 Reskin `web/src/components/DiffViewer.tsx` with MU-TH-UR theme: dark background, green for additions (matching `--mu-text-primary`), red for deletions (matching `--mu-error`), line numbers in muted green
- [ ] 7.2 Wire to selection state: DiffView receives `spanId` from store, fetches file changes for that span, and displays them
- [ ] 7.3 Add file tabs when a span has multiple file changes: each tab shows the filename, click to switch between diffs
- [ ] 7.4 Add syntax highlighting using a theme that matches MU-TH-UR colors (custom Monaco theme or highlight.js theme)

## 8. Files View

- [ ] 8.1 Reskin `web/src/components/FileExplorer.tsx` with MU-TH-UR theme: dark tree background, phosphor green file/folder names, folder icons in amber, file icons in green
- [ ] 8.2 Reskin `web/src/components/FilePreview.tsx`: Monaco editor (read-only) with a custom MU-TH-UR dark theme (dark background, green text, amber keywords)
- [ ] 8.3 Wire to selection state: FileExplorer receives `runId` from store, browses that run's workspace files
- [ ] 8.4 Add file click handler: clicking a file sets `store.filePath` and shows preview in the right portion of the files view (or in the detail panel)

## 9. Shell View

- [ ] 9.1 Reskin `web/src/components/ShellTerminal.tsx` with MU-TH-UR xterm.js theme: background `#0a0a0a`, foreground `#00ff41`, cursor amber (`#ff8c00`), selection rgba green
- [ ] 9.2 Wire to selection state: ShellTerminal connects to the selected run's shell endpoint
- [ ] 9.3 Add resize handler: when the workspace zone resizes (splitter drag), call `terminal.fit()` to reflow the terminal
- [ ] 9.4 Add MU-TH-UR title bar above terminal: shows "SHELL: {run-id}" with status indicator

## 10. Log Stream (Bottom Panel)

- [ ] 10.1 Create `web/src/components/LogStream.tsx`: persistent bottom panel using xterm.js in read-only mode. Title bar shows "LOGS: {run-id} — {phase}" with minimize/maximize buttons.
- [ ] 10.2 Apply MU-TH-UR xterm.js theme (same as ShellTerminal but with slightly different background `#080808` to visually distinguish)
- [ ] 10.3 Wire to selection state: LogStream attaches to the selected run's log stream. When `runId` changes, disconnect from previous stream and connect to new one.
- [ ] 10.4 Add minimize behavior: clicking minimize collapses to 32px title-bar-only. Store previous height. Restore on click or Ctrl+L.
- [ ] 10.5 Add maximize behavior: clicking maximize expands log stream to take over the workspace row (workspace hidden). Click again to restore.
- [ ] 10.6 Add auto-scroll: scroll to bottom on new log output unless user has scrolled up. Show "New output" indicator when auto-scroll is paused.
- [ ] 10.7 Add resize handler: call `terminal.fit()` when the log stream zone height changes via splitter drag

## 11. Detail Panel (Right)

- [ ] 11.1 Create `web/src/components/DetailPanel.tsx`: contextual right panel that shows different content based on selection depth: spec metadata (when only spec selected), run metadata (when run selected), span metadata (when span selected), file preview (when file selected)
- [ ] 11.2 Create `web/src/components/SpecDetail.tsx`: shows spec name, description, total runs, pass/fail ratio, creation date, last run date
- [ ] 11.3 Create `web/src/components/RunDetail.tsx`: shows run ID, phase with status badge, duration, repos list, prompt (truncated with expand), backend type, creation timestamp, and action buttons (Cancel, Send Input)
- [ ] 11.4 Create `web/src/components/SpanDetail.tsx`: shows span name, type, duration, start/end timestamps, attributes (key-value pairs), and link to view diff
- [ ] 11.5 Style all detail sub-components with MU-TH-UR theme: dark elevated background, section headers in amber, values in phosphor green, subtle dividers between sections
- [ ] 11.6 On medium viewports (1024-1439px), convert detail panel to a slide-over overlay: position fixed, right: 0, slide-in animation, close button and click-outside-to-close behavior

## 12. Run Creator Modal

- [ ] 12.1 Create `web/src/components/RunCreator.tsx`: modal overlay with Radix Dialog. Dark overlay background with green border on the modal.
- [ ] 12.2 Add spec selector: Radix Select component listing existing specs + "New Spec" option. On "New Spec", show an inline text input for the spec name.
- [ ] 12.3 Add repos section: initial row with URL (required), branch, path inputs. "Add Repository" button adds rows. Each row has a delete button (except when only one remains).
- [ ] 12.4 Add prompt textarea: full-width, required, with MU-TH-UR styling (dark bg, green text, green focus ring)
- [ ] 12.5 Add backend selector: Radix RadioGroup with Pod, KubeVirt, External options. MU-TH-UR styled radio buttons.
- [ ] 12.6 Add collapsible "Advanced" section using Radix Collapsible: devboxConfig (textarea), ttlSeconds (number input), envVars (dynamic key-value pair rows with add/remove), image (text input)
- [ ] 12.7 Add client-side validation: required fields get red glow border on submit if empty. Error messages in `--mu-error` color below each field.
- [ ] 12.8 Wire form submission: on submit, call the API to create the run. On success, close modal, add run to store, navigate to it. On error, display error message at top of form.
- [ ] 12.9 Add template presets: "Load Preset" dropdown at top of form, "Save as Preset" button at bottom. Presets stored in localStorage. Loading a preset populates all form fields.

## 13. Status Bar

- [ ] 13.1 Create `web/src/components/StatusBar.tsx`: fixed bar at the very bottom of the viewport (below the CSS Grid), 28px tall, full width
- [ ] 13.2 Left section: system health dot (green/amber/red) with tooltip showing individual service status (API, Temporal, K8s)
- [ ] 13.3 Center-left section: active run count ("N active runs" in phosphor green)
- [ ] 13.4 Center-right section: connection status ("Connected" in green or "Disconnected" with red pulse animation)
- [ ] 13.5 Right section: keyboard shortcut hints for current context (e.g., "Ctrl+G Graph | Ctrl+L Logs | Ctrl+F Files | Ctrl+K Command")
- [ ] 13.6 Style with MU-TH-UR theme: `--mu-bg-primary` background, `--mu-text-muted` for labels, `--mu-text-primary` for values, 1px top border in muted green
- [ ] 13.7 Wire health check: poll a health endpoint every 30s to determine system health status. Update the health dot accordingly.

## 14. Keyboard Shortcuts

- [ ] 14.1 Create `web/src/hooks/useKeyboardShortcuts.ts`: hook that registers global `keydown` listeners. Accepts a map of shortcut definitions (`{key, ctrl, handler}`). Returns cleanup function.
- [ ] 14.2 Add focus-tracking logic: before executing a shortcut, check if `document.activeElement` is an `input`, `textarea`, or has `contenteditable`. If so, skip the shortcut. Also check for xterm.js and Monaco focus via their container classes.
- [ ] 14.3 Register shortcuts in `App.tsx`: Ctrl+G → `store.setActiveView('graph')`, Ctrl+L → `store.toggleLogStream()`, Ctrl+F → `store.setActiveView('files')`, Ctrl+T → `store.setActiveView('shell')`, Ctrl+K → open command palette, Escape → close palette / deselect
- [ ] 14.4 Prevent default browser actions for overridden shortcuts (`e.preventDefault()` on Ctrl+F, Ctrl+T, etc.)
- [ ] 14.5 Add arrow key navigation in the navigator tree: when navigator is focused, Up/Down moves selection between visible tree nodes, Left collapses current node, Right expands it

## 15. Command Palette

- [ ] 15.1 Install `cmdk` (or equivalent SolidJS command palette library) in `web/package.json`
- [ ] 15.2 Create `web/src/components/CommandPalette.tsx`: centered overlay with search input and scrollable results list. MU-TH-UR styled: dark background, green border, phosphor green text, amber highlight on selected result.
- [ ] 15.3 Register command sources: view switching commands ("Graph View", "Files View", etc.), spec list (by name), run list (by ID prefix), panel toggles ("Toggle Logs", "Toggle Navigator")
- [ ] 15.4 Implement fuzzy matching: typing partial text matches against all command labels. Results sorted by relevance.
- [ ] 15.5 Wire result selection: selecting a command executes its action (view switch, navigation, toggle) and closes the palette. Pressing Escape closes without action.
- [ ] 15.6 Add focus management: opening the palette focuses the search input. Closing returns focus to the previously focused element.

## 16. Responsive Layout

- [ ] 16.1 Create `web/src/hooks/useResponsiveLayout.ts`: hook that tracks viewport width and returns the current breakpoint ('full' | 'medium' | 'unsupported'). Uses `matchMedia` listeners for efficient tracking.
- [ ] 16.2 Wire breakpoint to store: store panel visibility flags adjust based on breakpoint. On 'medium': auto-collapse navigator, auto-hide detail panel.
- [ ] 16.3 Implement navigator icon rail: at 'medium' breakpoint, navigator renders as 48px column showing only spec status icons (colored dots). Hover expands full navigator as an overlay (position absolute, z-index above workspace).
- [ ] 16.4 Implement detail panel overlay: at 'medium' breakpoint, detail panel renders as a fixed-position slide-in from the right. Click outside or Escape to close.
- [ ] 16.5 Implement unsupported screen message: at 'unsupported' breakpoint, replace entire grid with centered message component styled with MU-TH-UR theme.
- [ ] 16.6 Add panel toggle buttons: each panel (navigator, detail, log stream) gets a toggle button that's always visible. On medium viewports, these are the primary way to show/hide panels.

## 17. Component Reskinning

- [ ] 17.1 Reskin `web/src/components/StatusBadge.tsx`: use MU-TH-UR status colors (green/amber/red), add glow effect on active status, monospace font
- [ ] 17.2 Reskin `web/src/components/ConfirmDialog.tsx`: use Radix AlertDialog with MU-TH-UR styling (dark overlay, green border, amber action buttons)
- [ ] 17.3 Reskin `web/src/components/Toast.tsx`: dark background, phosphor green text for success, red for error, slide-in from bottom-right with glow
- [ ] 17.4 Reskin `web/src/components/ErrorBoundary.tsx`: MU-TH-UR error display with red glow, CRT flicker animation on error state
- [ ] 17.5 Reskin `web/src/components/Skeleton.tsx`: use MU-TH-UR surface colors with animated green pulse instead of gray shimmer

## 18. Integration and Wiring

- [ ] 18.1 Wire SpecNavigator to live data: on mount, fetch spec list from API. On spec expand, fetch runs. Subscribe to status updates via existing watch streams.
- [ ] 18.2 Wire OrchestrationGraph to live data: when specId changes, fetch the run graph for that spec and pass to the DAG renderer
- [ ] 18.3 Wire LogStream to live data: when runId changes, disconnect previous log stream and connect to new run's watchAgentRun log events
- [ ] 18.4 Wire DetailPanel to live data: when selection changes, fetch metadata for the most specific selected entity (span > run > spec)
- [ ] 18.5 Wire RunCreator form submission to API: call createAgentRun, handle loading/error states, navigate on success
- [ ] 18.6 Wire StatusBar health check: poll health endpoint every 30s, aggregate service statuses, update display

## 19. Verification

- [ ] 19.1 Run `npx tsc --noEmit -p web/tsconfig.json` — all new and reskinned components compile without type errors
- [ ] 19.2 Run `npm run build` in `web/` — production build succeeds
- [ ] 19.3 Run `npm run dev` in `web/` — verify four-zone layout renders, navigator shows specs, workspace tabs switch views, log stream is visible and resizable
- [ ] 19.4 Verify keyboard shortcuts: Ctrl+G/L/F/T/K all trigger correct actions, are disabled in input fields
- [ ] 19.5 Verify responsive behavior: resize browser to 1200px (medium) and verify navigator collapses, detail becomes overlay; resize to 900px and verify unsupported message
- [ ] 19.6 Verify MU-TH-UR theme: scanline overlay visible, phosphor green text throughout, glow effects on active elements, monospace font everywhere
- [ ] 19.7 Run Storybook (if configured): verify all reskinned components render correctly in isolation
