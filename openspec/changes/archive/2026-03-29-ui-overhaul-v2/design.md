## Context

The uncworks frontend is a React SPA (Vite + Tailwind + shadcn/ui) with a monospace font, dark/light themes, and a keyboard-first design ethos. The codebase has ~10 view files and ~20 components, all in `web/src/`. The UI was recently overhauled visually (rounded corners, spacing, typography) but the information architecture was not changed. The app has no persistent navigation, no global error feedback, and a tab-based RunDetailView that hides critical information.

This change is pure frontend — no backend, API, or proto changes needed.

## Goals / Non-Goals

**Goals:**
- Add persistent global navigation so users always know where they are
- Fix structural layout issues in RunDetailView (tabs → sidebar+panel)
- Deliver quick wins that improve daily usability immediately
- Maintain keyboard-first design while adding mouse discoverability
- Consistent error feedback across all async operations

**Non-Goals:**
- No backend changes
- No new API endpoints
- Not adding authentication or multi-user features
- Not adding test infrastructure (separate concern)

## Decisions

**1. Sidebar navigation over top tab bar**
A left sidebar (200px, collapsible to icon-only at ~50px) scales to future nav items, shows live counts as badges, and keeps vertical space for content. A top tab bar would work but is less scalable and conflicts with the existing header pattern in each view.
- Alternative considered: Convert current footer bar to a horizontal tab bar. Rejected — footer is too small and at the wrong end of the reading flow.

**2. RunDetailView: sidebar-nav + main + right slide-in panel**
Current 4-tab structure forces modal context switching. The redesign uses a 160px left sidebar for Logs/Traces/Files/Shell navigation, a main content area, and a right detail panel that slides in when a trace span is selected. This enables seeing the activity feed while a span is selected.
- Alternative: Split-pane with resizable divider. Rejected — adds complexity; slide-in panel is simpler and sufficient.
- The existing tab buttons become sidebar nav items (same keyboard shortcuts 1-4 preserved).

**3. HITL modal over footer bar**
The current yellow footer bar for "waiting for input" is missed when a user is on another tab. A modal guarantees attention. If dismissed, it minimizes to a persistent amber badge in the header. The modal is a standard shadcn Dialog.

**4. Phased delivery (tier 1 → 2 → 3)**
Tier 1 changes are independent one-liners that can ship immediately. Tier 2 includes the sidebar (which restructures Layout.tsx) and should land as one PR. Tier 3 items (RunDetail layout, trace search, DAG viz) are each independent and can ship separately after tier 2.

**5. Cron parsing: cronstrue library**
`cronstrue` (npm) converts cron expressions to human-readable strings with no backend dependency. Lightweight (~10kB). Used inline in ScheduleListView for tooltip + detail view.

**6. Unified project field in NewRunView**
The `projectRef` (CRD) dropdown and classification `project` text input will be merged into a single `<Select>` that offers CRD projects as options plus a "custom..." option that reveals a text input. This matches the mental model of "one project concept" and removes the confusing dual-field layout.

**7. Failure diagnosis panel**
When `run.status.phase === "failed"`, show a collapsible panel directly below the header. It reads the first failing span from the trace data (already loaded) and surfaces: failed tool name, error message, elapsed time. No new API calls needed — data already exists in the activity feed and trace.

**8. ChainRunDetail DAG**
Use `react-flow` (already a popular dep in this space) for the DAG visualization. Each node = a chain step, edges = dependencies. Color by phase. Show duration inside node. Fallback to existing text arrows if react-flow fails to load.
- Alternative: Raw SVG + D3. Rejected — higher implementation cost for same result.

## Risks / Trade-offs

- **Sidebar adds 200px to horizontal layout** → Mitigated by collapsible mode (icon-only at 50px); users on narrow screens can collapse. The existing `h-screen w-screen overflow-hidden` layout handles this cleanly.
- **RunDetailView refactor is large** → Tier 3 item; can be done in isolation after sidebar lands. Existing tab structure preserved as fallback.
- **react-flow bundle size** → ~150kB gzipped. Only loaded on ChainRunDetailView (lazy import). Acceptable given existing ~340kB chunk.
- **Merging project fields may break existing URLs** → `?project=` query param preserved; `?projectRef=` added. Both work. No breaking change.

## Migration Plan

1. **Tier 1**: Ship as single PR — independent changes, all in existing files
2. **Tier 2**:
   - Add `GlobalNav.tsx` + restructure `Layout.tsx` first (sidebar)
   - Then layer in per-view improvements (filter bar, column reorder, HITL modal, etc.)
   - Add `ScheduleDetailView.tsx` as new route
3. **Tier 3**: Each item ships as its own PR:
   - RunDetailView layout refactor
   - TraceTimeline search/filter
   - ChainRunDetail DAG
   - NewRunView tabs
   - ChainList split

Rollback: all changes are frontend-only. Any PR can be reverted without data migration.

## Open Questions

- Should the sidebar be collapsed by default on first visit? (Recommended: expanded on desktop, collapsed on narrow viewports)
- For the unified project field — should "custom project label" (classification only) be visually distinguished from CRD projects? (Recommended: yes, with muted styling)
- Should react-flow be added as a dependency or built with raw SVG? (Recommended: react-flow for maintainability)
