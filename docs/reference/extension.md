# Determinism extension reference

`extensions/aot-determinism.ts` — loaded into every agent run via `--extension /opt/aot/extensions/aot-determinism.ts`. Registers custom tools and enforces policies.

## Custom tools

### `ask_user`

Pauses the agent and asks the operator a question via the dashboard.

| Param | Type | Required | Notes |
|-------|------|----------|-------|
| `question` | `string` | yes | |
| `options` | `string[]` | no | Optional choices |

Writes `/workspace/.aot/input/question.json`; polls `/workspace/.aot/input/response.txt` until the sidecar's `SendInput` writes it. 5-minute timeout.

Manage role only — `implement` agents are blocked.

### `delegate_task`

Marker for dashboard visibility; subtask handled inline.

| Param | Type | Required | Notes |
|-------|------|----------|-------|
| `task` | `string` | yes | |
| `context` | `string` | no | |

Writes `/workspace/.aot/subagents/<id>.json`.

## Policies

### Loop detection

Blocks the 3rd consecutive identical call. Counter resets on a different call.

### Turn cap

Kills the agent after 50 turns.

### Roles (`PI_ROLE`)

| Role | Restrictions |
|------|--------------|
| `manage` | Writes confined to `/workspace/openspec/` + `/workspace/.aot/`. Can use `ask_user`. |
| `implement` | Repo writes allowed. No `ask_user` — surface questions in output. |

### Plan-stage write validation (`PI_STAGE=plan`)

- Spec files (`*/specs/*/spec.md`): must use `SHALL` or `MUST` in requirements.
- `tasks.md`: ≤ 30 checkboxes.

### Protected paths

Writes outside `/workspace` blocked.

## Env

| Var | Default | Purpose |
|-----|---------|---------|
| `PI_STAGE` | `""` | `plan` / `execute` / `verify` |
| `PI_ROLE` | `implement` | `manage` / `implement` |

## File contracts

| Path | Use |
|------|-----|
| `/workspace/.aot/input/question.json` | HITL question payload |
| `/workspace/.aot/input/response.txt` | HITL response |
| `/workspace/.aot/subagents/*.json` | Delegation markers |
| `/workspace/.aot/logs/agent.jsonl` | Audit log of agent execution |

## Sidecar-level backups

The sidecar also kills the agent on 5 consecutive identical tool-call signatures (defense in depth in case the extension misses).
