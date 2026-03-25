## Why

Six subsystem analysis agents identified 40+ critical and high severity issues across security, data integrity, reliability, and observability. These are not theoretical — several are exploitable vulnerabilities or silent data loss bugs in production paths that compound over time.

## What Changes

**Security fixes:**
- `internal/hydration`: Validate `Repository.Path` against path traversal (`..`, absolute paths) before `filepath.Join`
- `internal/hydration`: Rewrite `injectTokenInURL()` to use `url.Parse()` with host validation — prevents token exfiltration to attacker-controlled hosts
- `internal/server`: Reject webhook requests when HMAC secret is unset rather than silently allowing unauthenticated triggers

**Data integrity fixes:**
- `internal/brain/store.go`: Log and propagate `json.Unmarshal` errors in `GetRunSpans` instead of silently discarding metadata
- `internal/controller`: Log and requeue on `status.Update` failures instead of silently dropping them
- `internal/controller`: Clean up completed/failed `ChainRun` refs from `Schedule.status.active` during reconcile
- `internal/hydration`: Detect broken `.bare` clone directories via `git rev-parse`, remove and retry instead of proceeding with corrupt state
- `internal/hydration`: Log a warning (not silently ignore) when `AOT_REPOS` JSON is malformed

**Reliability fixes:**
- `internal/temporal`: Replace `workflow.Go()` background goroutine with selector-based approach to fix Temporal determinism violation
- `internal/controller`: Return `ctrl.Result{RequeueAfter: backoff}` on transient errors instead of `nil` (which drops the item)
- Frontend: Add `cancelled = true` guard before all `setState` calls in polling effects across 13+ views to prevent stale-closure state updates
- `internal/hydration`: Remove partial `.bare` directory on clone failure to prevent corrupt-state retry loops

**Resilience / observability:**
- `internal/temporal/knowledge_activities.go`: Return error from embedding failures instead of silent success with empty output
- `internal/server`: Add hard cap (500 items) on all list endpoints to prevent OOM from unbounded responses
- Frontend: Wrap `<Outlet>` in `Layout.tsx` with the existing `ErrorBoundary` component so single-view crashes don't kill the app
- Frontend: Replace `window.confirm()` destructive-action dialogs with the existing `AlertDialog` component

**API type quality:**
- `api/v1alpha1`: Define typed constants for all phase/status string values (`ChainRunPhaseRunning`, `AgentRunPhaseSucceeded`, etc.)
- `api/v1alpha1`: Add `+kubebuilder:validation:XValidation` CEL rule to `ScheduleSpec` enforcing `chainRef` and `templateRef` mutual exclusivity

## Capabilities

### New Capabilities
- `input-validation-hardening`: Validates repo paths and git URLs at hydration boundaries to prevent path traversal and token injection
- `webhook-auth-enforcement`: Requires HMAC secret on webhook endpoint; rejects unauthenticated trigger attempts
- `list-response-limits`: Hard cap on all server list endpoints to prevent OOM

### Modified Capabilities
- `context-hydration`: Adds path validation, URL parsing, broken-repo detection, error logging
- `run-pipeline`: Temporal workflow determinism fix; controller requeue-on-error; embedding error propagation
- `ui-views`: Polling race condition fixes; error boundary; AlertDialog for destructive actions

## Impact

- `internal/hydration/hydrator.go` — path validation, URL injection fix, broken-clone detection, partial cleanup, AOT_REPOS warning
- `internal/brain/store.go` — JSON error propagation in GetRunSpans
- `internal/controller/` — status.Update logging+requeue, Schedule active list cleanup, transient error requeue
- `internal/temporal/workflow.go` (or activities) — workflow.Go() → selector
- `internal/temporal/knowledge_activities.go` — embedding error propagation
- `internal/server/` — webhook auth check, list caps
- `api/v1alpha1/` — phase constants, ScheduleSpec CEL validation
- `web/src/` — polling guards (13+ views), ErrorBoundary in Layout, AlertDialog replacements
