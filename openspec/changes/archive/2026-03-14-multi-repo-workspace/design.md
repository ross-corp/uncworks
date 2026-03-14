## Context

The system already supports multiple repos per AgentRun — the hydrator clones N repos into `/workspace/src/<name>/` as git worktrees. However, the agent experience is single-repo-centric:

- The workflow computes `workspacePath` from the first repo and passes it as the agent's working directory
- The sidecar starts the agent process in that single repo path via `devbox run`
- Devbox setup only runs against the primary repo's `devbox.json`
- The UI form has a single-repo dropdown
- The web-side types flatten `repos[0]` into a single `repoURL` string, losing all other repos
- There's no way to save and reuse multi-repo configurations

The agent has no structured awareness of what else is in its workspace, and the UI actively prevents multi-repo usage despite the backend supporting it.

## Goals / Non-Goals

**Goals:**
- Agent starts at `/workspace` with full visibility of all repos
- Hydrator generates `uncspace.yaml` manifest at workspace root describing the workspace layout
- Devbox environments from individual repos are composed into a single root-level `devbox.json` automatically
- Agent can install additional devbox packages as needed during a run
- UI carries full multi-repo data through the entire component tree
- Workspace presets let users save and reuse multi-repo + branch configurations
- All UI components (form, table, detail panel, sidebar, search) are multi-repo aware

**Non-Goals:**
- Server-side workspace storage — localStorage is sufficient to validate the UX; server-side can come later
- Managing cross-repo git operations (PRs, merges) — the agent handles git like a developer would
- Devbox version pinning or lock file management across repos — let devbox handle conflicts naturally

## Decisions

### 1. Agent working directory: `/workspace` root

**Decision**: Change agent start directory from `/workspace/src/<first-repo>` to `/workspace`.

**Rationale**: The agent is working on a task that may span repos. Anchoring it in one repo is arbitrary. Starting at the root gives it a natural vantage point over the entire workspace. The `uncspace.yaml` file at the root gives it structured context.

**Alternative considered**: Start at `/workspace/src/` — rejected because the devbox shell and `uncspace.yaml` live at `/workspace`.

### 2. `uncspace.yaml` manifest format

**Decision**: Hydrator generates `/workspace/uncspace.yaml` with this structure:

```yaml
repos:
  - path: src/api
    url: https://github.com/org/api.git
    branch: main
  - path: src/web
    url: https://github.com/org/web.git
    branch: main
devbox:
  composed: true
  sources:
    - src/api/devbox.json
    - src/web/devbox.json
```

**Rationale**: YAML is human-readable and matches the rest of the platform's config format. The `devbox.sources` field tells the agent which repos contributed to the environment. Paths are relative to `/workspace`.

**Alternative considered**: JSON — rejected for readability. Environment variable — rejected because it's structured data the agent should be able to read from a file.

### 3. Devbox composition via root-level include

**Decision**: Hydrator scans each repo worktree for `devbox.json`, then generates a root `/workspace/devbox.json` using devbox's `include` directive:

```json
{
  "include": [
    "src/api/devbox.json",
    "src/web/devbox.json"
  ]
}
```

Then runs `devbox install` at `/workspace`. The sidecar runs `devbox run` from `/workspace`.

**Rationale**: Repos keep owning their environment definitions. The hydrator handles composition mechanically. `devbox include` is the native way to compose configs — no custom merging logic needed. If a repo has no `devbox.json`, it's simply skipped.

**Alternative considered**: Agent manually runs `devbox shell` in each repo directory — rejected because it wastes agent tokens on mechanical setup and creates a fragmented environment.

### 4. Agent can self-serve additional packages

**Decision**: The agent can run `devbox add <package>` during a run if it discovers it needs something not covered by repo configs.

**Rationale**: Just like a human developer would install a missing tool. The composed devbox.json provides the baseline; the agent fills gaps. No special mechanism needed — devbox already supports this.

### 5. Existing `devbox_config` field behavior

**Decision**: If `AgentRunSpec.devbox_config` is set, it takes precedence as an explicit override — the hydrator uses it as-is instead of auto-composing. If it's empty (the common case), auto-composition kicks in.

**Rationale**: Backward compatibility. Existing runs that specify a devbox config path keep working. Auto-composition is the new default for runs that don't specify one.

### 6. `workspace_name` proto field

**Decision**: Add `string workspace_name = 12` to the proto `AgentRunSpec` message and corresponding `WorkspaceName string` to the CRD `AgentRunSpec` type.

**Rationale**: The backend doesn't act on this field — it's metadata that flows through to the UI so runs can be filtered by workspace. A proto field is cleaner than stuffing it in env vars or localStorage-only tagging (which would be lost on browser clear). One field, backward compatible (empty string = no workspace).

**Alternative considered**: localStorage mapping of `runId → workspaceName` — rejected because it's fragile and lost on storage clear. Env vars hack — rejected because it's a misuse of the field.

### 7. Workspace presets in localStorage

**Decision**: Workspace presets are stored client-side in localStorage under key `uncworks:workspaces` as a JSON array of `Workspace` objects:

```typescript
interface Workspace {
  id: string;          // uuid
  name: string;        // "payments-platform"
  description: string; // human-readable summary
  repos: Array<{
    url: string;
    branch: string;
  }>;
  createdAt: string;   // ISO timestamp
  updatedAt: string;
}
```

**Rationale**: Zero backend changes. Fastest to ship. Validates the UX before committing to a server-side resource. Workspaces are a UI convenience — they don't affect agent behavior. The `workspace_name` field on the run spec provides the link back.

**Alternative considered**: Server-side `Workspace` CRD — rejected for now as premature. Can be added later if cross-device/cross-user sync is needed.

### 8. Web type model: stop flattening repos

**Decision**: Replace `repoURL: string` + `branch: string` on the web-side `AgentRunSpec` type with `repos: Repository[]` and `workspaceName?: string`. The `mapRun()` function in `useClient.ts` passes through the full repos array instead of extracting `repos[0]`.

**Rationale**: The current flattening is the root cause of every UI limitation. Every component that touches repo data needs the full array. This is a breaking change to the web types but affects no external API.

### 9. AgentRunForm redesign

**Decision**: The form has two modes:
1. **Workspace mode**: Select a saved workspace preset — repos pre-fill but can be modified for this run (add/remove/change branches)
2. **Custom mode**: Build a repo list from scratch

Each repo row has its own URL and branch field. Repos can be added/removed. The form submits the full `repos[]` array and optionally the `workspaceName`.

**Rationale**: Workspace presets eliminate repetitive multi-repo configuration. Allowing modification per-run means presets don't constrain you. Per-repo branches reflect reality — different repos are often on different branches.

### 10. Sidebar workspace filtering

**Decision**: Add a "Workspaces" collapsible section to the sidebar between "Agent Runs" and "Repositories". Each workspace shows a count of runs tagged with that workspace name. Clicking filters to those runs. The "Repositories" section remains and shows runs that include a given repo (regardless of workspace).

**Rationale**: Workspaces and repos are orthogonal filters. Workspace = "runs I created for this project." Repo = "all runs touching this codebase." Both are useful. A run with `[api, web]` appears under both `api` and `web` in the repo filter, and under its workspace name.

### 11. Multi-repo table display

**Decision**: The repo column in `AgentRunTable` shows comma-separated repo names. If >2 repos, truncate to `api, web +1` with a title tooltip showing all. The branch is not shown in the table column (too dense) — it's visible in the detail panel.

**Rationale**: Table space is limited. Repo names are short and scannable. Branches add noise at the table level — they're important in the detail view.

### 12. Repo registry

**Decision**: Transform `ReposView` from a read-only table into an interactive registry. Users can add repo URLs (which become available in the form dropdown) and remove repos they no longer use. Repos are stored in localStorage under `uncworks:repos` as a simple URL array. The existing derived-from-runs list is merged with the registry.

**Rationale**: Currently you can only pick repos that have been used in past runs. For new repos, you'd have to type the URL manually in the form. A registry lets you pre-register repos you plan to use.

## Risks / Trade-offs

**Devbox include conflicts** — Two repos may declare incompatible versions of the same package. → Mitigation: Let `devbox install` fail naturally and surface the error in the hydration phase. The hydrator logs which sources it composed so the error is diagnosable.

**Agent context overhead** — The agent now needs to understand `uncspace.yaml` and navigate multiple repos. → Mitigation: The manifest is small and structured. The agent's system prompt should reference it.

**Backward compatibility of working directory change** — Existing runs expect the agent to start in a specific repo. → Mitigation: The `devbox_config` override preserves old behavior for explicit configs. For auto-composed runs, the agent starts at `/workspace` but all repos are still at `/workspace/src/<name>/`.

**localStorage loss** — Clearing browser storage loses workspace presets and repo registry. → Mitigation: Acceptable for now. Workspace names on runs survive (they're in the proto). Presets can be re-created. Server-side storage is the long-term answer.

**Multi-repo sidebar counts** — A run touching 3 repos appears under all 3 in the repo filter, so counts sum to more than total runs. → Mitigation: This is expected and matches how tags work everywhere. Not confusing if clearly labeled.

**Form complexity** — The redesigned form is significantly more complex than the current single-repo dropdown. → Mitigation: Workspace presets make the common case (reusing a saved config) simpler than today. Custom mode is only needed for new combinations.
