# Creating runs

A run is one `AgentRun` CRD → one Temporal workflow → one agent pod. The CRD spec is the contract; everything in the UI and CLI maps onto it.

## Required fields

- `repos[]` — at least one git URL (HTTPS or SSH). `branch` and `path` default sensibly.
- `prompt` — task description. Auto-derived from `specContent` for spec-driven runs.

Everything else has defaults.

## Modes

| Mode | When |
|------|------|
| `single` | One agent, one prompt. Default for ad-hoc work. |
| `auto` | Senior agent decomposes into junior agents. Currently falls back to single-run execution. |
| `manual` | You list subtasks in `orchestration.tasks[]`. Max 7. Each task gets a junior agent. |
| `spec-driven` | Full Plan / Execute / Verify with OpenSpec. Auto-selected when `specContent` is set. See [spec-driven.md](spec-driven.md). |

## Approval gates

`approvalMode` controls what runs need before flipping to `Succeeded`. The default (empty) is **hybrid**.

| Mode | LLM judge | Human approval |
|------|-----------|----------------|
| `none` | — | — |
| `llm-judge` | yes | — |
| `hitl` | — | yes |
| `hybrid` (default) | yes — must pass | yes — after judge |

The judge uses a cheap dedicated model (`deepseek-v3.1`) regardless of the agent's model — the judge model is independent of cost choices for the run itself.

When a run is awaiting human approval, it sits in `WaitingForInput` and the UI shows Approve/Reject buttons. From the CLI:

```bash
uncworks input <run-id> approve
uncworks input <run-id> reject "reason"
```

## Phases

`Pending → Running → (WaitingForInput) → Succeeded | Failed | Cancelled`

`Running` covers everything from pod provisioning through the agent finishing and the approval gate completing. `WaitingForInput` is used for both `ask_user` calls during the run and the human-approval step at the end.

## Auto-push and PR

If `autoPush: true`, successful runs push to `aot/<run-id>`. With `autoPR: true`, a PR is opened against `prBaseBranch` (default `main`). CI failures on `aot/*` branches trigger an autofix run; after 3 attempts it falls back to a comment on the PR.

## OpenSpec integration

Set `openspecChange` to the change name when the run is implementing a specific OpenSpec proposal. The Verify stage uses it as a task-completion gate (`openspec list --change <name>`). Ad-hoc runs leave it empty and skip that gate.

## Where it shows up

| Field | Purpose |
|-------|---------|
| `project` | Project label, also displayed in the sidebar |
| `feature` | Feature/unit-of-work bucket |
| `tags[]` | Cross-cutting filters |
| `projectRef` | If set, empty run fields inherit from the Project CRD |
| `specRef` | Pulls spec from the project's config repo |

Full field reference: [reference/crd.md](../reference/crd.md).
