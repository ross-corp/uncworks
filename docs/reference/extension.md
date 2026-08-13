# Determinism extension reference

`extensions/aot-determinism.ts` loads into every agent run through
`--extension /opt/aot/extensions/aot-determinism.ts`. It registers custom tools
and enforces the run policies.

## Custom tools

### `ask_user`

Pauses the agent and asks the operator a question through the dashboard.

| Param | Type | Required | Notes |
|-------|------|----------|-------|
| `question` | `string` | yes | |
| `options` | `string[]` | no | Optional choices |

The tool writes `/workspace/.aot/input/question.json`, then polls
`/workspace/.aot/input/response.txt` until the sidecar's `SendInput` writes it.
It times out after 5 minutes.

Only the `manage` role may call this tool. The extension blocks an `implement`
agent.

### `delegate_task`

Writes a marker so the dashboard can show the delegation. The subtask runs
inline.

| Param | Type | Required | Notes |
|-------|------|----------|-------|
| `task` | `string` | yes | |
| `context` | `string` | no | |

Writes `/workspace/.aot/subagents/<id>.json`.

## Policies

### Loop detection

The extension blocks the third identical call in a row. A different call resets
the counter.

### Turn cap

Kills the agent after 50 turns.

### Roles, read from `PI_ROLE`

| Role | Restrictions |
|------|--------------|
| `manage` | May write only to `/workspace/openspec/` and `/workspace/.aot/`. May call `ask_user` |
| `implement` | May write to the repository. May not call `ask_user`, so it raises questions in its output |

### Plan-stage write validation, active when `PI_STAGE=plan`

- Every requirement in a spec file at `*/specs/*/spec.md` MUST use `SHALL` or
  `MUST`.
- `tasks.md` MUST hold no more than 30 checkboxes.

### Protected paths

The extension blocks every write outside `/workspace`.

## Environment variables

| Variable | Default | Purpose |
|-----|---------|---------|
| `PI_STAGE` | `""` | One of `plan`, `execute`, or `verify` |
| `PI_ROLE` | `implement` | Either `manage` or `implement` |

## File contracts

| Path | Use |
|------|-----|
| `/workspace/.aot/input/question.json` | The question for the human |
| `/workspace/.aot/input/response.txt` | The human's response |
| `/workspace/.aot/subagents/*.json` | Delegation markers |
| `/workspace/.aot/logs/agent.jsonl` | Audit log of agent execution |

## Sidecar-level backups

The sidecar also kills the agent after 5 identical tool-call signatures in a row.
This is a second line of defense, in case the extension misses the loop.
