## 1. Token Simplification

- [x] 1.1 Audit all CSS custom properties in `web/src/` — list every `--color-*` and `--*` token currently defined
- [x] 1.2 Define the 10 canonical tokens in `:root` scope: `--color-bg`, `--color-fg`, `--color-border`, `--color-muted`, `--color-accent`, `--color-success`, `--color-active`, `--color-warning`, `--color-error`, `--color-neutral`
- [x] 1.3 Define `.dark` scope overrides for all 10 tokens
- [x] 1.4 Find-and-replace all old token references across components to use canonical tokens
- [x] 1.5 Delete all unused CSS custom property definitions
- [x] 1.6 Remove any `text-transform: uppercase` declarations — normal case everywhere
- [x] 1.7 Verify IoskeleyMono font-family is applied globally at 13px base size

## 2. Delete Old Components

- [x] 2.1 Delete `RunCard` component and its imports
- [x] 2.2 Delete `RunFeed` component and its imports
- [x] 2.3 Delete `FilterSidebar` component and its imports
- [x] 2.4 Delete `FilterChip` component and its imports
- [x] 2.5 Delete `FilterChipGroup` component and its imports
- [x] 2.6 Remove all references to deleted components from `App.tsx` and any barrel exports
- [x] 2.7 Delete any orphaned CSS/style files associated with deleted components

## 3. RunList Component (Dense Table)

- [x] 3.1 Create `web/src/components/RunList.tsx` with `<table>` element and `table-layout: fixed`
- [x] 3.2 Implement table header row with columns: status (20px), ID (100px), prompt (flex), repo (120px), phase (80px), time (60px)
- [x] 3.3 Implement table body rows at 32px height with all 6 data cells
- [x] 3.4 Add status dot rendering: colored circle using semantic tokens, pulse animation for running phase
- [x] 3.5 Add prompt truncation with `text-overflow: ellipsis` and `title` attribute for full text
- [x] 3.6 Add alternating row backgrounds: transparent for odd, `--color-muted` at 5% for even
- [x] 3.7 Add selected row styling: `--color-accent` at 10% background, 2px left accent border
- [x] 3.8 Wire click handler for single-click selection (one row at a time)
- [x] 3.9 Wire double-click handler to open detail pane
- [x] 3.10 Add `role="grid"` and `aria-activedescendant` for accessibility

## 4. Icon Rail

- [x] 4.1 Create `web/src/components/IconRail.tsx` — 48px fixed-width vertical rail
- [x] 4.2 Add filter funnel icon with click handler to open filter popover
- [x] 4.3 Create filter popover with radio buttons: All, Active, Done, Failed
- [x] 4.4 Wire filter selection to update run list filter state and close popover
- [x] 4.5 Add visual indicator on filter icon when a non-All filter is active
- [x] 4.6 Add plus icon with click handler to open create run dialog
- [x] 4.7 Add sun/moon theme toggle icon with click handler
- [x] 4.8 Add tooltip component: 200ms hover delay, positioned right of rail

## 5. Command Palette

- [x] 5.1 Create `web/src/components/CommandPalette.tsx` using Radix Dialog as portal overlay
- [x] 5.2 Implement search input at top with auto-focus on open
- [x] 5.3 Implement result list with three group types: Runs, Commands, Filters — each with icon and label
- [x] 5.4 Implement case-insensitive `includes()` search across run names, IDs, prompts, and repos
- [x] 5.5 Add built-in commands: New Run, Toggle Theme, Filter Active, Filter Done, Filter Failed, Show All
- [x] 5.6 Wire Enter to execute highlighted result and close palette
- [x] 5.7 Wire Escape to close palette and restore previous focus
- [x] 5.8 Wire arrow keys (Up/Down) to navigate result list with highlight
- [x] 5.9 Implement most-recently-used list (up to 5 items) shown when input is empty
- [x] 5.10 Store MRU list in localStorage, update on each command execution

## 6. Split Pane Layout

- [x] 6.1 Create `web/src/components/SplitPane.tsx` using CSS grid with `grid-template-columns`
- [x] 6.2 Implement closed state: `1fr 0` — detail pane hidden
- [x] 6.3 Implement open state: `1fr 1fr` (or stored width) — both panes visible
- [x] 6.4 Add CSS transition on `grid-template-columns` for slide animation
- [x] 6.5 Create drag handle element between panes with `col-resize` cursor
- [x] 6.6 Implement mousedown/mousemove/mouseup tracking on drag handle to resize columns
- [x] 6.7 Enforce minimum pane width of 200px during drag
- [x] 6.8 Store last pane width in localStorage key `clean-ui-pane-width`
- [x] 6.9 Restore pane width from localStorage on open; validate within [200px, 80vw] range

## 7. App.tsx Rewrite

- [x] 7.1 Rewrite `web/src/App.tsx` layout: IconRail (left) + SplitPane (RunList + DetailPane)
- [x] 7.2 Wire filter state: shared signal between IconRail popover, command palette, and RunList
- [x] 7.3 Wire selection state: shared signal between RunList, SplitPane open/close, and DetailPane
- [x] 7.4 Wire theme toggle: class `.dark` on document root, persisted in localStorage
- [x] 7.5 Mount CommandPalette at App root level (portal renders on top of everything)
- [x] 7.6 Mount KeyboardHintBar at bottom of viewport

## 8. Keyboard System

- [x] 8.1 Create `web/src/hooks/useKeyboard.ts` with single `document` event listener
- [x] 8.2 Implement `isInputFocused()` guard: check for input, textarea, contenteditable
- [x] 8.3 Wire j/k for list navigation with wrap-around and scroll-into-view
- [x] 8.4 Wire Enter to open detail pane for selected run
- [x] 8.5 Wire Escape to close detail pane (guard: skip if overlay is open)
- [x] 8.6 Wire q to close detail and deselect
- [x] 8.7 Wire ⌘K / Ctrl+K to open command palette (bypass input focus guard)
- [x] 8.8 Wire n to open create run dialog
- [x] 8.9 Wire 1/2/3/4 to set filter shortcuts (All/Active/Done/Failed)
- [x] 8.10 Wire Tab to cycle detail tabs when detail pane is open
- [x] 8.11 Create `web/src/components/KeyboardHintBar.tsx` — fixed bottom bar showing context-appropriate shortcuts
- [x] 8.12 Add dismissible toggle for hint bar, persist visibility in localStorage

## 9. Detail Pane

- [x] 9.1 Create `web/src/components/DetailPane.tsx` wrapper with tab bar: Info, Logs, Files, Shell, Traces
- [x] 9.2 Implement Info tab: run ID, full prompt, repo, phase, created/updated timestamps
- [x] 9.3 Wire Logs tab to existing LogViewer component
- [x] 9.4 Wire Files tab to existing FileExplorer component
- [x] 9.5 Wire Shell tab to existing ShellTerminal component
- [x] 9.6 Wire Traces tab to existing TraceTimeline component
- [x] 9.7 Track active tab in component state, default to Info on run change

## 10. E2E Test Updates

- [x] 10.1 Update selectors: replace RunCard/RunFeed selectors with RunList table row selectors
- [x] 10.2 Update filter tests: replace FilterSidebar/FilterChip interactions with IconRail popover or keyboard shortcuts
- [x] 10.3 Add E2E test: open command palette with ⌘K, search for a run, select it with Enter
- [x] 10.4 Add E2E test: navigate list with j/k keys, open detail with Enter, close with Escape
- [x] 10.5 Add E2E test: resize split pane via drag handle, verify width persists after close/reopen
- [x] 10.6 Add E2E test: toggle theme via icon rail, verify `.dark` class on document root

## 11. Verification and Deploy

- [x] 11.1 Run `npx tsc --noEmit -p web/tsconfig.json` — zero type errors
- [x] 11.2 Run all E2E tests — all pass
- [x] 11.3 Visual check: 1080p viewport shows 15+ run rows without scrolling
- [x] 11.4 Visual check: light and dark themes render correctly with 10 tokens
- [x] 11.5 Visual check: no uppercase text anywhere in the UI
- [x] 11.6 Keyboard walkthrough: j/k/Enter/Escape/q/⌘K/n/1234/Tab all work correctly
- [ ] 11.7 Commit and push
