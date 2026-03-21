## Why

The current diff capture approach (`captureGitDiff` in the sidecar) is fragile — it runs `git diff HEAD` at an arbitrary point in time, races with the agent's file writes, misses untracked files, and produces empty diffs for new repos. The trace flame graph shows `hasDiff=False` on every span despite the agent creating files. Engineers need reliable per-tool-call diffs to understand what each operation changed.

## What Changes

- Add git checkpoint logic to the sidecar that auto-commits after each `tool_execution_end` event
- Each checkpoint commit creates a deterministic snapshot with message `"aot-checkpoint: {tool_name}"`
- Span diffs are computed between consecutive checkpoint commits (`git diff {prev_sha}..{current_sha}`)
- Checkpoints are on a detached branch (`aot/checkpoints`) so they don't pollute the agent's working branch
- Remove the fragile `captureGitDiff` approach (git diff HEAD + untracked file scanning)
- Add checkpoint commit SHAs to span metadata for traceability

## Capabilities

### New Capabilities
- `git-checkpoints`: Reliable per-tool-call git snapshots with deterministic diff capture between consecutive commits

### Modified Capabilities
- None

## Impact

- **Sidecar** (`internal/sidecar/gateway.go`): Replace `captureGitDiff` with checkpoint-based diff capture
- **Agent image**: No changes needed — git is already available in the sidecar container
- **Trace spans**: Now include reliable diffs and checkpoint SHA metadata
- **Frontend**: No changes — trace flame graph already renders diffs when `hasDiff=true`
- **Performance**: One `git add -A && git commit` per tool call (~50ms overhead)
