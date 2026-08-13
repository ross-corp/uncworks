# Spec-driven pipeline

The pipeline runs three stages: Plan, Execute, and Verify. A Verify failure
retries Execute. Two agent roles share one source of truth, which is the OpenSpec
change directory.

## Stages

### Plan, run by the `manage` agent

1. Run `openspec init --tools pi --force` in `/workspace` if the directory is not
   initialized.
2. Run `openspec new change "<name>"`, which scaffolds
   `/workspace/openspec/changes/<name>/`.
3. Fetch the templates with `openspec instructions {proposal,specs,tasks} --json`.
4. Write `proposal.md`, `design.md`, the spec files, and `tasks.md`. The role is
   read-only outside `openspec/` and `.aot/`.
5. Run `openspec validate` and `openspec status` to confirm the artifacts are
   structurally sound.

Every spec requirement MUST use `SHALL` or `MUST`. Every scenario MUST use
`WHEN` and `THEN`. `tasks.md` MUST hold no more than 30 checkboxes. The
determinism extension enforces the cap.

### Execute, run by the `implement` agent

The agent reads the spec, writes the code, and marks each finished `tasks.md`
item as `[x]`. This role cannot call `ask_user`, so a question has to appear in
the output. On a retry, the prompt carries the previous failure report as a
prefix.

### Verify, run by the `manage` agent and the automated gates

Five gates run in order.

| Gate | What it checks |
|------|------|
| Task completion | Every `tasks.md` item is checked. Runs only when the run sets `openspecChange` |
| Structural validation | `openspec validate "<name>" --json` exits zero |
| File existence | Every backtick-wrapped path on a `THEN ... exist` line resolves |
| Test commands | Every backtick-wrapped command on a `WHEN` or `THEN` line with a keyword such as `run`, `test`, or `build` is executed |
| LLM judge | The manage agent evaluates each scenario against the diff and emits a JSON verdict |

The judge can mark a verdict salvageable, which means a retry can recover from
the failure. A failure that is not salvageable stops the pipeline at once.

When every gate passes, `openspec archive "<name>" --yes` moves the change into
the archive. An archive failure is logged and does not fail the run.

## Roles

| Role | Stage | What it can write |
|------|-------|-------------------|
| `manage` | plan, verify | `/workspace/openspec/` and `/workspace/.aot/` only |
| `implement` | execute | The repository source |

The determinism extension reads the role from the `PI_ROLE` environment variable.

## Retry

A Verify failure with retries left runs Execute again, with the failure report as
a prompt prefix. `MaxRetries` on the Execute stage bounds the loop and defaults
to 3. When the retries run out, the run fails and reports the last failure.

## Config

```yaml
pipelineConfig:
  plan:    { model: default-cloud, timeoutSeconds: 300, maxRetries: 2, onFailure: fail  }
  execute: { model: default-cloud, timeoutSeconds: 900, maxRetries: 3, onFailure: retry }
  verify:  { model: default-cloud, timeoutSeconds: 180, maxRetries: 1, onFailure: fail  }
```

`onFailure` takes `retry`, `fail`, or `skip`.

## Output

Verify writes a `VerificationResult` as JSON into the change directory.

```json
{
  "pass": false,
  "tasksCompleted": 5,
  "tasksTotal": 7,
  "validationValid": true,
  "automatedChecks": [
    {"name": "task_completion", "pass": false, "output": "5/7 tasks complete"}
  ],
  "llmVerdict": {"pass": false, "salvageable": true, "criteria": []},
  "failureReport": "...",
  "executionTimeMs": 12500
}
```

The UI shows it in the Verify tab. `GET /api/v1/runs/{id}/verification` returns
the same document.
