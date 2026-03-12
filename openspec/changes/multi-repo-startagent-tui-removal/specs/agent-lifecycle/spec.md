## MODIFIED Requirements

### Requirement: StartAgent receives workspace path
The workflow SHALL pass the primary repo's worktree path to the `StartAgent` activity, which SHALL forward it to the sidecar's `StartAgentRequest.repo_path` field.

#### Scenario: Sidecar launches agent in correct directory
- **WHEN** the workflow calls `StartAgent` after hydration completes
- **THEN** the sidecar SHALL receive `repo_path` set to the primary repo's worktree path (e.g., `/workspace/src/frontend`)
- **AND** the sidecar SHALL set `cmd.Dir` to that path before executing the agent process

#### Scenario: Sidecar defaults workspace when repo_path is empty
- **WHEN** the sidecar receives a `StartAgentRequest` with empty `repo_path`
- **THEN** the sidecar SHALL default `cmd.Dir` to `/workspace/src`

### Requirement: WaitForHydration returns workspace path
The `WaitForHydration` activity SHALL return both the pod IP and the primary workspace path in its output.

#### Scenario: Hydration output includes workspace path
- **WHEN** hydration completes successfully
- **THEN** `WaitForHydrationOutput` SHALL contain `PodIP` (the pod's cluster IP) and `WorkspacePath` (the primary repo's worktree path)

## REMOVED Requirements

### Requirement: TUI dashboard
**Reason**: The TUI package is being removed in favor of the web UI and Go CLI. The web dashboard provides all functionality that the TUI aimed to deliver.
**Migration**: Use the web dashboard at `http://localhost:3000` or the `aot` CLI tool. The `aot dashboard` subcommand is removed.
