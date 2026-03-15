## Context

The AOT web UI was assembled by 15 parallel agents across 6 iterative changes. The result: a card feed with 3 runs filling a 27" screen, a 280px sidebar for 4 filter buttons, filter chips nobody asked for, and a design system with 40+ CSS tokens doing the work of 10. Engineers who live in terminals close the tab.

This change deletes the current UI and rebuilds it as one coherent tool designed for engineers who care about ergonomics — modeled after Linear, k9s, and lazygit.

## Goals / Non-Goals

**Goals:**
- Dense information display: 15-20 runs visible without scrolling on 1080p
- Keyboard-first interaction: every action reachable without a mouse
- Command palette as primary navigation surface
- Split-pane layout where list and detail coexist
- Two clean themes with 10 CSS tokens, not 40
- Zero decorative chrome

**Non-Goals:**
- Mobile responsiveness — this is a desktop engineering tool
- Drag-and-drop run reordering
- Multi-select bulk actions
- Persistent user preferences beyond theme and pane width

## Decisions

### 1. Dense list via HTML table

Use a real `<table>` element with `table-layout: fixed`, not div soup with flexbox. Columns: status dot (20px), ID (mono, 100px), prompt (flex, truncated), repo (120px), phase (80px), time (60px). Rows are 32px tall. Alternating backgrounds: `bg-transparent` / `bg-muted/5`. Selected row: `bg-accent/10` with a 2px left accent border.

**Why table?** Semantic HTML. Screen readers understand it. Column alignment is free. `table-layout: fixed` prevents layout thrash. Div-based grids need manual width sync between header and body — tables don't.

### 2. Command palette via Radix Dialog

Radix Dialog renders the palette as a portal on top of everything. Input at top, results below. Fuzzy matching via simple `String.includes()` on lowercased terms — no library needed for this scale (dozens of runs, not thousands). Three result types: runs (icon + name), commands (icon + label), filters (icon + label). Enter selects top result, Escape closes, arrow keys navigate.

**Why not cmdk?** It's 8KB for something we can build in 80 lines. Our search space is small enough that `includes()` is instant.

### 3. Split pane via CSS grid

The main layout is a CSS grid: `grid-template-columns: 1fr 0` when detail is closed, `grid-template-columns: 1fr 1fr` when open. CSS transition on column change gives a smooth slide. The detail pane reuses existing tab content (LogViewer, FileExplorer, ShellTerminal, TraceTimeline) — no rewrite needed for those. A drag handle between panes tracks `mousedown` → `mousemove` → `mouseup` to resize. Last width stored in `localStorage`.

**Why not a splitter library?** The mousedown/mousemove pattern is ~30 lines. Libraries add bundle weight and opinions about accessibility we don't need here.

### 4. Icon rail (48px fixed)

A fixed 48px-wide vertical rail on the left. Three icons stacked: filter funnel (opens a small popover with status radio buttons), plus sign (opens create run dialog), sun/moon (toggles theme). Tooltips on hover. Everything else lives in the command palette.

**Why 48px not 40px?** 40px is tight for touch targets and icon padding. 48px gives 12px padding around 24px icons — comfortable without wasting space.

### 5. Unified keyboard system

A single `useKeyboard` hook handles all keyboard shortcuts via event delegation on `document`. One map of key combinations to action callbacks. A guard function `isInputFocused()` checks if the active element is an input, textarea, or contenteditable — if so, all navigation shortcuts are suppressed (only ⌘K and Escape pass through). No separate hooks per feature.

**Why one hook?** Multiple `addEventListener('keydown')` handlers create ordering bugs, duplicate handling, and make it impossible to reason about what key does what. One handler, one map, one guard.

### 6. Token simplification

Strip the CSS custom property system to exactly 10 tokens: `--color-bg`, `--color-fg`, `--color-border`, `--color-muted`, `--color-accent`, `--color-success`, `--color-active`, `--color-warning`, `--color-error`, `--color-neutral`. Two scopes: `:root` for light theme, `.dark` for dark theme. Delete all accumulated token cruft from prior changes.

**Why 10?** Background, foreground, border, muted text — that's 4 for layout. Accent for interactive elements — that's 1. Five semantic status colors (success, active/running, warning/pending, error/failed, neutral/cancelled) — that's 5. Total: 10. Everything else was decoration.

### 7. Typography

IoskeleyMono everywhere. Normal case — no `text-transform: uppercase`. The monospace font IS the brand identity. Forcing uppercase on a monospace font fights the typeface's design. Font sizes: 13px for table body, 12px for metadata, 11px for timestamps. Line heights: 1.4 for readability in the dense table.

## Risks / Trade-offs

- **[Risk] Deleting all existing components** — No incremental migration path. If the rebuild stalls, we have neither old nor new UI. Mitigation: the old components stay in git history; we can revert if needed.
- **[Risk] Table semantics for interactive list** — Tables have complex ARIA requirements for interactive use. Mitigation: add `role="grid"` with `aria-activedescendant` for keyboard navigation rather than fighting table semantics.
- **[Risk] localStorage for pane width** — Can cause layout jump on load if stored width is stale. Mitigation: validate stored width is within [200px, 80vw] range; fall back to 50% if not.
- **[Trade-off] No fuzzy matching library** — `includes()` won't handle typos or fuzzy ordering. Acceptable because the search space is small and exact substring matching covers 95% of use cases.
- **[Trade-off] Single keyboard hook** — Makes the hook file grow as features are added. Acceptable because a single 80-line map is easier to audit than 8 scattered event listeners.
