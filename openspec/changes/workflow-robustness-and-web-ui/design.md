## Context

The AOT platform has a functional backend (proto, CRD, API server, Temporal workflow, hydration, sidecar) but two gaps prevent real use:

1. **Workflow robustness**: The status polling loop in `internal/temporal/workflow.go` silently discards GetAgentStatus errors and cleanup (RevokeLLMKey, CleanupPod) errors. Under real conditions (sidecar crash, network partition), the workflow loops forever showing "Running" while leaking resources.

2. **Web UI**: `web/src/App.tsx` is a read-only dashboard that polls every 5s. The full API client (`packages/shared/src/grpc/client.ts`) and reactive store (`packages/shared/src/store/agent-store.ts`) exist but aren't wired to the UI. Users can't create runs, cancel them, send human input, or see real-time events.

## Goals / Non-Goals

**Goals:**
- Workflow fails gracefully after sustained sidecar errors instead of looping forever
- Cleanup errors are logged (visible in Temporal UI) instead of silently discarded
- Users can create agent runs from the web UI
- Users can cancel runs and send human input from the web UI
- Events stream in real-time via watchAgentRun instead of polling
- Runs are navigable via URL (shareable links)

**Non-Goals:**
- HITL pipeline fix (sidecar never reports WAITING_FOR_INPUT) — separate change
- Authentication/authorization — separate change
- CLI commands — separate change
- Production hardening (TLS, health probes, resource limits) — separate change

## Decisions

### 1. Error counting with configurable threshold

Add a `consecutiveErrors` counter to the polling loop. On GetAgentStatus error, increment and log a warning. On success, reset to 0. After 5 consecutive errors (constant `maxConsecutiveStatusErrors`), transition to Failed with a descriptive message.

**Why not retry policy on the activity?** The activity already has a retry policy (3 attempts). The counter tracks consecutive *activity-level failures* across polls — i.e., 5 polling cycles where all 3 retries failed. This distinguishes transient errors from a dead sidecar.

### 2. Cleanup error logging (not failure)

Change `_ = workflow.ExecuteActivity(...)` to capture the error and log it via `workflow.GetLogger(ctx).Error(...)`. The workflow still returns nil on cleanup — cleanup failures shouldn't change the workflow's final phase (Succeeded/Failed/Cancelled). But the errors become visible in Temporal UI for debugging resource leaks.

### 3. @solidjs/router for client-side routing

Add `@solidjs/router` to web/package.json. Routes: `/` for list, `/runs/:id` for detail. The list view shows all runs; clicking navigates to `/runs/:id`. The detail view fetches the run by ID and starts watchAgentRun.

**Why not hash routing?** The vite dev proxy and nginx production config already handle SPA fallback. Path routing gives cleaner URLs.

### 4. Replace polling with watchAgentRun streaming on detail view

On the list page, keep polling (listAgentRuns every 5s) — it's the simplest way to show all runs. On the detail page, use `client.watchAgentRun(id, onEvent)` for real-time updates. Events feed into the agent store which updates the run's phase and appends to the event log.

**Why not stream for the list too?** Would need one stream per run or a server-side "watch all" RPC which doesn't exist. Polling the list is fine at 5s intervals.

### 5. Store-first architecture

Replace App.tsx's local signals with `createAgentStore()`. The store is the single source of truth. List page calls `store.setRuns()` on poll. Detail page calls `store.updateRun()` and `store.addEvent()` from the watch stream. Components read from `store.state`.

### 6. Inline styles (keep existing pattern)

The existing components use inline styles. Keep this pattern rather than introducing a CSS framework. The UI is functional, not design-heavy.

## Risks / Trade-offs

- **[Risk] watchAgentRun reconnection** → The shared package has a `ReconnectingStream` helper. Use it to auto-reconnect on disconnect. If reconnection fails, fall back to polling.
- **[Risk] Store hydration on navigation** → When navigating to `/runs/:id` directly, the store is empty. The detail page must fetch the run by ID first, then start the watch stream.
- **[Risk] 5 consecutive errors threshold** → Too low might cause false positives on slow networks; too high delays failure detection. 5 polls × 5s interval × 3 retries = ~75s to detect a dead sidecar, which is reasonable.
