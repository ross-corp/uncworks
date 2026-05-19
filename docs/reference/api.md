# API reference

ConnectRPC + REST on `:50055` (same mux). All RPCs gRPC- and HTTP/JSON-callable.

Proto: `proto/aot/api/v1/api.proto` (`AOTService`) and `proto/aot/agent/v1/agent.proto` (`AgentSidecarService`, internal).

## `AOTService`

### `CreateAgentRun`

```proto
rpc CreateAgentRun(CreateAgentRunRequest) returns (CreateAgentRunResponse);
```

Creates an `AgentRun` CRD. Generates `ar-XXXXXX` ID, LLM-derived display name, auto-sets project/feature/repo/tag labels.

`AgentRunSpec` fields (selected):

| Field | Type | Notes |
|-------|------|-------|
| `backend` | `Backend` | `POD` (only option) |
| `repos[]` | `Repository` | At least one for most runs |
| `prompt` | `string` | Required for single/manual; auto for spec-driven w/ `spec_content` |
| `model_tier` | `string` | Default `default` |
| `manage_model_tier`, `implement_model_tier` | `string` | Per-role override |
| `ttl_seconds` | `int32` | Default 3600 |
| `env_vars` | `map<string,string>` | Extra env on the agent |
| `spec_content` | `string` | CodeSpeak markdown → auto spec-driven |
| `spec_source` | `string` | `editor`, `webhook:github:...`, `ci-autofix:...` |
| `project_ref` | `string` | Inherit empty fields from a `Project` |
| `spec_ref` | `string` | Spec name in project's config repo |
| `orchestration_mode` | enum | `SINGLE` / `AUTO` / `MANUAL` / `SPEC_DRIVEN` |
| `orchestration` | `Orchestration` | Task list for `MANUAL` |
| `pipeline_config` | `PipelineConfig` | Per-stage overrides |
| `max_budget` | `double` | USD cap, enforced by LiteLLM virtual key |
| `auto_push` / `auto_pr` / `pr_base_branch` | | Git/PR automation |
| `approval_mode` | `string` | `""`/`hybrid` (default), `none`, `hitl`, `llm-judge` |
| `openspec_change` | `string` | Enables task-completion gate in Verify |
| `parent_run_id`, `spec_run_id` | `string` | Orchestration links |
| `image`, `devbox_config`, `workspace_name` | `string` | Workspace overrides |

### `GetAgentRun`

`{ id } → AgentRun`. Live state from Temporal query merged with CRD. Populates `children[]`.

### `ListAgentRuns`

Filters: `phase_filter`, `spec_run_id`, `parent_run_id`, `stage_filter`, `project_filter`, `feature_filter`, `tag_filter`, `limit`. Newest-first. Archived hidden unless `X-Include-Archived: true`.

### `WatchAgentRun`

Server-stream of `AgentRunEvent`. Emits current state first, then deltas until terminal. Event types: `PHASE_CHANGED`, `LOG`, `TOOL_CALL`, `WAITING_FOR_INPUT`, `COMPLETED`.

### `CancelAgentRun`

`{ id }`. Cancels the Temporal workflow.

### `SendHumanInput`

`{ agent_run_id, input } → { accepted }`. Forwards user input to a paused agent. For HITL questions: the agent's answer. For approval gates: `approve` / `reject` / `deny` / `no` (anything else is treated as approve, with the input used as a reject reason where applicable).

### `GetRunGraph`

`{ id } → RunGraph`. Tree of parent + children via `aot.uncworks.io/spec-run-id`. Nodes carry `name`, `phase`, `role` (`single`/`senior`/`junior`), `started_at`, `completed_at`.

### `SearchPastWork`

Vector search over past run artifacts. Needs the brain/embedder subsystem.

Filters: `query` (required), `repo_url`, `source_filter` (`CODE` / `TRACE` / unset), `created_after`, `created_before`, `limit` (default 10, max 100).

## REST

### Runs

| Method | Path | Returns |
|--------|------|---------|
| GET | `/api/v1/runs/{id}/files` | Directory listing |
| GET | `/api/v1/runs/{id}/files/content?path=` | File content |
| GET | `/api/v1/runs/{id}/logs` | `agent.log` (human-readable) |
| GET | `/api/v1/runs/{id}/logs/structured` | `agent.jsonl` |
| GET | `/api/v1/runs/{id}/logs/thinking` | Reasoning blocks |
| GET | `/api/v1/runs/{id}/verification` | `VerificationResult` JSON |

### Traces

| Method | Path | Returns |
|--------|------|---------|
| GET | `/api/v1/runs/{id}/traces` | Spans |
| GET | `/api/v1/runs/{id}/traces/{spanId}/diff` | Per-span git diff |
| GET | `/api/v1/runs/{id}/traces/watch` | SSE stream |

### Archive

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/runs/{id}/archive` | Soft-delete |
| POST | `/api/v1/runs/bulk-archive` | Bulk |

### Debug / exec

| Method | Path | |
|--------|------|---|
| POST/DELETE | `/api/v1/runs/{id}/debug` | Start / stop debug session (scales pod to 1) |
| GET | `/api/v1/runs/{id}/connect` | WebSocket pod connect |
| GET | `/api/v1/runs/{id}/exec` | WebSocket shell |

### Projects

| Method | Path | |
|--------|------|---|
| GET/POST | `/api/v1/projects` | List / create |
| GET/DELETE | `/api/v1/projects/{name}` | Read / delete |
| GET | `/api/v1/projects/{name}/files` | Config repo listing |
| GET/PUT | `/api/v1/projects/{name}/files/{path...}` | Read / write (commits) |

Create body:

```json
{
  "name": "my-project",
  "displayName": "My Project",
  "description": "...",
  "repos": [{"url": "https://github.com/owner/repo.git", "branch": "main"}],
  "devbox": {"packages": ["go@1.22", "nodejs@20"]},
  "defaults": {
    "modelTier": "default-cloud",
    "ttlSeconds": 1800,
    "autoPush": true,
    "autoPR": true,
    "prBaseBranch": "main"
  }
}
```

Write body:

```json
{ "content": "...", "commitMessage": "update spec (optional)" }
```

### Specs

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/specs/push` | Push to GitHub |
| GET | `/api/v1/specs/pull` | Pull from GitHub |
| GET | `/api/v1/specs/{id}/graph` | Run graph for a spec execution |
| GET | `/api/v1/specs/{id}/graph/watch` | SSE |

### Misc

| Method | Path | |
|--------|------|---|
| POST | `/api/v1/classify` | LLM classification of a prompt → project/feature/tags |
| POST | `/api/v1/webhooks/github` | GitHub webhook |

Webhooks:

| Event | Action | Behavior |
|-------|--------|----------|
| `push` | — | Scans commits for `.cs.md` files; creates an `AgentRun` per spec at the push SHA. |
| `check_run` | `completed` / `failure` on `aot/*` | CI autofix: debounce 30s, fetch logs, condense, create a fix run. Max 3 attempts per branch. |
| `check_run` | `completed` / `success` | Updates `lastCIStatus` on the run associated with the branch. |

Env:

| Var | Purpose |
|-----|---------|
| `GITHUB_WEBHOOK_SECRET` | HMAC-SHA256 secret; unset = no validation |
| `GITHUB_WEBHOOK_REPOS` | Comma-separated `owner/repo` allowlist; unset = open |
| `CI_AUTOFIX_MAX_RETRIES` | Default 3 |

## `AgentSidecarService` (internal)

Used by the Temporal worker → agent pods. Not for external clients.

| RPC | |
|-----|---|
| `StartAgent` | Spawn pi |
| `GetStatus` | Process state |
| `StopAgent` | SIGINT → SIGKILL |
| `SendInput` | Write HITL response |
| `ExecCommand` | Shell command in pod |
| `StreamOutput` | Server-stream stdout/stderr |
