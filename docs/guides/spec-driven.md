# Spec-driven pipeline

Three stages — Plan, Execute, Verify — with a retry loop on Verify failure. Two agent roles, one source of truth (the OpenSpec change directory).

## Stages

### Plan — `manage` agent

1. `openspec init --tools pi --force` in `/workspace` (if needed).
2. `openspec new change "<name>"` scaffolds `/workspace/openspec/changes/<name>/`.
3. Templates fetched via `openspec instructions {proposal,specs,tasks} --json`.
4. Agent writes `proposal.md`, `design.md`, spec files, `tasks.md` — read-only outside `openspec/` and `.aot/`.
5. `openspec validate` and `openspec status` confirm structural integrity.

Spec requirements must use `SHALL` or `MUST`. Scenarios use `WHEN/THEN`. `tasks.md` is capped at 30 checkboxes — enforced by the determinism extension.

### Execute — `implement` agent

Reads the spec, writes code, ticks `tasks.md` items as `[x]`. No `ask_user` from this role — questions must surface in output. On retry, the prompt is prefixed with the previous failure report.

### Verify — `manage` agent + automated gates

Five gates, evaluated in order:

| Gate | What |
|------|------|
| Task completion | All `tasks.md` items checked (only when `openspecChange` is set on the run) |
| Structural validation | `openspec validate "<name>" --json` |
| File existence | Backtick-wrapped paths in `THEN ... exist` lines must resolve |
| Test commands | Backtick-wrapped commands on `WHEN/THEN` lines with keywords (`run`, `test`, `build`, …) get executed |
| LLM judge | Manage agent evaluates each scenario against the diff and emits a JSON verdict |

The judge can mark a verdict **salvageable** — meaning the failure is recoverable on retry. Non-salvageable failures terminate the pipeline immediately.

On overall pass, `openspec archive "<name>" --yes` moves the change to the archive. Archive failure is logged, not fatal.

## Roles

| Role | Stage | What it can touch |
|------|-------|-------------------|
| `manage` | plan, verify | `/workspace/openspec/`, `/workspace/.aot/` only |
| `implement` | execute | Repo source code |

Loaded via `PI_ROLE` env var by the determinism extension.

## Retry

Verify failure with retries left → Execute again, prompt prefixed with the failure report. `MaxRetries` on the Execute stage is the bound (default 3). When exhausted, the run fails with the last report.

## Config

```yaml
pipelineConfig:
  plan:    { model: default-cloud, timeoutSeconds: 300, maxRetries: 2, onFailure: fail  }
  execute: { model: default-cloud, timeoutSeconds: 900, maxRetries: 3, onFailure: retry }
  verify:  { model: default-cloud, timeoutSeconds: 180, maxRetries: 1, onFailure: fail  }
```

`onFailure`: `retry`, `fail`, or `skip`.

## Output

Verify writes a `VerificationResult` JSON to the change directory:

```json
{
  "pass": false,
  "tasksCompleted": 5,
  "tasksTotal": 7,
  "validationValid": true,
  "automatedChecks": [
    {"name": "task_completion", "pass": false, "output": "5/7 tasks complete"}
  ],
  "llmVerdict": {"pass": false, "salvageable": true, "criteria": [/* ... */]},
  "failureReport": "...",
  "executionTimeMs": 12500
}
```

Visible in the UI's Verify tab and via `GET /api/v1/runs/{id}/verification`.
