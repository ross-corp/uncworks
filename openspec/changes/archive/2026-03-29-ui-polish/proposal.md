## Why

The web frontend has a set of consistent UX gaps — missing loading spinners on first paint, absent empty-state calls-to-action, no config-gate warning on the New Run form when an LLM key is missing, and minor spacing/color inconsistencies across all list views. These gaps make the app feel unfinished and can confuse first-time users who have no signal that configuration is incomplete or that a list is truly empty.

## What Changes

- **Empty states**: All five list views (RunListView, ProjectListView, TemplateListView, ChainListView, ScheduleListView) get properly structured empty states using the existing `Empty` / `EmptyHeader` / `EmptyDescription` components with an icon and a CTA button.
- **Loading states**: Replace bare "Loading…" text with the `Spinner` component centered in the scroll area across all list views.
- **Config gate on NewRunView**: Show an inline amber warning banner when `configStatus.hasLLMKey` is false, with a link to Settings, so the user knows the run will fail before submitting.
- **Spacing consistency**: Standardize view headers to `h-12 border-b flex items-center px-4 gap-2` (already mostly consistent), and body row padding to `px-4 py-2.5` across all list views. Fix `ChainListView` and `ScheduleListView` empty states that are plain text with no CTA.
- **Error handling**: `ProjectListView.fetchProjects` and `ScheduleListView.fetchData` silently swallow errors. Add `toast.error` on catch like the other views do.
- **TemplateListView**: Has duplicate fetch logic — an inline `useEffect` that duplicates `fetchData`. Remove the redundant one.
- **ChainRunListView**: Missing a "New run" or "Go to chains" CTA on empty state. Add one.
- **GlobalNav**: `configIncomplete` flag only checks `hasGitHubToken` alongside `hasLLMKey`. The dot correctly appears when either is missing — this is fine. No changes needed here.
- **SettingsView**: Scroll wrapper uses `flex-1 overflow-y-auto min-h-0` at the top-level div but the Layout `<main>` is `flex flex-col overflow-hidden`. The `min-h-0` is present so scrolling works. No structural fix needed — but the `px-8 py-8` padding is wider than all other views' `px-4`. Leave as-is since Settings is intentionally a form layout not a list.
- **NewRunView**: No LLM key warning shown. Add a dismissible amber `Alert` below the header when `!configStatus.hasLLMKey`.

## Capabilities

### New Capabilities
- `config-gate-banner`: Inline warning on NewRunView when LLM key is not configured
- `list-view-empty-states`: Structured empty states with icons and CTAs across all list views
- `list-view-loading-states`: Consistent spinner loading states across all list views

### Modified Capabilities
- None — no spec-level API or behavior changes, implementation polish only.

## Impact

- Files changed: `web/src/views/NewRunView.tsx`, `web/src/views/ProjectListView.tsx`, `web/src/views/TemplateListView.tsx`, `web/src/views/ChainListView.tsx`, `web/src/views/ScheduleListView.tsx`, `web/src/views/ChainRunListView.tsx`
- No new dependencies — uses existing `Spinner`, `Empty`, `EmptyHeader`, `EmptyDescription`, `EmptyContent`, `Alert` components already in the project
- No API changes
- No breaking changes
