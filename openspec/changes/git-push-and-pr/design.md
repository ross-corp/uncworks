## Architecture

### Enhanced Post-Verify Flow

```
Verify passes
  │
  ├── Read proposal.md from workspace via execInSidecar
  ├── Get diff stats via: git diff --stat HEAD~1
  │
  ├── PushChanges
  │   ├── git checkout -b aot/{runId}
  │   ├── git add -A && git commit
  │   ├── Inject token into remote URL
  │   ├── git push --force origin aot/{runId}  (force handles existing)
  │   └── Restore remote URL
  │
  └── CreatePR
      ├── Title: "feat({change}): {prompt truncated}"
      ├── Body:
      │   ## Summary
      │   {proposal.md content}
      │
      │   ## Changes
      │   N files changed, +M/-K
      │
      │   ## Pipeline
      │   - Run: ar-{id}
      │   - Change: {changeName}
      │   - Model: {modelTier}
      │   - Attempt: {N}
      │
      └── POST /repos/{owner}/{repo}/pulls
```

### Reading Proposal from Workspace

```go
// In postVerifyPushAndPR, before CreatePR:
proposalContent, _ := execInSidecar(ctx, sc, runName, "/workspace",
    fmt.Sprintf("cat openspec/changes/%s/proposal.md 2>/dev/null || echo ''", changeName))
```

### Diff Stats

```go
// In PushChanges, after commit:
diffStat, _ := gitExec(ctx, sc, runName, repoPath, "git diff --stat HEAD~1")
// Returns: " 3 files changed, 42 insertions(+), 5 deletions(-)"
output.DiffStat = strings.TrimSpace(diffStat)
```

### Force Push

Change `git push origin {branch}` to `git push --force origin {branch}` to handle re-runs that reuse the same branch name.
