## Context

The spec-driven pipeline was implemented across 50 tasks in the `spec-driven-agent-runs` change. The Go code compiles, unit tests pass, and proto code is generated. However, the Docker images haven't been rebuilt, the cluster hasn't been updated, and no real spec-driven run has ever executed. The current `execInSidecar` function spawns a full pi-coding-agent to run bash commands, which is slow and unreliable — it needs a lightweight alternative.

## Goals / Non-Goals

**Goals:**
- First successful spec-driven run end-to-end in aot-local
- Lightweight command execution in the sidecar (replace agent-based exec)
- All runtime issues fixed and verified
- Web UI showing stage progression and verification results

**Non-Goals:**
- Streaming stage output in real-time (future enhancement)
- Full automated scenario command extraction from spec WHEN/THEN (works but basic)
- Knowledge system integration (separate roadmap item)

## Decisions

### Decision 1: Add ExecCommand RPC to sidecar

Add a new `ExecCommand` RPC to the `AgentSidecarService` proto that runs a bash command directly in the workspace and returns stdout/stderr/exit code. This replaces the current `execInSidecar` which spawns a full pi-agent.

```proto
rpc ExecCommand(ExecCommandRequest) returns (ExecCommandResponse);

message ExecCommandRequest {
  string command = 1;
  string working_dir = 2;
  int32 timeout_seconds = 3;
}

message ExecCommandResponse {
  string stdout = 1;
  string stderr = 2;
  int32 exit_code = 3;
}
```

**Rationale:** The verification gates need to run `openspec validate --json`, `openspec list --json`, `openspec archive --yes`, and test commands. Spawning a full AI agent for each is a 10-30 second overhead per command. Direct exec is <1 second.

### Decision 2: Deploy-test-fix cycle, not big-bang

Deploy first, create a test run, observe failures, fix, redeploy. Iterate rapidly rather than trying to predict all issues upfront.

### Decision 3: Use default-cloud model for planning and verification agents

The plan and verify stages need capable models (not the 0.5b toy). Use the `default-cloud` (qwen3-coder) model for all spec-driven stages.

## Risks / Trade-offs

- **ExecCommand is a shell injection surface** — mitigated by only calling it from Temporal activities (server-side), never from user input. The workspace is already an untrusted environment (agents run arbitrary code).
- **First run will likely fail** — expected. The iteration cycle is: deploy → run → read logs → fix → repeat.
