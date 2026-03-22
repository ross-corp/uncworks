## Context

The UNCWORKS frontend has 5 views, 17 custom components, and 25 shadcn UI components already installed but underutilized. Monaco editor (`@monaco-editor/react` v4.7.0) is used for file preview but hardcoded to `vs-dark`. The theme system (`useThemeNew.ts`) supports light/dark/system via CSS variables and a toggle in the Layout footer. xterm.js powers the terminal with no theme integration. The run list shows name, status, stage, model, and age columns. Cost is calculated per-span in TraceTimeline but never aggregated to the run level. The CRD has no `archived` field. PVCs are cleaned up by `cleanupExpiredRuns()` only after 7 days.

## Goals / Non-Goals

**Goals:**
- Every form element uses shadcn components (no raw HTML selects/inputs)
- Monaco editors for prompt/spec input with markdown highlighting
- Theme-aware Monaco and xterm across all views
- Archive runs with PVC cleanup and mass-select
- Run list enriched with cost, diff stats, PR link, dual model display
- Dual model selectors for progressive runs
- Fix stuck "Loading activity..." in run detail logs

**Non-Goals:**
- Custom theme creation (just light/dark/system)
- Side-by-side diff viewer (keep unified patches)
- Real-time cost tracking for running jobs (only final cost for completed)
- Multiple repos per run in the UI (backend supports it, UI stays single-repo for now)

## Decisions

### 1. Archive via CRD status field

Add `archived: bool` to `AgentRunStatus` in the CRD. The apiserver exposes `ArchiveRun` and `BulkArchiveRuns` RPCs. On archive, the controller deletes the PVC for that run.

*Alternative*: Separate archive annotation. Rejected because status fields are already the mechanism for run state transitions, and annotations aren't surfaced in list queries without custom indexers.

### 2. Cost aggregation on the server side

Add a `totalCost` field to the run status response, computed lazily by the apiserver when returning run lists. The server reads `spans.jsonl`, sums per-span costs using the same pricing table as TraceTimeline, and caches in the run status. This avoids the frontend needing to fetch all spans for every run in the list.

*Alternative*: Frontend fetches traces for each visible run. Rejected because it would create N+1 API calls on every list render.

### 3. Diff stats aggregation

Similarly, aggregate `totalAdditions` and `totalDeletions` from trace spans into the run status response. The apiserver computes this when spans are available.

### 4. Monaco for prompt/spec editors

Wrap Monaco in a reusable `MarkdownEditor` component that:
- Uses `@monaco-editor/react` (already installed)
- Sets `language: "markdown"`, `wordWrap: "on"`, `minimap: { enabled: false }`
- Reads theme from `useThemeNew()` hook: `"vs"` for light, `"vs-dark"` for dark
- Configures height to fill available space with `min-height`

### 5. xterm.js theme sync

Create a `terminalTheme` object derived from CSS custom properties. The `ShellTerminalInner` component reads the current theme mode and applies an `ITheme` object to the Terminal constructor. Light mode uses a white background with dark text; dark mode uses the current dark palette.

### 6. Dual model config

Add `manageModelTier` and `implementModelTier` to `AgentRunSpec` (CRD + proto). The New Run view shows two model selectors when orchestration mode is "Progressive". The workflow reads `implementModelTier` (falling back to `modelTier` if unset) and passes it to the sidecar via env var `PI_MODEL` per stage.

### 7. shadcn component migration

Replace incrementally:
- Raw `<select>` → shadcn `Select` (model picker, orchestration picker)
- Raw `<button>` → shadcn `Button` (already partial)
- Tab bar in RunDetailView → shadcn `Tabs`
- Status indicators → shadcn `Badge`
- Mass-select checkboxes → shadcn `Checkbox`

### 8. Mass select UX

Enter selection mode via `x` key or a "Select" button. Checkboxes appear on each row. A floating action bar shows "N selected — Archive | Cancel". `Shift+click` selects a range. `Ctrl+A` selects all visible.

### 9. Run list column layout

New grid: `[checkbox] [name] [status] [stage] [models] [cost] [+/-] [PR] [age]`

The models column shows `manage/implement` when different, or just the single model. Cost shows `$0.12` or `—`. Diff shows `+42/-5` with semantic colors. PR shows a linked badge icon.

## Risks / Trade-offs

- **[Monaco bundle size]** — Monaco adds ~2MB to the JS bundle. Mitigated by lazy-loading the editor component (already the pattern used in FilePreview).
- **[Cost accuracy]** — Server-side cost calculation uses a static pricing table that may drift from actual provider prices. Acceptable for estimates.
- **[Archive PVC deletion is irreversible]** — Once archived, workspace data is gone. Mitigated by confirmation dialog and the fact that archived runs retain their trace/log data (stored separately).
- **[Mass select complexity]** — Range selection (shift+click) and select-all add interaction complexity. Start with simple checkboxes; range selection can be a follow-up.

## Open Questions

- Should archived runs be filterable by status (show only archived-succeeded vs archived-failed)?
- Should the theme picker offer accent color customization beyond light/dark?
