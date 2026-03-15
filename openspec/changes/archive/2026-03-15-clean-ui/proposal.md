## Why

The current UI is a Frankenstein — a card feed bolted onto a design system bolted onto filter chips bolted onto an orchestration graph. It was assembled by 15 parallel agents, not designed by one person with taste. An engineer who lives in emacs or a terminal would look at this and close the tab.

Real engineering tools (Linear, k9s, lazygit, vim) share traits: dense information, keyboard-first interaction, zero decorative chrome, split-pane layouts where every pixel carries meaning. AOT's UI has filter chips nobody asked for, 3 cards filling a 27" screen, and a sidebar eating 280px for 4 filter buttons.

This change deletes the current UI and rebuilds it as one coherent thing designed for engineers who care about ergonomics.

## What Changes

### Delete and replace everything
- Delete: RunCard, RunFeed, FilterSidebar, FilterChip, FilterChipGroup, RunDetail
- Delete: all the accumulated UI cruft from 6 iterative changes
- Build: one clean interface from scratch

### Dense list
- One line per run. Status icon (colored dot), run ID, prompt (truncated), repo, phase text, time ago
- 15-20 runs visible on screen simultaneously
- No cards, no borders, no padding waste — just rows
- Alternating subtle row backgrounds for scannability

### Minimal navigation rail
- 40px icon rail on the left, not a 280px sidebar
- Icons: filter funnel, plus (new run), settings
- Click filter icon → small popover with status filters
- Everything else accessible via command palette

### Command palette (⌘K)
- The primary way to do anything: create run, filter, search, open settings, toggle theme
- Fuzzy search over run names, prompts, repos, commands
- Like Linear/VS Code — this IS the navigation

### Split-pane detail
- Select a run → right pane slides open (50% width)
- List stays visible on the left, detail on the right
- Detail has tabs: Info | Logs | Files | Shell | Traces
- Resizable divider between list and detail
- Close detail: Escape or click the selected run again

### Keyboard-first
- j/k: move selection
- Enter: open detail pane
- Escape: close detail
- ⌘K or /: command palette
- n: new run
- 1/2/3/4: filter all/active/done/failed
- Tab: cycle detail tabs
- q: close everything

### Two clean themes
- Light: white bg, gray-900 text, colored status indicators. Default.
- Dark: gray-950 bg, gray-100 text, same status colors. IoskeleyMono glow effects.
- Toggle via command palette or header icon
- No CRT scanlines in light mode. Subtle in dark.

### Semantic status, nothing else
- Green dot: succeeded
- Blue dot (pulsing): running
- Amber dot: pending
- Red dot: failed
- Gray dot: cancelled
- No other colors for decoration. Period.

## Capabilities

### New Capabilities
- `dense-list`: One-line-per-run list replacing card feed — 15-20 visible, alternating rows, status dots
- `command-palette`: ⌘K fuzzy search over runs, commands, and filters — the primary interaction surface
- `split-pane-detail`: Resizable right pane for run detail, list stays visible
- `icon-rail`: 40px navigation rail replacing 280px sidebar
- `keyboard-system`: Complete keyboard-first interaction model

### Modified Capabilities
<!-- None — this is a clean slate -->

## Impact

- **Delete**: RunCard, RunFeed, FilterSidebar, FilterChip, FilterChipGroup, current App.tsx layout
- **Create**: RunList, CommandPalette, SplitPane, IconRail, DetailPane, KeyboardManager
- **Modify**: RunDetail (simplify), StatusBadge (simplify to dots), ThemeToggle (move to command palette)
- **CSS**: Simplify token system, remove unused tokens, tighten spacing
- **Tests**: Update all E2E tests for new selectors
