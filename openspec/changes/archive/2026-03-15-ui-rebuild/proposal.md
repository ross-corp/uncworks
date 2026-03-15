## Why

The current UI is tool-centric — it presents a table of agent runs and asks the user to manage them. It should be task-centric — showing the user what's happening with their work and letting them drill into detail progressively. The sidebar has navigation categories (Repositories, Events) that should be filters. Status is communicated through text badges instead of semantic color. Everything is mono-amber, making it impossible to scan status at a glance. There's no light mode. A developer who's never seen the tool should be able to understand what's happening within seconds — that requires semantic color, familiar patterns (card feeds like GitHub/Linear/Vercel), and progressive disclosure.

## What Changes

### Layout: Card Feed, Not Table
- **Replace the AgentRunTable** with a card-based work feed. Each card shows: name, status (color-coded dot/icon), repo, model tier, time ago, first line of prompt. Cards are scannable — you see 10 runs and instantly know which are green (done), blue (running), red (failed).
- **Sidebar becomes filters**, not categories. Status filter (toggle chips: All/Active/Done/Failed), repo filter (removable chips), model filter (chips), workspace filter (chip). No "Repositories" or "Events" as navigation destinations — repos are filter chips derived from runs.
- **Detail view expands** from a cramped side panel to a full-width slide-up panel or dedicated view. Progressive disclosure: summary → tabs (Logs/Files/Shell/Traces/Diff) → rich content.

### Semantic Color System
- **Green** = succeeded/healthy (things are good)
- **Blue** = running/active (something's busy — pulsing)
- **Amber** = warning/pending (pay attention)
- **Red** = failed/error (something broke)
- **Gray** = cancelled/neutral (nothing happening)
- **Accent** = interactive elements (buttons, links — click me)
- **Muted** = metadata/context (not the focus)
- Colors carry meaning, never decoration. Every colored element communicates state. Status colors are identical in dark and light mode.

### Dark/Light Mode
- Dark mode: black backgrounds, light foreground (default, matches MU-TH-UR heritage)
- Light mode: white backgrounds, dark foreground
- Toggle in header. Persists to localStorage.
- MU-TH-UR effects (scanlines, glow) only active in dark mode.
- IoskeleyMono font stays in both modes.

### Progressive Disclosure (Hick's Law)
- **Level 1**: Work feed — scan status of all runs at a glance
- **Level 2**: Expanded card — click a run, see summary with metadata
- **Level 3**: Detail tabs — Logs, Files, Shell, Traces, Diff
- **Level 4**: Rich content — full terminal, file tree, trace timeline

### Familiar Patterns (Jakob's Law)
- Card feed like GitHub PR list / Linear issues
- Filter chips like Gmail / Vercel
- Detail panel like GitHub PR detail view
- Keyboard shortcuts (/ for search, j/k for navigation)

## Capabilities

### New Capabilities
- `card-feed-layout`: Card-based work feed replacing the table — status dots, repo chips, time ago, prompt preview
- `semantic-color-system`: Intent-based color tokens (success/active/warning/error/neutral) that work in both themes
- `dark-light-toggle`: Theme toggle with localStorage persistence, MU-TH-UR effects only in dark mode
- `filter-sidebar`: Chip-based filter sidebar replacing category navigation — status, repo, model, workspace as toggleable filter chips
- `detail-expansion`: Full-width detail view with progressive disclosure — summary → tabs → rich content
- `keyboard-navigation`: j/k navigation, / for search, Escape to close, Enter to open

### Modified Capabilities
<!-- No existing spec-level requirements change -->

## Impact

- **Complete rewrite** of App.tsx, Layout, Sidebar, AgentRunTable (→ RunFeed), AgentRunDetailPanel (→ RunDetail)
- **New components**: RunCard, FilterChip, ThemeToggle, RunFeed, RunDetail
- **CSS overhaul**: semantic color tokens, dark/light theme variables, conditional MU-TH-UR effects
- **index.css**: dual-theme token system
- **All existing components** updated for semantic colors
- **E2E tests**: updated for new selectors and layout
- **Storybook**: stories for all new components
