## Context

AOT currently runs agents as a single-stage pipeline: hydrate workspace → start agent → poll until exit → report exit code as success/failure. The agent has no structured acceptance criteria, no verification step, and no retry capability. The platform reports "Succeeded" for any clean exit regardless of outcome quality.

The OpenSpec CLI (`openspec`) is installed globally and provides machine-parseable JSON output for validation, status tracking, and change management. Crucially, it has **built-in verification primitives**:
- `openspec validate --json` — checks spec structural validity, returns machine-parseable pass/fail
- `openspec list --json` — returns `completedTasks/totalTasks` and `status` (in-progress/complete) per change
- `openspec archive` — **refuses to archive incomplete changes** (prompts with "N incomplete task(s) found"), making it a gate for completion
- `openspec status --json` — tracks artifact completion per change

These are the verification engine. We build on them, not beside them.

## Goals / Non-Goals

**Goals:**
- Agent run success is determined by OpenSpec's own tooling (validate, list, archive), not process exit code
- Multi-stage pipeline (Plan → Execute → Verify) with automatic retry on verification failure
- User input of any fidelity (vague prompt to full spec) is normalized into a structured, evaluatable OpenSpec change
- `openspec archive` is the final gate — a run only succeeds if its change can be archived
- Automated checks from OpenSpec CLI combined with LLM judge for semantic WHEN/THEN evaluation
- Verification results exposed through API and web UI
- Each run produces an OpenSpec change as its documentation artifact

**Non-Goals:**
- Real-time streaming of planning/verification stage output (future enhancement — stages run sequentially, output captured in structured logs)
- User-editable specs mid-run (the plan stage output is final; user can cancel and re-run with more specific input)
- Parallel execution of plan + execute stages (strict sequential pipeline)
- Custom verification scripts provided by the user (automated checks derive from spec scenarios and OpenSpec CLI only)
- Replacing the existing single-stage mode for simple prompts (spec-driven mode is opt-in via orchestration mode or auto-detected for spec content)

## Decisions

### Decision 1: OpenSpec CLI as the verification engine

The verification stage is a pipeline of OpenSpec CLI commands plus an LLM judge:

```
Verify Pipeline:
  1. openspec list --json → completedTasks == totalTasks?
     NO  → FAIL (agent didn't finish all tasks)
  2. openspec validate --json → valid?
     NO  → FAIL (spec structure broken during execution)
  3. Exec automated checks from spec scenarios (tests, builds)
     FAIL → FAIL (code doesn't work)
  4. LLM judge for semantic WHEN/THEN criteria
     FAIL → FAIL (intent not met)
  5. openspec archive --yes → archives change, merges specs
     SUCCESS → run is Succeeded
```

`openspec archive` is the **final seal**. It merges the change's delta specs into the main spec tree, creating a permanent record. A failed run's change stays unarchived as the failure artifact.

**Alternatives considered:**
- Build custom verification from scratch → duplicates what OpenSpec already provides
- LLM-only evaluation → non-deterministic, can't catch compilation failures, doesn't leverage OpenSpec task tracking
- Only use `openspec archive` → insufficient, archive warns but doesn't block with `--yes`, and doesn't run tests

**Rationale:** OpenSpec already has structured validation and task completion tracking. We use these as the deterministic baseline and add automated scenario checks + LLM judge on top. The layered approach means fast failures (task incomplete? → fail immediately, no LLM call needed).

### Decision 2: OpenSpec change per run, on workspace PVC

Each run creates an OpenSpec change directory on the workspace PVC. The planning agent runs `openspec new change "<run-id>"` to scaffold it.

**Alternatives considered:**
- Store specs in K8s ConfigMaps → too large, 1MB limit, poor tooling
- Store in PostgreSQL brain → adds DB dependency to critical path, harder to debug

**Rationale:** The PVC already persists across stages and survives pod scale-down. The OpenSpec CLI operates on the local filesystem. This is the natural location.

### Decision 3: Sequential single-pod stages, not separate pods per stage

All three stages (Plan, Execute, Verify) run sequentially in the same pod/workspace, not as separate pods. The sidecar manages stage transitions by starting/stopping agents within the same container.

**Alternatives considered:**
- Separate pods per stage → requires PVC sharing (ReadWriteMany), adds scheduling latency
- Separate Temporal child workflows per stage → over-engineered for sequential work

**Rationale:** The workspace PVC is ReadWriteOnce. All stages need the same workspace. Running sequentially in one pod avoids storage sharing complexity. The sidecar already manages agent process lifecycle — it just manages three sequential processes instead of one.

### Decision 4: Planning agent uses `openspec new change` + `/opsx:propose` pattern

The planning agent:
1. Runs `openspec new change "<run-id>"` to scaffold the change directory
2. Uses the `/opsx:propose` skill pattern to generate proposal, specs, and tasks
3. Runs `openspec validate --json` to ensure structural validity before proceeding

This reuses the exact same workflow that humans use with OpenSpec. The agent IS the human in the loop.

**Alternatives considered:**
- Generate a custom JSON spec format → loses OpenSpec ecosystem, can't use validate/archive
- Have the Temporal activity generate specs directly → loses LLM context, poor quality

**Rationale:** The planning agent has repo context, understands the codebase, and can write meaningful WHEN/THEN scenarios. The OpenSpec CLI validates its output. Same tools, same workflow, automated.

### Decision 5: Max 3 retries with failure context injection

On verification failure, Stage 2 retries with the verification report prepended to the agent's context. Max 3 retries, then the run is marked Failed with the full verification history.

**Rationale:** 3 retries balances reliability vs cost. The failure context helps the agent understand what went wrong. After 3 failures, a human needs to look at it.

### Decision 6: New orchestration mode "spec-driven"

Add `OrchestrationMode = "spec-driven"` alongside existing `single`, `auto`, `manual`. This mode activates the Plan → Execute → Verify pipeline. When `specContent` is provided with any mode, auto-upgrade to spec-driven.

**Rationale:** Orchestration mode is the existing mechanism for controlling run behavior. Adding a new mode is consistent and backwards-compatible.

### Decision 7: Sidecar stage management via StartAgent stage field

The sidecar's `StartAgent` RPC is extended with a `stage` field (`plan`, `execute`, `verify`). Each stage configures the pi-coding-agent differently:
- Plan: system prompt instructs spec generation via OpenSpec, limited tool set
- Execute: system prompt instructs implementation via `/opsx:apply`, full tool set
- Verify: system prompt instructs evaluation, read-only tools + exec for test commands

**Rationale:** Parameterizing the existing RPC is the minimal change. Each stage is an independent agent invocation with its own context window, enabling clean retries of individual stages.

## Risks / Trade-offs

- **Plan stage adds latency** (~30-60s for spec generation) → Acceptable for production reliability. Show "Planning..." phase in UI.
- **LLM judge is non-deterministic** → Mitigated by running OpenSpec validate + list + automated checks first (deterministic baseline). LLM only evaluates semantic criteria.
- **OpenSpec CLI dependency in sidecar image** → Adds ~50MB to sidecar image (Node.js already present). Pin version.
- **Retry loop can triple execution time** → Max 3 retries with TTL enforcement. Each retry gets failure context.
- **Spec quality depends on planning agent** → Bad spec → bad evaluation. Mitigated by `openspec validate --json` ensuring structural validity and by using a capable model for planning.
- **`openspec archive --yes` bypasses incomplete-task warning** → We check task completion explicitly via `openspec list --json` BEFORE attempting archive. Archive is the seal, not the check.

## Open Questions

- Should the verification LLM judge use the same model as the execution agent, or a different (potentially stronger) model?
- How should the web UI surface the retry history — inline in the log viewer, or as a separate "Attempts" tab?
