## 1. Webhook Handler for check_run Events

- [ ] 1.1 Add `checkRunPayload` and related types to `internal/server/ci_autofix.go` (check_run, check_suite, head_branch, conclusion, action fields)
- [ ] 1.2 In `WebhookHandler.ServeHTTP`, add a branch for `eventType == "check_run"` that calls `wh.handleCheckRunEvent(r.Context(), body)`
- [ ] 1.3 Implement `handleCheckRunEvent`: unmarshal payload, check `action == "completed"` and `conclusion == "failure"`, verify branch starts with `aot/`, check repo allowlist
- [ ] 1.4 Add `CI_AUTOFIX_MAX_RETRIES` env var read in `NewWebhookHandler` (default: 3), store as `maxFixRetries int` on `WebhookHandler`
- [ ] 1.5 Add debounce map (`pendingFixes map[string]*time.Timer`) to `WebhookHandler` to coalesce multiple check_run failures for the same commit SHA within a 30-second window
- [ ] 1.6 Add unit test: `TestHandleCheckRunEvent_FailureOnAotBranch` -- mock payload with aot/ branch and failure conclusion, verify fix run created
- [ ] 1.7 Add unit test: `TestHandleCheckRunEvent_SuccessIgnored` -- verify no action on success conclusion
- [ ] 1.8 Add unit test: `TestHandleCheckRunEvent_NonAotBranchIgnored` -- verify no action on non-aot branch

## 2. CI Log Extraction via GitHub API

- [ ] 2.1 Implement `fetchCILogs(ctx context.Context, owner, repo string, runID int64) (string, error)` -- fetch log zip from `GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs`, extract text from zip entries
- [ ] 2.2 Implement `condenseCIErrors(raw string) string` -- filter lines containing error indicators, truncate to 8000 chars with middle-out truncation
- [ ] 2.3 Implement `resolveActionsRunID(ctx context.Context, owner, repo string, checkSuiteID int64) (int64, error)` -- call `GET /repos/{owner}/{repo}/actions/runs?check_suite_id={id}`, return first run ID
- [ ] 2.4 Add `io.LimitReader` cap of 50 MB on the log zip response body
- [ ] 2.5 Add unit test: `TestCondenseCIErrors_FilterErrorLines` -- verify error lines are extracted and non-error lines dropped
- [ ] 2.6 Add unit test: `TestCondenseCIErrors_TruncateLongOutput` -- verify output is truncated to 8000 chars with marker
- [ ] 2.7 Add unit test: `TestResolveActionsRunID` -- mock GitHub API response, verify run ID extraction

## 3. Fix Run Workflow Variant

- [ ] 3.1 In `runSpecDrivenPipeline`, add early check: if `input.SpecSource` starts with `ci-autofix:`, skip the Plan stage
- [ ] 3.2 For fix runs, read `changeName` from existing OpenSpec artifacts in the workspace (the branch already has them from the original run)
- [ ] 3.3 If no OpenSpec change is found in the workspace, use a fallback prompt without spec references
- [ ] 3.4 Construct the fix prompt with CI error context: include condensed log, branch name, and instructions to fix specific errors
- [ ] 3.5 Fix runs still execute the Verify stage using the existing change's specs
- [ ] 3.6 Add workflow test: `TestSpecDrivenPipeline_CIAutofix_SkipsPlan` -- verify Plan activity is not called when specSource starts with `ci-autofix:`
- [ ] 3.7 Add workflow test: `TestSpecDrivenPipeline_CIAutofix_ExecutesAndVerifies` -- verify Execute and Verify are called normally

## 4. PushChanges Modification for Existing Branch

- [ ] 4.1 In `postVerifyPushAndPR`, verify that when `autoPush: true` and `autoPR: false`, only PushChanges is called (already works, confirm with test)
- [ ] 4.2 For fix runs, set the branch name to the original PR branch from the spec (not `aot/{newRunName}`)
- [ ] 4.3 Add logic to detect the original branch name from `spec.repos[0].branch` when `specSource` starts with `ci-autofix:`
- [ ] 4.4 Add unit test: `TestPostVerifyPushAndPR_FixRun_NoPRCreated` -- verify CreatePR is not called for fix runs

## 5. CRD Status Fields

- [ ] 5.1 Add `CIFixAttempts int32`, `LastCIStatus string`, and `ParentPRUrl string` to `AgentRunStatus` in `api/v1alpha1/types.go`
- [ ] 5.2 Add kubebuilder optional markers and JSON tags: `ciFixAttempts`, `lastCIStatus`, `parentPRUrl`
- [ ] 5.3 Update `deploy/crds/agentrun-crd.yaml` with the new status fields (run controller-gen or update manually)
- [ ] 5.4 In `createFixAgentRun`, set `status.ciFixAttempts` to the current attempt count and `status.lastCIStatus` to "failure"
- [ ] 5.5 In `handleCheckRunEvent`, when conclusion is "success" on an aot/ branch, update the latest AgentRun's `status.lastCIStatus` to "success"

## 6. UI Updates

- [ ] 6.1 In `web/src/views/RunDetailView.tsx`, show `parentPRUrl` as a link when present (label: "Fixing PR: #N")
- [ ] 6.2 In `web/src/views/RunDetailView.tsx`, show `ciFixAttempts` and `lastCIStatus` in the status section when non-zero/non-empty
- [ ] 6.3 In `web/src/views/RunListView.tsx`, add a "CI Fix" badge to runs where `specSource` starts with `ci-autofix:`
- [ ] 6.4 In `web/src/views/RunListView.tsx`, show `lastCIStatus` indicator (green check / red X) next to runs that have a `parentPRUrl`

## 7. Retry Tracking and Circuit Breaker

- [ ] 7.1 Implement `getFixAttemptCount(ctx context.Context, branch string) (int, error)` -- list AgentRuns with annotation `aot.uncworks.io/pr-branch: {branch}` where specSource starts with `ci-autofix:`, return count
- [ ] 7.2 In `handleCheckRunEvent`, call `getFixAttemptCount` before creating a fix run; if count >= max, call `postCircuitBreakerComment` instead
- [ ] 7.3 Implement `postCircuitBreakerComment(ctx context.Context, owner, repo string, prNumber int, attempts int) error` -- post issue comment via GitHub API
- [ ] 7.4 Implement `resolvePRNumber(ctx context.Context, branch string) (int, error)` -- find the PR number from the original AgentRun's `status.prUrl` or via GitHub API `GET /repos/{owner}/{repo}/pulls?head={branch}`
- [ ] 7.5 Add annotation `aot.uncworks.io/pr-branch: {branch}` to fix AgentRun CRDs at creation time
- [ ] 7.6 Add unit test: `TestGetFixAttemptCount` -- create mock AgentRuns with annotations, verify correct count
- [ ] 7.7 Add unit test: `TestCircuitBreaker_MaxRetriesReached` -- verify no fix run created and comment posted when count >= max
- [ ] 7.8 Add unit test: `TestCircuitBreaker_UnderMax` -- verify fix run created when count < max

## 8. Integration Testing

- [ ] 8.1 Add integration test: full webhook-to-fix-run flow with mocked GitHub API -- send check_run event, verify AgentRun CRD created with correct specSource, prompt, and branch
- [ ] 8.2 Add integration test: circuit breaker flow -- send check_run events until max retries, verify comment posted and no further runs created
- [ ] 8.3 Add integration test: debounce -- send two check_run failures for the same SHA within 30s, verify only one fix run created

## 9. Verification

- [ ] 9.1 Run `go test ./internal/server/...` -- all webhook and CI autofix tests pass
- [ ] 9.2 Run `go test ./internal/temporal/...` -- workflow tests including fix run variant pass
- [ ] 9.3 Run `go vet ./...` and `go build ./...` -- no errors
- [ ] 9.4 Run `npx tsc --noEmit -p web/tsconfig.json` -- web UI compiles with new status fields
- [ ] 9.5 Verify CRD YAML is valid: `kubectl apply --dry-run=client -f deploy/crds/agentrun-crd.yaml`
