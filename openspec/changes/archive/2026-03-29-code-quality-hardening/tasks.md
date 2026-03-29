## 1. Security — Hydration Path & URL Validation

- [x] 1.1 Add `validateRepoPath(path string) error` in `internal/hydration/hydrator.go` — reject absolute paths, paths containing `..`, and `filepath.Clean()`'d paths that start with `..`
- [x] 1.2 Call `validateRepoPath` in `Run()` for each repo before any filesystem operations; return error on violation
- [x] 1.3 Rewrite `injectTokenInURL()` to use `url.Parse()`, validate `u.Scheme == "https"` and `u.Host == "github.com"` (or configured allowlist), then set `u.User`; return original URL unchanged for SSH or non-allowlisted hosts
- [x] 1.4 Add unit tests in `internal/hydration/hydrator_test.go` for path traversal rejection, absolute path rejection, valid path acceptance, SSH passthrough, crafted URL with `@` rejection

## 2. Security — Webhook Auth Enforcement

- [x] 2.1 In the webhook handler (`internal/server/`), check if `WebhookSecret` is empty at request time; if so, return HTTP 401 with a descriptive message
- [x] 2.2 Add test: webhook returns 401 when secret is not configured
- [x] 2.3 Add test: webhook returns 200 with valid HMAC signature when secret is configured

## 3. Data Integrity — Brain Store

- [x] 3.1 In `internal/brain/store.go` `GetRunSpans()`, replace `_ = json.Unmarshal(metadataJSON, &sp.Metadata)` with proper error handling — log the error and return it (or log and continue with empty metadata, noting the corruption)

## 4. Data Integrity — Controller

- [x] 4.1 In the agent run controller reconcile loop, find all `status.Update` calls that currently ignore the error; log them and return the error instead
- [x] 4.2 In agent run and chain run controllers, find locations that `return nil` on transient errors (not-found of dependency, network errors); change to `return ctrl.Result{RequeueAfter: 10 * time.Second}, nil`
- [x] 4.3 In the schedule controller reconcile, add logic to iterate `.status.active` ChainRun refs, check each one's phase, and remove refs where phase is `succeeded` or `failed`; persist updated status

## 5. Data Integrity — Hydration

- [x] 5.1 In `cloneRepo()`, before using an existing `.bare` directory, run `git rev-parse --git-dir` in it; if the command fails, call `os.RemoveAll(bareDir)` and proceed with a fresh clone
- [x] 5.2 In `cloneRepo()`, on clone failure (git clone returns error), call `os.RemoveAll(bareDir)` before returning the error
- [x] 5.3 In `ConfigFromEnv()`, when `AOT_REPOS` JSON unmarshal fails, log `"WARNING: failed to parse AOT_REPOS as JSON: %v, falling back to single-repo"` instead of silently ignoring

## 6. Reliability — Temporal Determinism

- [x] 6.1 Locate the `workflow.Go()` call in `internal/temporal/workflow.go` (or the relevant workflow file) and identify what it is doing (likely signal/channel listening)
- [x] 6.2 Replace `workflow.Go()` with a `workflow.NewSelector`-based approach: add a receive case on the relevant channel/signal, handle it within the main select loop
- [x] 6.3 Verify the workflow compiles; run `go vet ./internal/temporal/...`

## 7. Reliability — Embedding Error Propagation

- [x] 7.1 In `internal/temporal/knowledge_activities.go` `HydrateContext()` and related functions, change embedding failure handling from logging a warning + returning empty success to returning the error so Temporal can retry

## 8. Reliability — Server List Caps

- [x] 8.1 Add a `capList[T any](items []T, max int) []T` helper in `internal/server/`
- [x] 8.2 Apply `capList(items, 500)` in all `handleList*` functions before marshaling the response
- [x] 8.3 Add a test verifying that listing with 600 items returns exactly 500

## 9. API Types — Phase Constants

- [x] 9.1 Create `api/v1alpha1/phases.go` with typed string constants for all phase values: `AgentRunPhase*` (pending, running, succeeded, failed, cancelled, waiting_for_input), `ChainRunPhase*`, `ChainRunStepPhase*`, `ScheduleLastResult*`
- [x] 9.2 Update all usages in `internal/controller/`, `internal/server/`, and `internal/temporal/` to reference the constants instead of raw strings

## 10. API Types — ScheduleSpec CEL Validation

- [x] 10.1 Add `+kubebuilder:validation:XValidation:rule="!(has(self.chainRef) && has(self.templateRef)) || (self.chainRef == '' || self.templateRef == '')",message="chainRef and templateRef are mutually exclusive"` to `ScheduleSpec` in `api/v1alpha1/schedule_types.go`
- [x] 10.2 Regenerate CRD manifests: `make generate manifests` (or equivalent task)

## 11. Frontend — Polling Race Condition Guards

- [x] 11.1 Fix `RunListView.tsx`: add `let cancelled = false` at start of `useEffect`, set in cleanup, check before `setRuns` and all other setState calls in the effect
- [x] 11.2 Fix `ProjectListView.tsx`: same pattern for `setProjects`
- [x] 11.3 Fix `ChainListView.tsx`: same pattern
- [x] 11.4 Fix `ChainRunListView.tsx`: same pattern
- [x] 11.5 Fix `ScheduleListView.tsx`: same pattern
- [x] 11.6 Fix `TemplateListView.tsx`: same pattern
- [x] 11.7 Fix `ActivityFeed.tsx`: add cancelled guard to all three polling intervals (thinking, structured logs, activity feed)
- [x] 11.8 Fix `GlobalNav.tsx`: add cancelled guard to `fetchCounts` polling
- [x] 11.9 Fix any remaining views with polling intervals (ProjectDetailView, ScheduleDetailView, ChainRunDetailView) using the same pattern

## 12. Frontend — Error Boundary

- [x] 12.1 In `web/src/components/Layout.tsx`, wrap the `<Outlet />` render with `<ErrorBoundary>` (the existing component at `web/src/components/ErrorBoundary.tsx`)

## 13. Frontend — AlertDialog for Destructive Actions

- [x] 13.1 In `RunListView.tsx`, replace `window.confirm()` delete confirmation with `AlertDialog` (already in codebase); add state for `pendingDeleteId`
- [x] 13.2 In `RunDetailView.tsx`, replace `window.confirm()` on archive and archive-with-workspace-delete with `AlertDialog`

## 14. Brain Store — Additional Data Integrity Fixes

- [x] 14.1 In `internal/brain/store.go` `GetRunSpans()`, initialize `sp.Metadata = make(map[string]interface{})` before calling `json.Unmarshal` — unmarshaling into a nil map silently produces nothing
- [x] 14.2 In `internal/brain/store.go` `SaveRunSpan()`, replace silent `metadataJSON = []byte("{}")` on marshal error with an actual error return
- [x] 14.3 In `internal/brain/store.go` `SaveCodeChunks()` and `SaveTraceChunks()`, wrap the per-chunk insert loop in a transaction so batch inserts are atomic (no partial writes on error)

## 15. Controller — Wiring Existing Validation

- [x] 15.1 In `internal/controller/chain_controller.go`, call `v1alpha1.ValidateChainDAG(chain)` at the start of ChainRun reconcile; if validation fails, set ChainRun phase to `failed` with a descriptive message and return without creating steps
- [x] 15.2 In `internal/controller/schedule_controller.go`, add an early check: if both `ChainRef` and `TemplateRef` are empty after normalization, set schedule condition to error state and return without triggering a run

## 16. Frontend — Archive Response Check

- [x] 16.1 In `RunDetailView.tsx`, fix the archive action to check `resp.ok` before calling `toast.success("Run archived")` — currently success toast fires even on HTTP 500

## 17. Verification

- [x] 17.1 `go test ./...` passes with no failures
- [x] 17.2 `cd web && npx tsc --noEmit` passes with no errors
- [ ] 17.3 `cd web && npm run build` succeeds
- [ ] 17.4 Manually verify: sending a webhook request without secret configured returns 401
- [ ] 17.5 Manually verify: attempting to create a Schedule with both chainRef and templateRef set is rejected by the API server
