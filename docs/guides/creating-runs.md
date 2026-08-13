# Creating runs

One run is one `AgentRun` resource, one Temporal workflow, and one agent pod. The
resource spec is the contract. Every field in the UI and the CLI maps onto it.

## Required fields

- `repos[]` takes at least one git URL, over HTTPS or SSH. `branch` and `path`
  have defaults.
- `prompt` describes the task. A spec-driven run derives it from `specContent`.

Every other field has a default.

## Modes

| Mode | When to use it |
|------|------|
| `single` | One agent and one prompt. Use it for ad-hoc work. |
| `auto` | A senior agent decomposes the work into junior agents. This currently falls back to single-run execution. |
| `manual` | You list the subtasks in `orchestration.tasks[]`, up to 7. Each subtask gets a junior agent. |
| `spec-driven` | The full Plan, Execute, and Verify pipeline with OpenSpec. Setting `specContent` selects it. See [spec-driven.md](spec-driven.md). |

## Approval gates

`approvalMode` decides what a run needs before it reaches `Succeeded`. An empty
value means `hybrid`.

| Mode | LLM judge | Human approval |
|------|-----------|----------------|
| `none` | no | no |
| `llm-judge` | yes | no |
| `hitl` | no | yes |
| `hybrid` (default) | yes, and it must pass | yes, after the judge |

The judge always uses `deepseek-v3.1`, whatever model the agent uses. This keeps
the judge's cost independent of the run's cost.

A run that waits for human approval sits in `WaitingForInput`, and the UI shows
Approve and Reject buttons. The CLI does the same thing.

```bash
uncworks input <run-id> approve
uncworks input <run-id> reject "reason"
```

## Phases

`Pending → Running → (WaitingForInput) → Succeeded | Failed | Cancelled`

`Running` covers pod provisioning, the agent's work, and the approval gate.
`WaitingForInput` covers both an `ask_user` call during the run and the human
approval step at the end.

## Auto-push and pull requests

A successful run pushes to `aot/<run-id>` when `autoPush` is true. It opens a
pull request against `prBaseBranch` when `autoPR` is true. `prBaseBranch`
defaults to `main`.

A CI failure on an `aot/*` branch starts an autofix run. After 3 attempts, the
platform comments on the pull request instead.

## OpenSpec integration

Set `openspecChange` to the change name when the run implements a specific
OpenSpec proposal. The Verify stage uses it as a task-completion gate through
`openspec list --change <name>`. An ad-hoc run leaves the field empty and skips
that gate.

## Organizing fields

| Field | Purpose |
|-------|---------|
| `project` | Project label. The sidebar displays it |
| `feature` | Groups runs into a unit of work |
| `tags[]` | Cross-cutting filters |
| `projectRef` | Empty run fields inherit from the named `Project` resource |
| `specRef` | Pulls the spec from the project's config repo |

[reference/crd.md](../reference/crd.md) documents every field.
