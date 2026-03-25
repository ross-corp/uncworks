## Why

The frontend has accumulated significant technical debt across type safety, component architecture, and network efficiency. Six views each duplicate the same polling boilerplate, GlobalNav hammers the backend with 6 full list fetches every 10 seconds just for badge counts, and a 1,442-line god component makes trace-related changes fragile and expensive. Addressing these now prevents further compounding as the UI grows.

## What Changes

- Extract a `usePoll(fn, intervalMs)` hook to replace 6 duplicated `setInterval` + cleanup patterns across RunListView, ProjectListView, ActivityFeed, TraceTimeline, RunDetailView, and GlobalNav
- Add a `GET /api/v1/counts` lightweight endpoint in the Go backend returning badge counts, replacing 6 full list fetches in GlobalNav every 10 seconds
- Replace 36 instances of `as unknown as { ... }` unsafe casts with properly typed API response interfaces, starting with `useClient.ts` lines 80–86
- Extract `useRunForm` hook from NewRunView to consolidate 32 `useState` calls into managed form state
- Improve error handler logging: 8+ silent catch blocks in NewRunView, GlobalNav, and useTraces should log error detail for debugging
- Split TraceTimeline (1,442 lines) into SpanWaterfall, SpanDetailPanel, and StageSummary sub-components *(lower priority)*

## Capabilities

### New Capabilities

- `poll-hook`: Reusable `usePoll(fn, intervalMs)` React hook with proper cancellation semantics
- `counts-api`: Lightweight `/api/v1/counts` backend endpoint returning per-entity badge counts
- `api-response-types`: Typed interfaces for all API response shapes, eliminating unsafe `unknown` casts
- `run-form-hook`: `useRunForm` hook encapsulating NewRunView form state (32 useState → 1 hook)

### Modified Capabilities

- `ui-activity-feed`: Polling implementation changes to use `usePoll` hook (no behavior change)
- `ui-views`: NewRunView form state management refactored via `useRunForm`
- `trace-detail-panel`: TraceTimeline split into sub-components (lower priority)

## Impact

- **Frontend**: `web/src/hooks/` (new hooks), `web/src/components/GlobalNav.tsx`, `web/src/components/TraceTimeline.tsx`, `web/src/views/NewRunView.tsx`, `web/src/hooks/useClient.ts`, all 6 polling views
- **Backend**: `internal/server/` — new route handler for `GET /api/v1/counts`
- **No breaking changes**: all refactors maintain existing external behavior
- **Dependencies**: no new npm packages or Go modules required
