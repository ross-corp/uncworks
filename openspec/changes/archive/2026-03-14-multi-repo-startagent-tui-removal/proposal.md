## Why

Agent runs are currently limited to a single repository, but real-world agents often need to work across multiple repos (e.g., a monorepo frontend + a backend service). The `StartAgent` sidecar RPC also fails silently because the workflow never passes the workspace path, making the entire agent lifecycle non-functional past hydration. Additionally, the TUI package is dead weight — we're committed to the web UI and Go CLI.

## What Changes

- **Multi-repo support**: Replace the single `repo_url`/`branch` fields with a repeated `Repository` message across the proto schema, CRD types, hydration logic, workflow inputs, and pod spec. Each repo is cloned and worktree'd into its own subdirectory under `/workspace/src/`. **BREAKING**: `repo_url` and `branch` fields removed from `AgentRunSpec` proto, replaced by `repeated Repository repos`.
- **Fix StartAgent sidecar failure**: The workflow's `StartAgentInput` now carries `RepoPath` (defaulting to `/workspace/src/<primary-repo>`). The sidecar receives this in `StartAgentRequest.repo_path` so it knows where to `cd` before launching the agent process. Also pass `WorkspacePath` in `WaitForHydrationOutput` so the workflow has the path available.
- **Remove TUI package**: Delete `packages/tui/` entirely. Remove all references from `Taskfile.yml`, `cmd/aot/main.go`, `devbox.json`, `README.md`, `AGENTS.md`, `docs/user-guide.md`, and proto service comments. No TUI replacement — web UI and CLI cover all use cases.

## Capabilities

### New Capabilities
- `multi-repo-hydration`: Support cloning and creating worktrees for multiple repositories in a single agent pod workspace.

### Modified Capabilities
- `agent-lifecycle`: Fix the StartAgent RPC to pass workspace path, making the full lifecycle (Create → Hydrate → Start → Poll → Complete) functional end-to-end.

## Impact

- **Proto schema**: `AgentRunSpec` gains `repeated Repository repos` field, loses `repo_url` and `branch`. Requires `buf generate` for Go + TS codegen.
- **CRD types**: `AgentRunSpec` struct changes. Existing CRDs with `repoURL`/`branch` need migration or the controller must handle both shapes.
- **Hydration**: `hydrator.go` changes from single-repo to multi-repo loop. Tests updated.
- **Workflow/Activities**: `WorkflowInput`, `CreateAgentPodInput`, `StartAgentInput` gain repo list and workspace path fields.
- **Sidecar**: `startAgentProcess` uses the provided `repo_path` instead of empty string.
- **API server**: `specProtoToCRD` maps repeated repos to CRD.
- **Web dashboard**: Update `CreateAgentRun` form to accept multiple repos (or at minimum, pass through the new proto shape).
- **Deleted**: `packages/tui/` (~10 source files + node_modules), Taskfile targets, CLI dashboard subcommand, doc references.
- **CI**: Remove `test:tui` from the `test` aggregate task. Lint/type-check steps for TUI no longer needed.
