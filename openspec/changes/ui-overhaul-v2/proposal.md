## Why

The uncworks UI was audited across all 4 surfaces (RunList, RunDetail, NewRun, Layout/Projects/Chains/Schedules) and found to have three structural problems: no global navigation (every view reinvents its own nav buttons), a tab-based RunDetailView that hides critical information behind modals, and silent failures throughout. Beyond structure, dozens of high-value UX improvements were identified across discoverability, visual hierarchy, and interaction quality. This work addresses all three tiers: quick wins, medium-effort structural improvements, and high-effort layout overhauls.

## What Changes

**Tier 1 — Quick wins**
- Add error toasts to all silent `catch(() => {})` blocks across all views and components
- Wire Ctrl+Enter keyboard shortcut in NewRunView (currently advertised but not implemented)
- Fix repository branch field width (hardcoded 80px truncates real branch names)
- Add distinct "waiting for input" visual state in RunListView (amber, pulsing — not same as running)
- Add jump-to-latest button in ActivityFeed when user has scrolled up
- Add human-readable cron translation in ScheduleListView (e.g., "Daily at 9am" alongside "0 9 * * *")
- Make "Improve with AI" button visible (increase size, add icon, currently 11px ghost text)
- Consolidate the two project fields in NewRunView (projectRef CRD dropdown + classification label text input → single unified control)

**Tier 2 — Medium effort**
- Add persistent global navigation sidebar (Runs / Projects / Chains / Schedules with live counts)
- Replace vim-key-only filter discovery in RunListView with persistent labeled filter bar (keep vim keys as shortcuts)
- Reorder RunListView row columns for scanning: status → name → PR+CI (unified chips) → cost → diff → model → age
- Add failure diagnosis panel in RunDetailView (when failed: prominent error, jump to span, retry/edit options)
- Improve ActivityFeed (scroll position indicator, jump-to-latest, per-entry search)
- Add horizontal phase/stage step indicator in RunDetailView header (Planning → Executing → Verifying)
- Overhaul "waiting for input" UX — modal + header badge instead of easy-to-miss yellow footer bar
- Add Runs tab to ProjectDetailView (runs filtered to this project)
- Add ScheduleList detail view with cron editor (human-readable, last executions, edit form)

**Tier 3 — High effort**
- Overhaul RunDetailView layout: 4 tabs → persistent sidebar nav + main content + right slide-in detail panel
- Add span search/filter to TraceTimeline (filter by type/status/tool, text search, collapse persistence)
- Add visual DAG with timing to ChainRunDetailView (SVG edges, step durations, color by phase)
- Add progressive disclosure to NewRunView (Core tab: prompt+repos; Config tab: model/TTL/orchestration/classification)
- Split ChainListView into separate chain definitions view and chain runs view

## Capabilities

### New Capabilities
- `global-nav`: Persistent left sidebar navigation across all views
- `run-detail-layout`: Sidebar+main+detail-panel layout replacing tab-based RunDetailView
- `hitl-modal`: Modal-based human-in-the-loop input replacing footer bar
- `failure-diagnosis`: Failure diagnosis panel on RunDetailView when phase=failed
- `trace-search`: Span search and filter in TraceTimeline
- `chain-dag-viz`: Visual SVG DAG with timing in ChainRunDetailView
- `schedule-detail`: Schedule detail view with editable cron and execution history

### Modified Capabilities
- `ui-activity-feed`: Add jump-to-latest, scroll indicator, entry search
- `ui-views`: RunListView column reorder, filter discoverability, waiting-for-input state
- `model-selection-ui`: Consolidate dual project fields; clarify model tier descriptions
- `run-list-hierarchy`: Feature grouping header affordance improvements
- `project-management`: Add Runs tab to ProjectDetailView

## Impact

- `web/src/views/`: All view files touched
- `web/src/components/ActivityFeed.tsx`, `TraceTimeline.tsx`, `StageProgress.tsx`: Modified
- `web/src/views/Layout.tsx`: Major restructure (sidebar added)
- New components: `GlobalNav.tsx`, `HitlModal.tsx`, `FailureDiagnosisPanel.tsx`, `ChainDagViz.tsx`, `ScheduleDetailView.tsx`
- No backend changes required — all frontend
- No breaking API changes
