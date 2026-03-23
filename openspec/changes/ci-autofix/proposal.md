## Why

UNCWORKS can now create PRs automatically after a successful spec-driven run (Plan, Execute, Verify, Push, PR). But the pipeline stops there. When CI checks fail on the PR — linting errors, test failures, type errors, build issues — nobody fixes them. The PR sits with a red X until a human intervenes.

This is the biggest gap in the autonomous loop. The agent wrote code, the internal verifier approved it, but the external CI (the actual source of truth for code quality) rejected it. The agent should be able to read the CI failure, understand what went wrong, fix the code, and push again — the same way a human developer would.

Without this, every PR that fails CI requires manual intervention, which defeats the purpose of autonomous agent runs. With it, UNCWORKS becomes a closed-loop system: code that passes both internal verification AND external CI.

## What Changes

### GitHub webhook for check events
Extend the existing webhook handler (`internal/server/webhook.go`) to listen for `check_run` and `check_suite` events. When a check fails on a branch that UNCWORKS created (`aot/*`), trigger the autofix flow.

### CI failure log extraction
Read the failing check's output via the GitHub Actions API (`GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs`). Parse the log to extract the specific error messages — compiler errors, test failures, lint violations. Condense into a structured prompt for the fix agent.

### Autofix agent run
Spawn a new AgentRun that:
- Checks out the existing PR branch (not main)
- Receives the CI error log as context in the prompt
- Fixes the specific issues identified in the CI output
- Commits and pushes to the same branch (triggering CI re-run)
- Does NOT create a new PR (the existing PR updates automatically)

### Retry loop with circuit breaker
Track fix attempts per PR. After N failed fix attempts (default: 3), stop and add a comment to the PR explaining what was tried and what failed. This prevents infinite fix loops.

### PR status tracking
Add a `ciFixAttempts` counter and `lastCIStatus` field to the run status. The UI shows whether the PR is passing, failing, or being auto-fixed.

## Capabilities

### New Capabilities
- `ci-webhook-handler`: Webhook handler for `check_run` completed events on `aot/*` branches, triggers autofix flow
- `ci-log-extraction`: Read GitHub Actions run logs via API, parse and condense failure messages into agent prompts
- `ci-autofix-run`: Spawn fix runs targeting existing PR branches with CI error context, push fixes to same branch
- `ci-retry-circuit-breaker`: Track fix attempts per PR, stop after max retries, comment on PR with failure summary

### Modified Capabilities
- None

## Impact

- **Modified**: `internal/server/webhook.go` — add `check_run` event handling alongside existing `push` handler
- **New**: `internal/server/ci_autofix.go` — CI log extraction, fix run spawning, attempt tracking
- **Modified**: `internal/temporal/workflow_spec_driven.go` — new workflow variant for fix runs (skip plan, go straight to execute with error context)
- **Modified**: `internal/temporal/activities_git.go` — support pushing to existing branch without creating new PR
- **Modified**: `api/v1alpha1/types.go` — add `CIFixAttempts`, `LastCIStatus`, `ParentPRUrl` to AgentRunStatus
- **Modified**: `deploy/crds/agentrun-crd.yaml` — new status fields
- **Modified**: `web/src/views/RunDetailView.tsx` — show CI fix status, link to parent PR
- **Modified**: `web/src/views/RunListView.tsx` — show CI fix badge on runs that are fixing a PR
- **Dependencies**: GitHub Actions API (already available via the GitHub token)
