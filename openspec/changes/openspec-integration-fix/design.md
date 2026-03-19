## Context

An audit of the spec-driven pipeline found that OpenSpec CLI integration is partially faked. The plan stage returns hardcoded `SpecsValid: true`. The verify gates swallow errors. The validation gate hardcodes success. The system works because the agent happens to do the right thing, not because Temporal verifies it.

## Goals / Non-Goals

**Goals:**
- Every OpenSpec CLI call has real error handling — no `2>/dev/null`, no `|| true`
- PlanRun validates output via `openspec validate` and `openspec status` before proceeding
- Verify gates fail correctly when OpenSpec commands fail
- All OpenSpec command output is captured in structured logs
- Pre-execute check ensures OpenSpec change exists before running the execute agent

**Non-Goals:**
- Changing the pipeline architecture (stages stay the same)
- Adding new OpenSpec features (just fixing what's there)
- Changing the agent system prompts (they're already correct)

## Decisions

### Decision 1: ExecCommand with explicit error capture

Replace all `2>/dev/null` and `|| true` patterns with direct ExecCommand calls that capture stdout, stderr, and exit code. Parse JSON from stdout with Go's `json.Unmarshal` instead of piping through python3.

**Rationale:** The current python3 inline JSON parsing is fragile — if the output format changes or the command fails, errors vanish. Go-native parsing is deterministic.

### Decision 2: PlanRun runs openspec validate + status after agent completes

After `pollUntilAgentDone`, PlanRun:
1. Runs `openspec validate --json <change-name>` via ExecCommand
2. Parses the JSON response and checks `items[0].valid == true`
3. Runs `openspec status --change <change-name> --json` via ExecCommand
4. Checks that all `applyRequires` artifacts have `status: "done"`
5. Only then returns `SpecsValid: true`

If either fails, returns the error so the workflow can retry planning or fail.

### Decision 3: Pre-execute artifact check

Between plan and execute stages, the workflow checks (via ExecCommand):
```bash
test -f /workspace/openspec/changes/<id>/proposal.md && \
test -d /workspace/openspec/changes/<id>/specs && \
test -f /workspace/openspec/changes/<id>/tasks.md
```

If any artifact is missing, the workflow retries planning or fails.

### Decision 4: OpenSpec init in workspace

Before the planning agent starts, run `openspec init` if no `.openspec.yaml` exists in the workspace. This ensures the OpenSpec CLI has project context.

## Risks / Trade-offs

- **Stricter validation may increase pipeline failures** — if the planning agent produces imperfect OpenSpec output, the pipeline will now fail instead of silently passing. This is intentional — better to fail honestly than succeed fake.
- **Removing error swallowing may expose latent issues** — commands that were silently failing will now produce visible errors. This is debugging information, not a regression.
