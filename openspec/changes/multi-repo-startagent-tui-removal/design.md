## Context

The AOT platform orchestrates agent runs via: API → CRD → Controller → Temporal Workflow → Pod (init + agent + sidecar). The current data model assumes one repo per agent run (`repo_url` + `branch` on `AgentRunSpec`). The hydration init container clones that single repo into `/workspace/src/`. The sidecar's `StartAgent` RPC then tries to launch the agent process in `req.RepoPath`, but the workflow never populates that field — so the agent fails to start.

The TUI package (`packages/tui/`) was an experiment with SolidJS + ANSI rendering. It has a working renderer and views but was never production-ready. The web dashboard and Go CLI cover all user-facing needs.

## Goals / Non-Goals

**Goals:**
- Agent runs accept multiple repositories, each cloned and worktree'd into the workspace
- The full agent lifecycle works end-to-end: Create → Hydrate → Start → Poll → Complete
- TUI code and all references are fully removed
- All existing tests continue to pass; new tests cover multi-repo hydration

**Non-Goals:**
- Cross-repo dependency resolution (repos are cloned independently)
- Monorepo sparse checkout (entire repo is cloned)
- Web UI multi-repo form (the API supports it; the UI can add multi-repo input later)
- Replacing devbox as the agent launcher (sidecar still uses `devbox run`)

## Decisions

### 1. Proto schema: `repeated Repository` message

Replace `string repo_url = 2` and `string branch = 3` with:
```protobuf
message Repository {
  string url = 1 [(buf.validate.field).string.uri = true];
  string branch = 2;
  // Directory name under /workspace/src/. Derived from repo name if empty.
  string path = 3;
}

message AgentRunSpec {
  Backend backend = 1;
  repeated Repository repos = 2;  // was repo_url + branch
  string prompt = 4;
  // ... rest unchanged
}
```

**Alternative considered**: Keep `repo_url` as the primary and add an optional `additional_repos` field. Rejected because it creates two code paths and the "primary" distinction is artificial.

**Migration**: Field numbers 2 and 3 are reassigned. This is a breaking proto change but we have no external consumers — only our own web UI and CLI. The generated Go/TS code will be regenerated.

### 2. CRD types: `[]Repository` struct

```go
type Repository struct {
    URL    string `json:"url"`
    Branch string `json:"branch,omitempty"`
    Path   string `json:"path,omitempty"`
}

type AgentRunSpec struct {
    Backend      string       `json:"backend"`
    Repos        []Repository `json:"repos"`
    Prompt       string       `json:"prompt"`
    // ... rest unchanged
}
```

Existing CRDs in the cluster with `repoURL`/`branch` fields will be ignored (they're from test runs). No migration needed.

### 3. Hydration: loop over repos

The hydrator receives a `[]RepoConfig` instead of a single `RepoURL`+`Branch`. For each repo:
1. Clone bare into `/workspace/.bare/<repo-name>/`
2. Create worktree at `/workspace/src/<path>/` (where `path` defaults to the repo name derived from URL)

The first repo in the list is the "primary" repo — its worktree path is returned as `WorktreePath` and used by the sidecar as the working directory for the agent process.

### 4. StartAgent fix: pass workspace path through workflow

`WaitForHydrationOutput` already returns `PodIP`. Add `WorkspacePath string` to carry the primary worktree path. The workflow passes this to `StartAgentInput.RepoPath`, which maps to the proto `StartAgentRequest.repo_path`. The sidecar uses it as `cmd.Dir`.

Fallback: if `RepoPath` is empty, the sidecar defaults to `/workspace/src` for backward compatibility.

### 5. TUI removal: delete and clean references

Straight deletion:
- `packages/tui/` directory (source + node_modules)
- `cmd/aot/main.go` dashboard subcommand
- `Taskfile.yml` targets: `test:tui`, `dev:tui`, TUI npm install in deps
- `devbox.json` test:tui script
- Doc/comment references in README.md, AGENTS.md, docs/user-guide.md, proto comments

## Risks / Trade-offs

- **[Risk] Proto field number reuse**: Reusing field numbers 2-3 in `AgentRunSpec` could cause wire-format confusion with old cached messages. → **Mitigation**: No external consumers exist. All clients are rebuilt together. The generated code is checked in.
- **[Risk] Hydration ordering**: Multiple repos clone sequentially, increasing init container time. → **Mitigation**: Acceptable for now. Parallel cloning is a future optimization.
- **[Risk] Workspace path assumptions**: The sidecar hardcodes `devbox run -- agent` as the launch command, which assumes devbox is configured in the repo. → **Mitigation**: Out of scope for this change. The sidecar's agent launching strategy is a separate concern.
