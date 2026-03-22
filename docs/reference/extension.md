# Determinism Extension Reference

The UNCWORKS determinism extension (`extensions/aot-determinism.ts`) enforces guardrails on agent behavior within the spec-driven pipeline. It registers custom tools and applies policies to prevent runaway execution.

## Custom Tools

### ask_user

Pauses the agent and asks the human operator a question via the UNCWORKS dashboard.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `question` | `string` | Yes | The question to display |
| `options` | `string[]` | No | Optional list of choices |

**Behavior:** Writes a question payload to `/workspace/.aot/input/question.json`, then polls `/workspace/.aot/input/response.txt` until the sidecar's `SendInput` RPC writes a response. Times out after 5 minutes.

**Role restriction:** Only available to `manage` agents. `implement` agents are blocked from calling this tool and must surface questions in their output.

### delegate_task

Tracks a subtask delegation for dashboard visibility.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `task` | `string` | Yes | Description of the subtask |
| `context` | `string` | No | Additional context, file paths, or constraints |

**Behavior:** Writes a marker file to `/workspace/.aot/subagents/<id>.json` and returns guidance to handle the task inline. The delegation is tracked for observability but executed within the current agent.

## Policies

### Loop Detection

Blocks repeated identical tool calls. If the same tool is called 3+ consecutive times with identical input, the call is blocked and the agent receives an error message. The counter resets after a different tool call.

- **Threshold:** 3 consecutive identical calls (`MAX_REPEAT_CALLS`)

### Turn Limit

Kills the agent after a maximum number of conversational turns to prevent runaway execution.

- **Limit:** 50 turns (`MAX_TURNS`)

### Role-Based Restrictions

**manage agents** (`PI_ROLE=manage`):
- Cannot write or edit files outside `/workspace/openspec/` and `/workspace/.aot/`
- Can use `ask_user`

**implement agents** (`PI_ROLE=implement`):
- Can read and write repository source code
- Cannot use `ask_user`

### Write Validation (Plan Stage)

During `PI_STAGE=plan`, the extension validates writes to spec files:
- Spec files (`*/specs/*/spec.md`) must contain `SHALL` or `MUST` in requirement text
- Task files (`tasks.md`) are limited to 30 checkboxes to prevent over-decomposition

### Protected Paths

All write/edit operations to absolute paths outside `/workspace` are blocked.

## Configuration

The extension reads its configuration from environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PI_STAGE` | Current pipeline stage (`plan`, `execute`, `verify`) | `""` |
| `PI_ROLE` | Agent role (`manage` or `implement`) | `implement` |

## File Paths

| Path | Purpose |
|------|---------|
| `/workspace/.aot/input/question.json` | HITL question payload |
| `/workspace/.aot/input/response.txt` | HITL response from user |
| `/workspace/.aot/subagents/*.json` | Delegation tracking markers |
| `/workspace/.aot/logs/agent.jsonl` | Agent execution log |
