## Why

The PushChanges and CreatePR activities exist and are wired into the post-verification flow, but they were never exercised in production because autoPush/autoPR fields weren't reaching the workflow (CRD schema bug, now fixed). With the fields flowing correctly, the push and PR flow needs to be verified end-to-end and the PR body needs to be richer — including the OpenSpec proposal, diff stats, and a link back to the UNCWORKS run.

## What Changes

- Enhance the PR body to include the OpenSpec proposal.md content, diff stats (+N/-N), and a link to the UNCWORKS run
- Read proposal.md from the workspace via sidecar before creating the PR
- Add diff stats summary (files changed, additions, deletions) to the PR body
- Ensure the push flow handles the case where the branch already exists (force push or create unique branch name)
- Add e2e test that creates a run with autoPush=true and verifies a branch is created

## Capabilities

### New Capabilities
- `rich-pr-body`: PR body includes OpenSpec proposal content, diff stats, run link, and change metadata

### Modified Capabilities
- None

## Impact

- **Modified**: `internal/temporal/workflow_spec_driven.go` — read proposal.md before CreatePR, include in body
- **Modified**: `internal/temporal/activities_git.go` — add diff stats to PushChangesOutput, handle existing branch
- **Test**: Verify full push+PR flow with GitHub token
