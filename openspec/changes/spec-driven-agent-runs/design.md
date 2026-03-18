## Context

AOT currently runs agents as a single-stage pipeline: hydrate workspace → start agent → poll until exit → report exit code as success/failure. The agent has no structured acceptance criteria, no verification step, and no retry capability. The platform reports "Succeeded" for any clean exit regardless of outcome quality.

The OpenSpec CLI (`openspec`) is installed globally and provides machine-parseable JSON output for validation, status tracking, and change management. It uses a BDD-style spec format with WHEN/THEN scenarios that can serve as both human-readable documentation and machine-evaluatable acceptance criteria.

The Temporal workflow engine already manages the agent lifecycle with activities, signals, and retries — adding new stages is a natural extension of the existing architecture.

## Goals / Non-Goals

**Goals:**
- Agent run success is determined by spec verification, not process exit code
- Multi-stage pipeline (Plan → Execute → Verify) with automatic retry on verification failure
- User input of any fidelity (vague prompt to full spec) is normalized into a structured, evaluatable OpenSpec change
- Automated checks (tests pass, files exist, builds compile) combined with LLM judge for semantic evaluation
- Verification results exposed through API and web UI
- Each run produces an OpenSpec change as its documentation artifact

**Non-Goals:**
- Real-time streaming of planning/verification stage output (future enhancement — stages run sequentially, output captured in structured logs)
- User-editable specs mid-run (the plan stage output is final; user can cancel and re-run with more specific input)
- Parallel execution of plan + execute stages (strict sequential pipeline)
- Custom verification scripts provided by the user (automated checks derive from spec scenarios only)
- Replacing the existing single-stage mode for simple prompts (spec-driven mode is opt-in via orchestration mode or auto-detected for spec content)

## Decisions

### Decision 1: OpenSpec change per run, on workspace PVC

Each run creates an OpenSpec change directory at `/workspace/.openspec/changes/<run-id>/` on the workspace PVC. This collocates the spec artifacts with the code changes, making them available to all stages (plan, execute, verify) without cross-pod communication.

**Alternatives considered:**
- Store specs in K8s ConfigMaps → too large, 1MB limit, poor tooling
- Store in PostgreSQL brain → adds DB dependency to critical path, harder to debug
- Store in etcd via CRD annotations → size limits, not designed for documents

**Rationale:** The PVC already persists across stages and survives pod scale-down. The OpenSpec CLI operates on the local filesystem. This is the natural location.

### Decision 2: Sequential single-pod stages, not separate pods per stage

All three stages (Plan, Execute, Verify) run sequentially in the same pod/workspace, not as separate pods. The sidecar manages stage transitions by starting/stopping agents within the same container.

**Alternatives considered:**
- Separate pods per stage → requires PVC sharing (ReadWriteMany), adds scheduling latency, complex orchestration
- Separate Temporal child workflows per stage → over-engineered, adds workflow overhead for sequential work

**Rationale:** The workspace PVC is ReadWriteOnce. All stages need the same workspace. Running sequentially in one pod avoids storage sharing complexity and keeps the workflow simple. The sidecar already manages agent process lifecycle — it just manages three sequential processes instead of one.

### Decision 3: OpenSpec CLI for validation, LLM for semantic evaluation

Stage 3 uses a two-tier evaluation:
1. **Automated tier**: `openspec validate --json` for spec structure, `openspec list --json` for task completion, plus exec of commands from spec scenarios (test/build commands).
2. **LLM tier**: A separate agent call with the spec + git diff + agent log, asked to evaluate semantic WHEN/THEN criteria that can't be checked mechanically.

**Alternatives considered:**
- LLM-only evaluation → non-deterministic, expensive, can't catch compilation failures
- Automated-only evaluation → can't verify semantic intent ("does the auth middleware actually validate JWT tokens?")

**Rationale:** Hybrid gives deterministic baseline (tests pass, files exist) with semantic judgment for what machines can't check. Automated checks run first — if they fail, no LLM call needed (saves cost/time).

### Decision 4: Max 3 retries with failure context injection

On verification failure, Stage 2 retries with the verification report prepended to the agent's context. Max 3 retries, then the run is marked Failed with the full verification history.

**Alternatives considered:**
- No retries → too fragile, agents often need 2-3 attempts
- Unlimited retries → runaway cost, infinite loops
- Human-in-the-loop before retry → too slow for automated pipelines

**Rationale:** 3 retries balances reliability vs cost. The failure context helps the agent understand what went wrong. After 3 failures, a human needs to look at it.

### Decision 5: New orchestration mode "spec-driven"

Add `OrchestrationMode = "spec-driven"` alongside existing `single`, `auto`, `manual`. This mode activates the Plan → Execute → Verify pipeline. When `specContent` is provided with any mode, auto-upgrade to spec-driven.

**Alternatives considered:**
- Replace single mode entirely → breaks existing simple-prompt use cases
- Make it an API flag, not an orchestration mode → inconsistent with existing model

**Rationale:** Orchestration mode is the existing mechanism for controlling run behavior. Adding a new mode is consistent and backwards-compatible.

### Decision 6: Sidecar stage management via StartAgent variants

The sidecar's `StartAgent` RPC is extended with a `stage` field (`plan`, `execute`, `verify`). Each stage configures the pi-coding-agent differently:
- Plan: system prompt instructs `/opsx:propose`, limited tool set
- Execute: system prompt instructs `/opsx:apply`, full tool set
- Verify: system prompt instructs evaluation, read-only tool set + exec for test commands

The Temporal workflow calls `StartAgent` three times sequentially, waiting for each to complete before starting the next.

**Alternatives considered:**
- Three separate sidecar RPCs (StartPlanAgent, StartExecuteAgent, StartVerifyAgent) → duplicative, harder to extend
- Single long-running agent that handles all stages → model context window limits, can't retry individual stages

**Rationale:** Parameterizing the existing RPC is the minimal change. Each stage is an independent agent invocation with its own context window, enabling clean retries.

## Risks / Trade-offs

- **Plan stage adds latency** (~30-60s for spec generation before execution starts) → Acceptable for production reliability. Show "Planning..." phase in UI so users know work is happening.
- **LLM judge is non-deterministic** → Mitigated by running automated checks first (deterministic baseline). LLM only evaluates semantic criteria that can't be automated.
- **OpenSpec CLI dependency in sidecar image** → Adds ~50MB to sidecar image (Node.js already present). Pinned version avoids drift.
- **Retry loop can triple execution time** → Max 3 retries with TTL enforcement. Each retry gets the failure context, making it likely to succeed faster. Users see retry count in UI.
- **Spec quality depends on planning agent** → A bad spec leads to bad evaluation. Mitigated by `openspec validate --json` ensuring structural validity and by using a capable model for planning (not the 0.5b toy model).

## Open Questions

- Should the verification LLM judge use the same model as the execution agent, or a different (potentially stronger) model?
- Should we expose a "skip verification" option for users who want the old single-stage behavior on spec-driven runs?
- How should the web UI surface the retry history — inline in the log viewer, or as a separate "Attempts" tab?
