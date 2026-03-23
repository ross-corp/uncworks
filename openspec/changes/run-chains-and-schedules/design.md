## Context

UNCWORKS runs are currently fire-and-forget: each run is triggered manually, executes once, and has no relationship to other runs. There is no way to schedule recurring runs (weekly dependency updates, nightly code reviews) or chain runs where one depends on another's output (analyze first, then fix, then test). This change introduces four new CRDs with single-responsibility design: RunTemplate (what), Chain (order), Schedule (when), ChainRun (execution).

## Goals / Non-Goals

**Goals:**
- Extract reusable run configuration into RunTemplate, separate from scheduling and sequencing
- Define DAG-based chains of steps with fan-out, fan-in, context passing, and branch propagation
- Provide cron-based scheduling with concurrency policies mirroring Kubernetes CronJob semantics
- Track chain execution as a first-class resource with per-step status and Temporal durability
- Visualize chain execution as a DAG and provide schedule management UI

**Non-Goals:**
- Modifying the AgentRun CRD or its workflow (AgentRunWorkflow remains the execution primitive)
- Supporting event-based triggers (webhooks, PR events) — future work
- Supporting conditional branching (if/else in DAG) — future work
- Building a full workflow engine (Chains are simple DAGs, not Argo-style pipelines)

## Decisions

### 1. SRP Decomposition: four resources, four responsibilities

Each resource owns exactly one concern:

| Resource     | Responsibility        | Contains                                     | Does NOT contain          |
|-------------|----------------------|----------------------------------------------|---------------------------|
| RunTemplate | What to run           | prompt, repos, model, push/PR, projectRef     | scheduling, dependencies  |
| Chain       | Execution order       | steps[], dependsOn edges, contextFrom, branchFrom | scheduling, run state     |
| Schedule    | When to trigger       | cron expression, target ref, concurrency policy | execution logic, DAG      |
| ChainRun    | Execution tracking    | per-step status, agentRunRefs, timing          | configuration, scheduling |

This decomposition means a RunTemplate can be triggered manually, by a Schedule, or as a Chain step without any code duplication. A Chain can be triggered manually or by a Schedule without the Chain knowing about scheduling. A Schedule can target a RunTemplate or a Chain without knowing about DAGs.

### 2. Context passing: contextFrom and branchFrom

Chain steps need two kinds of context from parent steps:

**contextFrom** injects parent output into the child's prompt. When step B declares `contextFrom: ["A"]`, the chain executor reads step A's AgentRun status.logOutput (the persisted agent log, up to 1MB) and prepends a context section to step B's prompt:

```
## Context from step "A"
{truncated log output from step A's AgentRun, max 8000 chars}

## Your Task
{step B's original prompt from the RunTemplate}
```

For multiple parents (`contextFrom: ["A", "B"]`), each parent's output is included as a separate labeled section. The order follows the contextFrom array order.

**branchFrom** propagates git branches between steps. When step B declares `branchFrom: "A"`, the chain executor reads step A's AgentRun to determine the branch it pushed to (from the AgentRun's PR URL or a new status field tracking the pushed branch name). Step B's AgentRun is then created with `repos[].branch` set to step A's branch, so step B starts with step A's code changes.

branchFrom is limited to a single step reference (not an array) because merging multiple branches is complex and error-prone. If a fan-in step needs changes from multiple parents, it should use contextFrom for instructions and branchFrom from one parent, then the agent applies the other parent's changes manually.

### 3. Temporal integration: ChainRun as a Temporal workflow

The ChainRun controller does NOT manage execution directly. Instead, it delegates to a new Temporal workflow `ChainRunWorkflow` that:

1. **Reads the Chain spec** from the ChainRun's chainRef to get the step definitions and DAG edges.
2. **Performs topological sort** of steps to determine execution order and parallelism groups.
3. **Executes steps as child workflows**: each step spawns an `AgentRunWorkflow` child workflow. The child workflow ID is `{chainRunName}-step-{stepName}`.
4. **Manages fan-out**: steps whose dependencies are all satisfied are launched in parallel using `workflow.Go` goroutines.
5. **Manages fan-in**: after launching parallel steps, the workflow collects their results using `workflow.Future`. A step with multiple dependencies waits for all parent futures.
6. **Injects context**: before creating each step's AgentRun, the workflow queries completed parent AgentRuns for log output (contextFrom) and branch info (branchFrom), then enriches the child WorkflowInput.
7. **Updates ChainRun status**: after each step completes, the workflow executes an activity that patches the ChainRun status with the step's phase, agentRunRef, and timing.

```
ChainRunWorkflow
  |
  ├── topoSort(steps) → [[A], [B, C], [D]]
  |
  ├── level 0: launch AgentRunWorkflow("A")
  |     └── wait → Succeeded
  |
  ├── level 1: launch AgentRunWorkflow("B"), AgentRunWorkflow("C") in parallel
  |     ├── inject context from A into B (contextFrom)
  |     ├── set B's branch to A's branch (branchFrom)
  |     └── wait for both → Succeeded
  |
  └── level 2: launch AgentRunWorkflow("D")
        ├── inject context from B and C (contextFrom)
        └── wait → Succeeded
```

The workflow supports cancellation: when a cancel signal arrives, it cancels all running child workflows and marks remaining steps as Skipped.

The workflow uses a Temporal task queue `aot-chain-runs` separate from the agent run queue to avoid head-of-line blocking.

### 4. Schedule controller: ticker-based cron evaluation

The schedule controller is a standard Kubernetes controller that reconciles Schedule CRDs. Instead of using Temporal schedules (which would add a dependency on Temporal's schedule feature and complicate the control plane), the controller uses a simple ticker approach:

1. **On reconcile**: compute `nextFireTime` from the cron expression and the current time. Store it in the Schedule status.
2. **On tick** (RequeueAfter = time until nextFireTime, minimum 30s, maximum 60s): check if `now >= nextFireTime`.
3. **If firing**: check concurrency policy against active runs. If allowed, create AgentRun (for RunTemplate target) or ChainRun (for Chain target). Update lastFireTime, compute next nextFireTime.
4. **If suspended**: skip evaluation, requeue after 60s to check for unsuspend.

The controller uses `ctrl.Result{RequeueAfter: timeUntilNextFire}` to schedule itself efficiently. This avoids a global ticker goroutine and lets the controller-runtime manage scheduling.

Concurrency policy evaluation:
- **Forbid**: query active runs with label `aot.uncworks.io/schedule: {scheduleName}` and phase Running. If any exist, skip.
- **Replace**: query active runs, cancel them (send cancel signal via Temporal or update status), then create new run.
- **Allow**: create new run unconditionally.

### 5. DAG execution: topological sort and level-based parallelism

The chain executor uses Kahn's algorithm for topological sort:

1. Build an adjacency list and in-degree map from the Chain's step definitions.
2. Initialize a queue with all steps having in-degree 0 (root steps).
3. Process the queue level by level: all steps at the same level can execute in parallel.
4. For each level, launch child workflows for all steps in that level, then wait for all to complete before proceeding to the next level.

This level-based approach is simpler than a fully event-driven DAG executor but has a limitation: within a level, a step that finishes early cannot immediately unblock the next level. In practice this is acceptable because chain steps are agent runs that take minutes, so the overhead of waiting for the slowest step in a level is negligible compared to the total execution time.

The topological sort is performed once at workflow start and stored in the workflow's local state. If a step fails, all steps in subsequent levels that transitively depend on the failed step are marked Skipped. Steps in the same level that do not depend on the failed step continue executing.

### 6. Interaction with Projects

RunTemplates, Chains, and Schedules all support an optional `projectRef` field:

- **RunTemplate.spec.projectRef**: when set, empty fields (repos, modelTier, TTL, etc.) are resolved from the Project's defaults at trigger time. This is the same inheritance pattern used by AgentRunSpec.projectRef.
- **Chain.spec.projectRef**: informational grouping. All steps in a Chain share the same project context. The Chain's projectRef is propagated to ChainRun labels for filtering.
- **Schedule.spec.projectRef**: informational grouping. Used for filtering schedules by project in the UI.

Resolution happens at trigger time, not at CRD creation time, so changes to Project defaults are picked up by the next trigger.

### 7. CRD naming and API group

All four CRDs are in the `aot.uncworks.io` API group, `v1alpha1` version, consistent with AgentRun and Project:

- `runtemplates.aot.uncworks.io`
- `chains.aot.uncworks.io`
- `schedules.aot.uncworks.io`
- `chainruns.aot.uncworks.io`

Short names: `rt`, `chain`, `sched`, `cr` (avoiding collision with the built-in `cr` for ClusterRole — use `chainrun` as the preferred short name).

### 8. New Go types structure

New files in `api/v1alpha1/`:
- `runtemplate_types.go` — RunTemplateSpec, RunTemplateStatus, RunTemplate, RunTemplateList
- `chain_types.go` — ChainSpec, ChainStep, ChainStatus, Chain, ChainList
- `schedule_types.go` — ScheduleSpec, ScheduleStatus, Schedule, ScheduleList
- `chainrun_types.go` — ChainRunSpec, ChainRunStepStatus, ChainRunStatus, ChainRun, ChainRunList

Each file follows the same pattern as `types.go` (AgentRun) and `project_types.go` (Project): marker comments for kubebuilder, deepcopy generation via `+kubebuilder:object:root=true`, and an `init()` function registering with the SchemeBuilder.

### 9. AgentRun labels for chain/schedule tracking

When the chain executor or schedule controller creates an AgentRun, it sets labels to enable tracing back to the source:

```yaml
labels:
  aot.uncworks.io/chain-run: "{chainRunName}"
  aot.uncworks.io/chain-step: "{stepName}"
  aot.uncworks.io/schedule: "{scheduleName}"
  aot.uncworks.io/run-template: "{templateName}"
```

These labels enable:
- Filtering runs by chain run, schedule, or template in the UI
- Concurrency policy checks (find active runs for a schedule)
- Garbage collection (history limits delete runs by schedule label)

## Risks / Trade-offs

- **[Risk] Level-based parallelism is less optimal than event-driven DAG execution.** A step that finishes early in a level cannot immediately unblock the next level. Mitigation: agent runs take minutes, so the overhead of waiting for the slowest step in a level is negligible. Event-driven execution can be added later without changing the CRD schema.
- **[Risk] Schedule controller ticker drift.** The controller uses RequeueAfter which is not guaranteed to fire at the exact time. Mitigation: the controller checks `now >= nextFireTime` on each reconcile, so it self-corrects. Worst case is a few seconds of delay, which is acceptable for cron-granularity scheduling.
- **[Risk] Large log output in contextFrom.** AgentRun status.logOutput can be up to 1MB. Passing multiple parents' logs into a prompt could exceed LLM context limits. Mitigation: truncate each parent's log to 8000 characters in the context section. The agent can read full logs from the workspace if needed.
- **[Trade-off] Separate Temporal task queue for chains.** Using `aot-chain-runs` avoids chain workflows blocking agent run workflows, but requires the worker to register on both queues. This is a one-line change in the worker setup.
- **[Trade-off] No event-based triggers.** Schedules only support cron. Webhook/PR-event triggers are deferred to a future change. The Schedule CRD schema leaves room for a future `trigger` field alongside `cronExpression`.
- **[Trade-off] branchFrom limited to single parent.** Multi-branch merge is complex. A fan-in step that needs multiple parents' code changes should use contextFrom for instructions and branchFrom from one parent. This is a deliberate simplification.
