## 1. Sidecar Checkpoint Logic

- [x] 1.1 Add `createGitCheckpoint(workDir, toolName string) (sha string, diff *SpanDiff)` function to `gateway.go` that stages all changes, commits, and returns the SHA + diff against previous checkpoint
- [x] 1.2 Add package-level `lastCheckpointSHA` state with mutex, reset in `StartAgent`
- [x] 1.3 Configure `git user.name` and `git user.email` in `StartAgent` after resolving workspace
- [x] 1.4 Replace `captureGitDiff()` call in `tool_execution_end` handler with `createGitCheckpoint()`
- [x] 1.5 Add `checkpointSHA` and `prevCheckpointSHA` to tool span metadata
- [x] 1.6 Remove the old `captureGitDiff` function and its untracked-file scanning logic

## 2. Tests

- [x] 2.1 Add unit test for `createGitCheckpoint` with a real temp git repo: verify commit is created, SHA returned, diff contains the right files
- [x] 2.2 Add test for no-change scenario: verify no commit is created when git status is clean
- [x] 2.3 Add test for checkpoint state reset: verify `lastCheckpointSHA` is cleared when `StartAgent` is called
- [x] 2.4 Add integration test: simulate consecutive checkpoints, verify second diff only shows changes since first
