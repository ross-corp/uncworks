## 1. Semantic Color System

- [x] 1.1 Define semantic color tokens in `web/src/index.css` under `:root` (light theme): `--color-success` (green), `--color-success-muted`, `--color-active` (blue), `--color-active-muted`, `--color-warning` (amber), `--color-warning-muted`, `--color-error` (red), `--color-error-muted`, `--color-neutral` (gray), `--color-neutral-muted`
- [x] 1.2 Define the same semantic color tokens under `.dark` scope in `web/src/index.css` with values adjusted for contrast on dark backgrounds — same hue per intent, different lightness/saturation
- [x] 1.3 Define accent color tokens (`--color-accent`, `--color-accent-hover`) for interactive elements (buttons, links) in both `:root` and `.dark` scopes
- [x] 1.4 Define surface color tokens for both themes: `--color-bg-primary`, `--color-bg-surface`, `--color-bg-elevated`, `--color-text-primary`, `--color-text-secondary`, `--color-text-muted`, `--color-border`
- [x] 1.5 Add CSS `@keyframes pulse` animation for the active status dot (opacity cycles between 1.0 and 0.4 over 2s)
- [x] 1.6 Create a status-to-token mapping utility in `web/src/lib/statusColors.ts` that maps run phases (Succeeded, Running, Pending, Failed, Cancelled) to the corresponding semantic token names

## 2. Theme Toggle Component

- [x] 2.1 Create `web/src/components/ThemeToggle.tsx`: button component with sun icon (in dark mode) and moon icon (in light mode), `aria-label="Toggle theme"`
- [x] 2.2 Create `web/src/hooks/useTheme.ts`: hook/store that manages the current theme. Reads from localStorage on init, falls back to `prefers-color-scheme` media query, defaults to dark
- [x] 2.3 Wire theme state to the document: toggling theme adds/removes `.dark` class on `<html>` element and writes "dark" or "light" to `localStorage` under a consistent key (e.g., `aot-theme`)
- [x] 2.4 Add anti-flash script: inline `<script>` in `web/index.html` that reads localStorage and sets `.dark` class before first paint to prevent light-to-dark flash
- [x] 2.5 Wrap MU-TH-UR effects (scanline overlay, glow classes) in `.dark &` CSS selectors so they are only active in dark mode. In light mode, scanlines `display: none` and glow `box-shadow: none`
- [x] 2.6 Place the ThemeToggle component in the application header, right-aligned

## 3. RunCard Component

- [x] 3.1 Create `web/src/components/RunCard.tsx`: card component accepting a run object prop. Renders status dot, run name (bold), repo name (muted), prompt preview (single line, truncated), time-ago label
- [x] 3.2 Implement status dot as a small circle element whose `background-color` is set via the semantic token for the run's phase. Apply the pulse animation class when phase is "Running"
- [x] 3.3 Implement time-ago display using a lightweight relative-time formatter (e.g., `Intl.RelativeTimeFormat` or a small utility). Show "3m ago", "2h ago", "1d ago" etc.
- [x] 3.4 Add hover state: subtle background color change on mouse hover, `cursor: pointer`
- [x] 3.5 Add selected state: left border or outline using `--color-accent`, slightly elevated background color. Controlled via a `selected` prop
- [x] 3.6 Make the entire card surface clickable. `onClick` prop calls parent handler with the run ID
- [x] 3.7 Ensure RunCard works in both dark and light themes by using only CSS custom property tokens for all colors

## 4. RunFeed Component

- [x] 4.1 Create `web/src/components/RunFeed.tsx`: renders a vertical scrollable list of RunCard components. Accepts `runs` array, `selectedRunId`, `onSelectRun` callback, and `isLoading` flag
- [x] 4.2 Implement reverse-chronological ordering: sort runs by creation time descending before rendering
- [x] 4.3 Implement empty state: when `runs` is empty and `isLoading` is false, render centered "No runs yet" message with a call-to-action to create a run
- [x] 4.4 Implement loading state: when `isLoading` is true, render 3-5 skeleton placeholder cards using the existing Skeleton component
- [x] 4.5 Wire RunFeed into the main layout to replace the existing AgentRunTable component. Pass the same data source that AgentRunTable used
- [x] 4.6 Connect RunFeed card clicks to open the detail view (set selected run ID in the store/state)

## 5. Filter Sidebar

- [x] 5.1 Create `web/src/components/FilterSidebar.tsx`: sidebar component containing filter groups. Replace the existing Sidebar component in the layout
- [x] 5.2 Create `web/src/components/FilterChipGroup.tsx`: reusable component that renders a label and a set of toggle chips. Accepts `label`, `options`, `selected`, `onToggle` props
- [x] 5.3 Create `web/src/components/FilterChip.tsx`: single chip component with active/inactive visual states. Active state uses `--color-accent` background. Inactive state uses `--color-bg-surface`
- [x] 5.4 Implement Status filter group: chips for All, Active, Succeeded, Failed. "All" selected by default. Selecting "All" clears others. Selecting specific statuses deselects "All". Multiple non-All chips can be active simultaneously
- [x] 5.5 Implement Repo filter group: auto-populate chips from the repos present in the loaded runs. Each chip has an X to exclude that repo. All repos included by default
- [x] 5.6 Implement Model filter group: auto-populate chips from models in the loaded runs. Toggle behavior
- [x] 5.7 Implement Workspace filter group: auto-populate chips from workspaces in the loaded runs. Toggle behavior
- [x] 5.8 Create filter state store/signals: track active filters for status, repos, models, workspaces. Expose a computed/derived filtered runs list that the RunFeed consumes
- [x] 5.9 Add "+ New Run" button at the top of the sidebar. On click, open the AgentRunForm dialog
- [x] 5.10 Remove navigation links: ensure no "Repositories" or "Events" navigation links exist in the sidebar. Remove the ReposView and EventsView route destinations if they were sidebar-triggered

## 6. Detail View

- [x] 6.1 Create `web/src/components/RunDetail.tsx`: full-width detail view that replaces the feed when a run is selected. Contains a header and a tabbed content area
- [x] 6.2 Implement detail header: displays run name, status (using semantic color badge), and a close button (X icon). Add breadcrumb "Runs / {run-name}" where clicking "Runs" closes the detail
- [x] 6.3 Implement tab bar with tabs: Info, Logs, Files, Shell, Traces. Active tab has accent-colored bottom border. Info tab active by default
- [x] 6.4 Implement Info tab content: display run status, duration, repository list, full prompt text, creation timestamp, completion timestamp. Lay out as a structured summary
- [x] 6.5 Implement Logs tab: render the existing LogViewer component for the selected run. Lazy-load log data only when the tab is activated
- [x] 6.6 Implement Files tab: render the existing FileExplorer/FileTree components for the selected run's workspace. Lazy-load file listing only when the tab is activated
- [x] 6.7 Implement Shell tab: render the existing ShellTerminal component connected to the selected run. Connect the terminal only when the tab is activated
- [x] 6.8 Implement Traces tab: render the existing TraceTimeline component for the selected run. Lazy-load trace data only when the tab is activated
- [x] 6.9 Wire close button: clicking X sets selected run to null, hiding the detail view and showing the feed
- [x] 6.10 Wire Escape key: pressing Escape when detail is open and no input is focused closes the detail view (coordinate with keyboard-navigation spec)

## 7. Layout Rewrite

- [x] 7.1 Rewrite `web/src/App.tsx` layout: replace the existing layout with a two-column layout — FilterSidebar on the left, content area on the right. Content area renders either RunFeed or RunDetail based on whether a run is selected
- [x] 7.2 Add application header bar: contains the app title/logo on the left, search input in the center, ThemeToggle on the right
- [x] 7.3 Remove the existing `Layout.tsx` component usage if it no longer matches the new layout structure. Replace with the new layout directly in App.tsx or a new Layout component
- [x] 7.4 Ensure the sidebar is a fixed width (e.g., 280px) and the content area fills remaining space
- [x] 7.5 Remove the existing `AgentRunDetailPanel.tsx` side panel integration from the layout — detail is now full-width via RunDetail

## 8. StatusBadge Migration

- [x] 8.1 Update `web/src/components/StatusBadge.tsx`: replace all hardcoded or mono-amber color references with semantic color tokens. Map each phase to its semantic token (Succeeded -> success, Running -> active, Pending -> warning, Failed -> error, Cancelled -> neutral)
- [x] 8.2 Update StatusBadge background to use the muted variant of the semantic token and text/icon to use the foreground variant
- [x] 8.3 Audit all StatusBadge usages across the codebase and verify they render correctly with the new semantic colors in both themes
- [x] 8.4 Update StatusBadge Storybook stories to show all five status variants with the new semantic colors

## 9. Keyboard Navigation

- [x] 9.1 Create `web/src/hooks/useKeyboardNavigation.ts`: hook that registers global `keydown` listeners for j, k, Enter, Escape, and /. Includes guard logic to skip when an input/textarea/select/contenteditable is focused
- [x] 9.2 Implement j/k navigation: maintain a `selectedIndex` in the feed. `j` increments (clamped to list length - 1), `k` decrements (clamped to 0). Update the selected run ID in the store
- [x] 9.3 Implement scroll-into-view: when j/k changes selection, call `scrollIntoView({ block: 'nearest' })` on the newly selected RunCard element
- [x] 9.4 Implement Enter to open: when a run is selected and Enter is pressed, open the detail view for that run
- [x] 9.5 Implement Escape to close: when the detail view is open, Escape closes it and returns to the feed with the previous selection intact
- [x] 9.6 Implement / to focus search: pressing / focuses the search input in the header. Prevent the / character from being typed into the input (`e.preventDefault()`)
- [x] 9.7 Wire keyboard navigation hook into App.tsx or the main layout component. Ensure it activates when the feed is visible and deactivates when detail is open (except for Escape)

## 10. Search Integration

- [x] 10.1 Add a search input in the header bar. Styled with theme tokens. Placeholder text "Search runs..."
- [x] 10.2 Implement client-side search: filter runs by name, repo, or prompt content as the user types. Update the filtered runs list that the RunFeed displays
- [x] 10.3 Integrate search with the filter state: search is additive with sidebar filters. Runs must match both the search query and active filters to appear

## 11. Form/Dialog Updates for New Theme

- [x] 11.1 Update `web/src/components/AgentRunForm.tsx`: replace any hardcoded colors with semantic/theme tokens. Ensure inputs, labels, and buttons use `--color-bg-surface`, `--color-text-primary`, `--color-accent` tokens
- [x] 11.2 Update `web/src/components/ConfirmDialog.tsx`: apply theme tokens for backgrounds, text, and button colors
- [x] 11.3 Update `web/src/components/Toast.tsx`: success toasts use `--color-success`, error toasts use `--color-error`
- [x] 11.4 Update `web/src/components/GitHubModal.tsx`: apply theme tokens for all colors
- [x] 11.5 Verify all dialogs and forms render correctly in both dark and light mode

## 12. Cleanup

- [x] 12.1 Remove `web/src/components/AgentRunTable.tsx` and its Storybook story file after RunFeed is wired and verified
- [x] 12.2 Remove `web/src/components/AgentRunDetailPanel.tsx` and its Storybook story file after RunDetail is wired and verified
- [x] 12.3 Remove `web/src/components/Sidebar.tsx` and its Storybook story file after FilterSidebar is wired and verified
- [x] 12.4 Remove `web/src/components/ReposView.tsx` — repos are now filter chips, not a separate view
- [x] 12.5 Remove `web/src/components/EventsView.tsx` and its Storybook story file — events are not a separate view
- [x] 12.6 Remove old mono-amber CSS tokens from `web/src/index.css` that are fully replaced by the semantic and theme token system
- [x] 12.7 Remove any unused imports and dead code references to the deleted components throughout the codebase
- [x] 12.8 Remove the old `Layout.tsx` component if it was fully replaced

## 13. E2E Test Updates

- [x] 13.1 Update Playwright selectors: replace any selectors referencing AgentRunTable with selectors for RunFeed/RunCard
- [x] 13.2 Update test flows that click table rows to instead click RunCard elements
- [x] 13.3 Update test flows that interact with the side panel to interact with the full-width RunDetail view
- [x] 13.4 Update sidebar-related tests: replace navigation link assertions with filter chip assertions
- [x] 13.5 Add E2E test: verify theme toggle switches between dark and light mode and persists across reload
- [x] 13.6 Add E2E test: verify status filter chips filter the RunFeed correctly
- [x] 13.7 Add E2E test: verify j/k keyboard navigation moves selection through RunCards
- [x] 13.8 Add E2E test: verify Enter opens detail and Escape closes it

## 14. Verification

- [x] 14.1 Run `npx tsc --noEmit -p web/tsconfig.json` — all new and updated components compile without type errors
- [x] 14.2 Run `npm run build` in `web/` — production build succeeds with no errors
- [x] 14.3 Run `npm run dev` in `web/` — verify the card feed renders with semantic-colored status dots, sidebar shows filter chips, theme toggle works
- [x] 14.4 Verify dark mode: black background, light text, MU-TH-UR scanlines visible, glow effects active, semantic status colors correct
- [x] 14.5 Verify light mode: white background, dark text, no scanlines, no glow, semantic status colors correct with good contrast
- [x] 14.6 Verify keyboard navigation: j/k moves selection, Enter opens detail, Escape closes detail, / focuses search, shortcuts disabled in inputs
- [x] 14.7 Verify filter sidebar: status chips filter correctly, repo chips auto-populate, multiple filters combine correctly, "+ New Run" opens form
- [x] 14.8 Verify detail view: opens full-width replacing feed, tabs switch content, lazy loading works, all three close methods (X, Escape, breadcrumb) work
- [x] 14.9 Verify no old components remain: AgentRunTable, AgentRunDetailPanel, Sidebar, ReposView, EventsView are deleted and no imports reference them
- [x] 14.10 Run existing E2E test suite — all updated tests pass
