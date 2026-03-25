## 1. Tier 1 — Quick wins (independent, ship fast)

- [x] 1.1 Replace all silent `catch(() => {})` blocks across views with `toast.error(...)` calls — audit NewRunView, RunListView, ProjectDetailView, ActivityFeed, RunDetailView
- [x] 1.2 Wire `Ctrl+Enter` / `Cmd+Enter` keyboard shortcut in NewRunView to submit form when `canRun` is true; show "⌘↵" hint near Run button
- [x] 1.3 Fix branch field width in NewRunView repo inputs — change `w-20` (80px) to `min-w-0 flex-1` so branch names aren't truncated
- [x] 1.4 Add distinct "waiting for input" visual state in RunStatusBadge — amber color, pulsing animation, separate from blue "running" badge
- [x] 1.5 Add "waiting" to RunListView status filter tabs alongside all/running/failed/succeeded
- [x] 1.6 Add jump-to-latest button in ActivityFeed — appears when scrolled >100px from bottom, smooth scroll on click
- [x] 1.7 Install `cronstrue` npm package (`npm install cronstrue --prefix web`) and add human-readable cron tooltip to ScheduleListView using `cronstrue.toString(cronExpr)`
- [x] 1.8 Change next-scheduled-time display in ScheduleListView from `toLocaleString()` to relative time (use existing `formatAge` or add `formatRelative`)
- [x] 1.9 Make "Improve with AI" button visible — increase to `h-8 text-sm`, change to `variant="outline"`, add ✨ icon prefix
- [x] 1.10 Add error toast to "Improve with AI" failure path in NewRunView (currently silent catch)
- [x] 1.11 Make feature group header chevron larger (text-sm, foreground color not muted) and add full-row hover background in RunListView
- [x] 1.12 Add bulk archive feedback — show "Archiving N runs..." toast on start, "N runs archived" on completion in RunListView
- [x] 1.13 Add "Clear all filters" button to RunListView that resets statusFilter, activeProject, and filter text in one click

## 2. Tier 1 — NewRunView project field consolidation

- [x] 2.1 Merge `projectRef` CRD dropdown and classification `project` text input into a single unified project field in NewRunView
- [x] 2.2 CRD projects shown as primary options in the selector; add "Custom label..." option that reveals a text input for free-form classification label
- [x] 2.3 Auto-fill repos/model/orchestration when a CRD project is selected (existing `handleProjectRefChange` logic)
- [x] 2.4 Update model tier select options to show trade-off labels ("Fast & cheap", "Best quality", "Local / offline") with raw model ID as secondary text

## 3. Tier 2 — Global navigation sidebar

- [x] 3.1 Create `web/src/components/GlobalNav.tsx` — persistent left sidebar with Runs/Projects/Chains/Schedules nav items, active route highlighting, live count badges
- [x] 3.2 Add collapse/expand toggle to sidebar; persist state in localStorage under key `nav-collapsed`
- [x] 3.3 Restructure `web/src/views/Layout.tsx` to use sidebar + main content layout; move theme toggle into sidebar footer
- [x] 3.4 Remove Projects/Chains/Schedules nav buttons from RunListView header (now in sidebar)
- [x] 3.5 Add breadcrumb component to RunDetailView, ProjectDetailView, ChainRunDetailView — clickable segments navigate to parent

## 4. Tier 2 — RunListView filter discoverability

- [x] 4.1 Add persistent labeled filter bar above run list — field selector (Name/State/Stage/Model) + search input always visible
- [x] 4.2 Preserve vim-key shortcuts (/, ?, ', ") as fast-path to activate the corresponding field
- [x] 4.3 Show active filter state visually (highlighted field chip + value); clear-all button clears field + text

## 5. Tier 2 — RunListView row column reorder

- [x] 5.1 Reorder RunRow columns: status badge → name → external-status (PR + CI unified as chips) → cost → diff stats → model → age
- [x] 5.2 Combine PR link and CI status into a single `ExternalStatus` mini-component showing chips side-by-side

## 6. Tier 2 — RunDetailView improvements

- [x] 6.1 Add horizontal phase/stage step indicator to RunDetailView header — Planning → Executing → Verifying with color coding (blue=active, green=done, red=failed)
- [x] 6.2 Add live elapsed time counter to RunDetailView header (increments every second while run is active)
- [x] 6.3 Create `FailureDiagnosisPanel.tsx` — collapsible panel shown when phase=failed; surfaces failing stage, error message, "View in Traces" button, Retry/Edit+Retry/Archive actions
- [x] 6.4 Implement "View in Traces" in FailureDiagnosisPanel — switches to Traces tab and auto-scrolls to first failed span
- [x] 6.5 Add "Edit & Retry" to FailureDiagnosisPanel — navigates to /new with run params pre-filled via query string

## 7. Tier 2 — HITL modal

- [x] 7.1 Create `HitlModal.tsx` using shadcn Dialog — shows agent prompt text, auto-focused input, Send/Cancel buttons
- [x] 7.2 Auto-open HitlModal when run phase transitions to waiting_for_input in RunDetailView
- [x] 7.3 Add persistent amber badge to RunDetailView header when in waiting_for_input phase; clicking badge re-opens modal
- [x] 7.4 Show confirmation toast "Input sent · Resuming run" on modal submit; clear badge on success

## 8. Tier 2 — ProjectDetailView Runs tab

- [x] 8.1 Add shadcn Tabs component to ProjectDetailView replacing Badge-based onClick toggles
- [x] 8.2 Add "Runs" tab content — fetch and display runs where `spec.project === projectName` in a run list
- [x] 8.3 Add empty state for Runs tab with "+ New Run" link to `/new?project=:name`

## 9. Tier 2 — Schedule detail view

- [x] 9.1 Create `web/src/views/ScheduleDetailView.tsx` — shows schedule name, human-readable cron, suspend/resume, execution history table
- [x] 9.2 Add `/schedules/:name` route in AppNew.tsx
- [x] 9.3 Add inline cron editor with frequency/time selectors; save calls API to update cron expression
- [x] 9.4 Fetch and display last 10 executions in a table (date, status badge, duration, link to run)

## 10. Tier 3 — RunDetailView layout overhaul

- [x] 10.1 Redesign RunDetailView to sidebar-nav (Logs/Traces/Files/Shell) + main content area + right slide-in detail panel
- [x] 10.2 Right detail panel slides in when trace span is selected — main content (waterfall) stays visible
- [x] 10.3 Preserve 1/2/3/4 keyboard shortcuts for switching main content; Escape closes right panel
- [x] 10.4 Default main content is ActivityFeed (Logs); preserve existing tab-1 behavior

## 11. Tier 3 — TraceTimeline search and expand/collapse

- [x] 11.1 Add filter bar to TraceTimeline header — text search input + filter chips (Failed, Bash, Write, LLM)
- [x] 11.2 Implement span text search — filter waterfall rows by span name match; show parent spans collapsed if they have matching children
- [x] 11.3 Add "Expand All" / "Collapse All" buttons to TraceTimeline header
- [x] 11.4 Show "[N hidden]" badge on collapsed span groups
- [x] 11.5 Improve span hover state — full row background highlight + underline on span name + cursor:pointer
- [x] 11.6 Persist selected span highlight (accent background) until another click or Escape

## 12. Tier 3 — ChainRunDetail visual DAG

- [x] 12.1 Add `react-flow` dependency to web package
- [x] 12.2 Create `ChainDagViz.tsx` using react-flow — nodes for each step, directed edges for dependencies
- [x] 12.3 Color nodes by phase (blue=running, green=succeeded, red=failed, gray=pending, yellow=skipped)
- [x] 12.4 Show elapsed duration inside completed nodes
- [x] 12.5 Wire node click to navigate to /run/:id for that step
- [x] 12.6 Add Timeline tab to ChainRunDetailView showing steps as horizontal bars on a time axis
- [x] 12.7 Fallback to existing text-based step list if react-flow fails to render

## 13. Tier 3 — NewRunView progressive disclosure

- [x] 13.1 Add "Core" and "Config" tabs to NewRunView — Core: prompt+repos; Config: model/TTL/orchestration mode/classification
- [x] 13.2 Move prompt/spec mode toggle to the NewRunView header next to the title for discoverability
- [x] 13.3 Split ChainListView into separate routes: `/chains` (chain definitions) and `/chainruns` (chain run executions)

## 14. Verification

- [ ] 14.1 Verify all error toasts fire correctly by testing failed fetches in dev (block network in devtools)
- [ ] 14.2 Verify sidebar nav shows correct active state on all routes
- [ ] 14.3 Verify waiting_for_input amber badge appears in RunList and RunDetail
- [ ] 14.4 Verify HITL modal opens automatically and sends input correctly
- [ ] 14.5 Verify FailureDiagnosisPanel surfaces the correct error and "View in Traces" jumps to the right span
- [ ] 14.6 Verify ScheduleDetailView cron editor saves and updates correctly
- [ ] 14.7 Verify TraceTimeline filter and expand/collapse work on a run with 50+ spans
- [ ] 14.8 Verify ChainDagViz renders correctly for a 3-step linear chain and a diamond DAG
