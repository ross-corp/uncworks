## 1. usePoll Hook

- [x] 1.1 Create `web/src/hooks/usePoll.ts` with signature `usePoll(fn: () => Promise<void>, intervalMs: number, deps?: DependencyList): void`
- [x] 1.2 Implement immediate invocation on mount, interval repeat, and `cancelled` guard inside the hook
- [x] 1.3 Migrate `RunListView.tsx` to use `usePoll`, removing inline `setInterval` + `cancelled` pattern
- [x] 1.4 Migrate `ProjectListView.tsx` to use `usePoll`
- [x] 1.5 Migrate `ActivityFeed.tsx` to use `usePoll`
- [x] 1.6 Migrate `TraceTimeline.tsx` to use `usePoll`
- [x] 1.7 Migrate `RunDetailView.tsx` to use `usePoll`
- [x] 1.8 Migrate `GlobalNav.tsx` to use `usePoll` (interval setup only; fetch logic updated in section 2)
- [x] 1.9 Verify no inline `setInterval` calls remain in the six migrated components

## 2. Counts API — Backend

- [x] 2.1 Create `internal/server/counts.go` with a `CountsHandler` struct and `RegisterCountsHandlers(mux)` method registering `GET /api/v1/counts`
- [x] 2.2 Implement the handler to read entity counts from the same storage layer used by the list handlers; return `{ runs, activeRuns, projects, templates, chains, chainruns, schedules }`
- [x] 2.3 Ensure `activeRuns` counts only runs with phase `running`, `pending`, or `waiting_for_input`
- [x] 2.4 Call `RegisterCountsHandlers` from the server setup (wherever other `RegisterXHandlers` calls are made)
- [x] 2.5 Write a basic test in `counts_test.go` verifying the response shape and empty-system zeros

## 3. Counts API — Frontend (GlobalNav)

- [x] 3.1 Update `GlobalNav.tsx` to call `GET /api/v1/counts` instead of the 6 parallel full-list fetches
- [x] 3.2 Remove the client-side `activeRuns` filtering logic; use `data.activeRuns` directly
- [x] 3.3 Update the `Counts` state type in `GlobalNav.tsx` to match the `CountsResponse` shape
- [x] 3.4 Add `console.error('[GlobalNav]', err)` to the silent catch block

## 4. API Response Types (ExtendedRunStatus)

- [x] 4.1 Define `ExtendedRunStatus` interface in `web/src/hooks/useClient.ts` extending the shared `RunStatus` with: `archived?: boolean`, `totalCost?: string`, `totalAdditions?: number`, `totalDeletions?: number`, `ciFixAttempts?: number`, `lastCIStatus?: string`, `parentPRUrl?: string`
- [x] 4.2 Replace the 7 `as unknown as { ... }` casts at lines 80–86 of `useClient.ts` with typed access via `ExtendedRunStatus`
- [x] 4.3 Audit all 36 files in `web/src/` for additional `as unknown as` patterns
- [x] 4.4 Resolve each remaining cast with a typed interface or typed assertion function with comment
- [x] 4.5 Confirm `grep -r "as unknown as" web/src/` returns no results

## 5. useRunForm Hook

- [x] 5.1 Create `web/src/hooks/useRunForm.ts` defining the `RunFormState` type covering all 32 form fields with their correct types and defaults
- [x] 5.2 Implement `useRunForm()` returning `{ form, set, reset }` with per-field setters
- [x] 5.3 Update `NewRunView.tsx` to call `useRunForm()` and remove all 32 individual `useState` form-field declarations
- [x] 5.4 Replace field references in JSX/handlers to use `form.fieldName` and `set.fieldName`
- [x] 5.5 Verify form behavior is identical: same defaults, same field update behavior, same reset on submission

## 6. Error Logging

- [x] 6.1 Add `console.error('[NewRunView]', err)` to each silent catch block in `NewRunView.tsx` (8+ instances)
- [x] 6.2 Add `console.error('[useTraces]', err)` to catch blocks in `web/src/hooks/useTraces.ts`
- [x] 6.3 Review and add error logging to any remaining silent catch blocks identified during the audit in step 4.3
