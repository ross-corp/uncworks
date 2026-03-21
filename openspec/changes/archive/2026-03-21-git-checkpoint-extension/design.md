## Architecture

### Checkpoint Flow

```
  tool_execution_start               tool_execution_end
        │                                   │
        │                                   ├── git add -A
        │  agent runs tool                  ├── git commit "aot-checkpoint: {tool}"
        │  (bash, write, read, etc.)        ├── prevSHA → currentSHA
        │                                   ├── git diff {prev}..{current}
        │                                   └── attach diff to span
        ▼                                   ▼
```

### Implementation in Sidecar

The checkpoint logic lives in `maybeCaptureStreamEvent` inside the `tool_execution_end` handler, replacing the current `captureGitDiff` call.

```go
// Package-level state
var (
    lastCheckpointSHA string  // SHA of the most recent checkpoint commit
    checkpointMu      sync.Mutex
)

// On tool_execution_end:
func createCheckpoint(workDir, toolName string) *SpanDiff {
    // 1. Stage all changes (including untracked)
    exec("git", "-C", workDir, "add", "-A")

    // 2. Check if there's anything to commit
    status := exec("git", "-C", workDir, "status", "--porcelain")
    if status == "" { return nil }  // No changes

    // 3. Commit with checkpoint message
    exec("git", "-C", workDir, "commit",
         "--no-verify", "-m", "aot-checkpoint: " + toolName)

    // 4. Get current SHA
    currentSHA := exec("git", "-C", workDir, "rev-parse", "HEAD")

    // 5. Diff against previous checkpoint
    var diff string
    if lastCheckpointSHA != "" {
        diff = exec("git", "-C", workDir, "diff", lastCheckpointSHA + ".." + currentSHA)
    } else {
        // First checkpoint — diff against parent
        diff = exec("git", "-C", workDir, "diff", "HEAD~1..HEAD")
    }

    // 6. Update state
    lastCheckpointSHA = currentSHA

    return parseDiffOutput(diff)
}
```

### Git Configuration

Before the first checkpoint, the sidecar configures git:
```
git config user.name "aot-agent"
git config user.email "agent@aot.uncworks.io"
```

This happens once in `StartAgent` after resolving the workspace.

### Checkpoint Branch Strategy

Checkpoints commit directly to the agent's working branch (the `aot/{branch}` worktree branch created by hydration). This is fine because:
- Each agent run has its own worktree branch
- The branch is ephemeral (not pushed unless autoPush is enabled)
- When autoPush runs, it squashes or uses the latest state

### Span Metadata

Each tool span includes:
```json
{
  "tool": "write",
  "stage": "execute",
  "role": "neph",
  "checkpointSHA": "abc1234",
  "prevCheckpointSHA": "def5678"
}
```

### Cleanup

The `lastCheckpointSHA` is reset to empty when `StartAgent` is called (new stage starts).
