## Context

The UI was migrated to MU-TH-UR 6000 (Radix UI, IoskeleyMono, CRT effects). The aesthetic is distinctive but the layout is unchanged — a table of runs with a side panel. The color system is mono-amber, making status scanning impossible. No light mode. The sidebar has navigation categories instead of filters.

Developers use GitHub, Linear, Vercel daily. Those tools use card feeds, semantic colors, filter chips, and progressive disclosure. AOT should feel familiar while keeping MU-TH-UR identity as flavor.

## Goals / Non-Goals

**Goals:**
- Someone who's never used AOT understands what's happening in 5 seconds
- Status instantly scannable via semantic color (green/blue/amber/red)
- Dark and light mode with toggle
- Card-based work feed replacing data table
- Sidebar = filters (chips), not navigation categories
- Progressive disclosure: feed → card → tabs → rich content
- Keyboard navigation (j/k, /, Escape, Enter)

**Non-Goals:**
- Mobile-first responsive design
- Real-time collaborative features
- Complete Storybook coverage for all new components

## Decisions

### 1. Semantic color tokens — dual-theme, intent-based

**Decision**: Define semantic tokens that map to different values in dark/light but always convey the same meaning:

```
Success (green): succeeded, healthy
Active (blue): running, in-progress — with pulse animation
Warning (amber): pending, attention needed
Error (red): failed, broken
Neutral (gray): cancelled, idle
```

Status colors are identical intent in both themes — adjusted for contrast.

**Rationale**: Von Restorff Effect — distinctive items are remembered. If running=blue and failed=red, you scan 20 runs in under a second.

### 2. Card feed instead of data table

**Decision**: Replace AgentRunTable with RunFeed rendering RunCard components. Each card: status dot (colored), run name (bold), repo (muted), prompt preview, time ago.

**Rationale**: Jakob's Law — GitHub PRs, Linear issues, Vercel deployments all use card feeds. Cards satisfy Law of Common Region — one run's info is visually grouped.

### 3. Sidebar = filter chips, not navigation

**Decision**: Sidebar contains only filter chip groups: Status (toggle), Repos (removable chips), Model (toggle), Workspace (chips), Actions (+ New Run, theme toggle).

No "Repositories view" or "Events view". Repos are filter chips, events are inline on cards.

**Rationale**: Hick's Law — fewer navigation choices = faster decisions. One view (feed) with filters.

### 4. Detail as full-width panel

**Decision**: Clicking a card opens full-width detail replacing the feed. Header with name/status/close, tab bar (Info/Logs/Files/Shell/Traces), full-width content.

**Rationale**: Side panel is too narrow for terminals and diffs. Progressive disclosure — detail only when asked.

### 5. Theme toggle with localStorage

**Decision**: Sun/moon icon toggles dark/light. MU-TH-UR effects (scanlines, glow) wrapped in `.dark &` — only in dark mode. IoskeleyMono stays in both.

### 6. Keyboard navigation

**Decision**: j/k moves selection, Enter opens detail, Escape closes, / focuses search. Only when no input focused.

**Rationale**: Linear and GitHub both support j/k. Jakob's Law.

## Risks / Trade-offs

**E2E test breakage** — All Playwright tests reference current layout. → Update tests in this change.

**Data density loss** — Cards show less per row than table. → Cards show the *right* data. Metadata is in detail view. Law of Prägnanz.

**MU-TH-UR diluted** — Light mode and semantic colors move from pure amber. → Font and dark-mode effects preserved. Identity is flavor, not constraint.
