## Context

The frontend was built iteratively and each feature that needed polling copied the same `useEffect` + `setInterval` + `cancelled` flag pattern. GlobalNav was wired last and inherited the same approach, but instead of receiving counts from a cheap endpoint, it fetches full lists from 6 endpoints every 10 seconds. Type safety was deferred by casting through `unknown` for fields that were added to the Go `RunStatus` struct after the TypeScript types were initially generated. NewRunView grew to 32 `useState` calls as options were added incrementally.

The backend uses `net/http` with a `ServeMux` and per-handler files (e.g., `internal/server/projects.go`). Routes are registered via `RegisterXHandlers(mux)` methods.

## Goals / Non-Goals

**Goals:**
- Ship a `usePoll` hook that all 6 polling sites use — eliminates duplicated cancellation logic
- Ship `GET /api/v1/counts` — reduces GlobalNav from ~60 KB of JSON per poll to <200 bytes
- Eliminate all `as unknown as { ... }` casts in `useClient.ts` with a typed `RunStatusFields` extension interface
- Consolidate NewRunView's 32 `useState` calls into a `useRunForm` hook
- Improve catch-block logging so errors are visible in the console for debugging

**Non-Goals:**
- TraceTimeline split is intentionally deferred — it is the riskiest refactor and provides no user-visible benefit in the short term
- No changes to the shared types package or proto definitions
- No new npm dependencies or Go modules

## Decisions

### D1: `usePoll` signature — callback-based, not query-based

`usePoll(fn: () => Promise<void>, intervalMs: number, deps?: DependencyList)` runs `fn` immediately on mount and after each interval, with a `cancelled` guard inside. Alternative: a suspense-compatible query hook (React Query style). Rejected — would require a new dependency and a larger migration. The simple callback form matches existing usage exactly and is a drop-in replacement.

### D2: `/api/v1/counts` lives in a new `counts.go` handler file

Consistent with the existing pattern where each resource domain has its own handler file. The handler reads counts from the same storage layer used by the list handlers. Returns a flat JSON object: `{ runs: N, activeRuns: N, projects: N, templates: N, chains: N, chainruns: N, schedules: N }`. The `activeRuns` field is the active-phase count GlobalNav currently computes client-side, so GlobalNav can drop all filtering logic.

Alternative: add a `?countOnly=true` query param to each list endpoint. Rejected — it would require changes to 6 existing handlers and adds request count overhead.

### D3: Typed extension interface, not regenerated types

The `RunStatus` type from the shared package omits fields added after initial generation (archived, totalCost, etc.). The fix is a local `ExtendedRunStatus` interface in `useClient.ts` that intersects the shared type with the missing fields. This is contained to one file and avoids touching the shared package.

Alternative: update the shared Go → TypeScript type generation. Correct long-term but out of scope here; left as a follow-up.

### D4: `useRunForm` returns a single state object + setter helpers

`useRunForm()` returns `{ form, set, reset }` where `form` is a typed object of all 32 fields. Individual `set.fieldName(value)` helpers are generated to keep call sites readable. Alternative: a single `dispatch` with action types (useReducer). More correct for complex forms but higher migration cost with no user benefit here.

### D5: Error logging — console.error alongside existing toast

The 8 silent catch blocks will each add `console.error('[ComponentName]', err)` before the existing toast or no-op. No change to UX. This is the minimal fix; structured error reporting is out of scope.

## Risks / Trade-offs

- **`usePoll` timing drift** → The hook resets the interval on each `fn` invocation. This is acceptable; the previous pattern had the same behavior.
- **`/api/v1/counts` consistency** → Counts may lag slightly behind list views since they are separate requests. Mitigation: this is badge-count UI only; consistency is not required.
- **`useRunForm` refactor breaks NewRunView** → The 32 `useState` → hook migration is mechanical but touches many lines. Mitigation: do it in a single commit with no behavioral changes, reviewed against the existing state shape before merging.
- **`ExtendedRunStatus` divergence** → The local interface can drift from the Go struct. Mitigation: add a TODO comment pointing to the shared type generation follow-up.

## Migration Plan

1. Land `usePoll` hook + migrate all 6 polling sites in one PR (no behavior change)
2. Land `/api/v1/counts` backend + update GlobalNav in one PR
3. Land `ExtendedRunStatus` typed interface + remove all `as unknown` casts in one PR
4. Land `useRunForm` + update NewRunView in one PR
5. Land catch-block logging improvements in one PR (trivial, can be merged any time)

Each step is independently deployable with no rollback risk. No database migrations, no API version bumps.

## Open Questions

- Should `usePoll` accept a `enabled: boolean` flag to pause polling (e.g., when tab is hidden)? Not needed for initial implementation but worth noting.
- Long-term: should counts be pushed via SSE rather than polled? Deferred.
