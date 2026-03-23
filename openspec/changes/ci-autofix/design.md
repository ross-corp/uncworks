## Context

The spec-driven pipeline currently ends at PR creation. When CI checks fail on the PR, the code sits with a red X until a human reads the logs, fixes the errors, and pushes again. This change closes the loop: the system detects CI failures on its own PRs, reads the error logs, spawns a fix agent, and pushes the fix to the same branch.

The existing infrastructure handles most of this already. The webhook handler (`internal/server/webhook.go`) processes GitHub events and creates AgentRun CRDs. The Temporal workflow (`internal/temporal/workflow_spec_driven.go`) runs Plan/Execute/Verify pipelines. The git activities (`internal/temporal/activities_git.go`) push branches and create PRs. This change extends each of those layers.

## Goals / Non-Goals

**Goals:**
- Detect CI failures on `aot/*` branches via `check_run` webhook events
- Fetch and parse GitHub Actions logs to extract actionable error messages
- Spawn fix runs that skip planning and go straight to Execute with CI error context
- Push fixes to the existing branch (no new PR) so CI re-runs automatically
- Stop after a configurable number of failed fix attempts per branch

**Non-Goals:**
- Support for non-GitHub CI providers (CircleCI, Jenkins, etc.) -- future work
- Fixing failures in user branches (only `aot/*` branches managed by the system)
- Modifying the agent's internal tooling or prompts beyond injecting CI error context
- Handling check_suite events directly (we use check_run which is more granular)

## Decisions

### 1. Extend existing webhook handler, do not add a new endpoint

The `WebhookHandler.ServeHTTP` method currently checks `X-GitHub-Event == "push"` and ignores everything else. Add a second branch for `check_run`. This keeps the single `/api/v1/webhooks/github` endpoint and reuses signature validation, repo allowlist, and token provider. The check_run handler calls a new method `handleCheckRunEvent` on WebhookHandler.

**Why not a separate endpoint?** GitHub sends all webhook events to one URL. Splitting handlers across endpoints would require registering the same URL twice with different event filters, which GitHub doesn't support per-URL.

### 2. Filter on branch prefix `aot/` and conclusion `failure`

The check_run payload includes `check_run.check_suite.head_branch`. Only trigger autofix when the branch starts with `aot/` (branches created by the system). The `check_run.conclusion` must be `failure` -- ignore `success`, `neutral`, `cancelled`, `skipped`, `timed_out`, and `action_required`.

Also filter on `action == "completed"` to avoid processing `created` and `requested_action` events.

### 3. Resolve check_run to Actions workflow run via check_suite ID

The check_run payload contains `check_run.check_suite.id`. Use this to find the associated workflow run: `GET /repos/{owner}/{repo}/actions/runs?check_suite_id={suite_id}`. This returns the workflow run(s) whose logs we need to fetch.

If the check_run is not from GitHub Actions (e.g., a third-party status check), the check_suite won't map to a workflow run. In that case, skip silently -- we can only fetch logs from GitHub Actions.

### 4. Log extraction: fetch zip, extract error lines, truncate to 8000 chars

GitHub Actions logs are served as a zip archive at `GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs`. Each file in the zip is one job's log. Extract all files, concatenate, then filter to lines containing error indicators (`error`, `Error`, `FAIL`, `failed`, `FAILED`, `panic`, `undefined`, `cannot find`). If the filtered output exceeds 8000 characters, keep the first 4000 and last 4000 with a truncation marker.

8000 chars is roughly 2000 tokens, leaving plenty of room in the agent's context window for the fix prompt and codebase.

### 5. Fix run workflow variant: skip Plan, reuse Execute/Verify

Create the fix AgentRun with the same repo/branch configuration but with `orchestrationMode: "spec-driven"` and a new spec field (e.g., `specSource: "ci-autofix:owner/repo#42"`). The workflow detects this source prefix and skips the Plan stage.

In `runSpecDrivenPipeline`, add a check at the top: if `input.SpecSource` starts with `ci-autofix:`, skip Plan and set `changeName` from the existing branch's OpenSpec artifacts (read from the workspace after hydration). Jump directly to the Execute/Verify loop.

The prompt for the Execute stage is:

```
CI AUTOFIX: The following CI checks failed on branch {branch}. Fix the errors and ensure CI passes.

CI Error Log:
{condensed_log}

The code is already checked out on the failing branch. Read the existing code, understand the errors, fix them, and commit.
Do NOT create new files unless absolutely necessary. Focus on fixing the specific errors shown above.
```

### 6. PushChanges reuse: push to existing branch, skip PR creation

The fix run sets `autoPush: true` and `autoPR: false`. The existing `postVerifyPushAndPR` function already handles this case: it pushes to the branch but does not create a PR. The `PushChanges` activity already uses `--force` push, so it works even if the branch has been force-pushed by a previous fix attempt.

The branch name for the fix run is the same as the original PR branch (e.g., `aot/ar-abc123`). This is passed through the AgentRun spec's `repos[0].branch` field.

### 7. Retry tracking: query existing AgentRuns by branch annotation

When a check_run failure arrives, the handler queries the Kubernetes API for AgentRuns with annotation `aot.uncworks.io/pr-branch: aot/ar-abc123` and `specSource` starting with `ci-autofix:`. Count the results to determine the current attempt number.

Store the max retries in an environment variable `CI_AUTOFIX_MAX_RETRIES` (default: 3) read at handler initialization.

**Why not store the counter in a ConfigMap or dedicated CRD?** The AgentRun CRDs already exist and are queryable. Adding an annotation makes the counter visible via `kubectl` and doesn't require a new resource type.

### 8. Circuit breaker: post PR comment via GitHub API

When max retries are reached, use the existing GitHub token provider to post an issue comment on the PR:

```
POST /repos/{owner}/{repo}/issues/{pr_number}/comments
{
  "body": "## CI Autofix Exhausted\n\nAutomatic fix attempts have been exhausted after {N} tries. The CI checks are still failing.\n\nPlease review the latest CI logs and fix the remaining issues manually.\n\n---\n*This comment was posted by the UNCWORKS CI autofix system.*"
}
```

The PR number is extracted from the branch name pattern (`aot/ar-{run_name}`) by looking up the original AgentRun's `status.prUrl` and parsing the PR number from it.

### 9. New CRD status fields

Add to `AgentRunStatus`:
- `CIFixAttempts int32` -- number of CI fix runs spawned for this PR
- `LastCIStatus string` -- last known CI status: "success", "failure", "pending"
- `ParentPRUrl string` -- URL of the PR this fix run is targeting

These fields are informational and populated by the webhook handler. The UI reads them to show CI fix status.

### 10. New file: `internal/server/ci_autofix.go`

All CI autofix logic lives in a new file to keep `webhook.go` focused on event routing. The new file contains:
- `handleCheckRunEvent(ctx, body)` -- parse payload, filter, orchestrate
- `fetchCILogs(ctx, owner, repo, runID)` -- fetch and extract Actions logs
- `condenseCIErrors(raw string)` -- parse and truncate error output
- `createFixAgentRun(ctx, params)` -- create the fix AgentRun CRD
- `getFixAttemptCount(ctx, branch)` -- query existing fix runs for this branch
- `postCircuitBreakerComment(ctx, owner, repo, prNumber, attempts)` -- post PR comment
- `checkRunPayload` and related types

## Risks / Trade-offs

- **[Risk] Infinite loop if fix introduces new failures** -- Mitigated by the circuit breaker (max 3 attempts). Even if each fix attempt causes different failures, the system stops after 3 tries.
- **[Risk] Race condition: multiple check_run events for the same commit** -- A single push can trigger multiple check runs (lint, test, build). Each failed check sends a separate `check_run` event. Mitigate by debouncing: when a check_run failure arrives, wait 30 seconds before creating the fix run. If another failure arrives for the same commit SHA during that window, merge them into a single fix run. Implement this with a simple in-memory map of `sha -> timer`.
- **[Risk] Log zip too large** -- GitHub Actions logs can be hundreds of MB for verbose builds. The zip fetch uses `io.LimitReader` capped at 50 MB. If the zip exceeds this, the truncated content may not contain useful errors, but the 8000-char condensation step handles this gracefully.
- **[Risk] Branch name assumption** -- The system assumes `aot/*` branches were created by UNCWORKS. If a user manually creates an `aot/` branch, the system would attempt to autofix CI failures on it. This is unlikely in practice and the circuit breaker limits the impact.
- **[Trade-off] No plan stage for fix runs** -- Fix runs skip planning because the change is already defined. This means the fix agent works without OpenSpec guidance, relying on the CI error log alone. If the errors are vague, the agent may struggle. Acceptable because CI errors are typically specific (file, line, message).
