## Why

UNCWORKS runs are fire-and-forget — you trigger one manually and it runs once. There is no way to schedule recurring runs (weekly dependency updates, nightly code reviews) or to chain runs together where one depends on another's output (analyze first, then fix, then test). These are table-stakes features for a development platform. Without them, every recurring task requires a human to remember to trigger it, and multi-step workflows require manually orchestrating each step.

## What Changes

Four new resources, each with a single responsibility:

### RunTemplate — "what to run"
A reusable, named run configuration extracted from AgentRunSpec. References a project, prompt, model, orchestration mode, and push/PR settings. No scheduling or dependency logic. Can be triggered manually, by a schedule, or as a chain step.

### Chain — "in what order"
A DAG of steps where each step references a RunTemplate and declares dependencies. Supports fan-out (parallel steps after a shared dependency), fan-in (step waits for multiple dependencies), and context passing between steps (inject parent output into child prompt, clone from parent branch). No scheduling logic — chains are triggered, not timed.

### Schedule — "when to run"
A cron expression that triggers either a single RunTemplate or a Chain on a recurring basis. Supports suspend/resume, concurrency policy (Forbid/Replace/Allow), and history limits. Mirrors Kubernetes CronJob semantics.

### ChainRun — "track execution"
An instance of a Chain execution. Tracks per-step status (pending/running/succeeded/failed), creates AgentRuns for unblocked steps, injects context between steps. Backed by a Temporal workflow for durable execution.

## Capabilities

### New Capabilities
- `run-template`: RunTemplate CRD for reusable named run configurations with CRUD API and UI management
- `chain-definition`: Chain CRD defining step DAGs with dependsOn edges, contextFrom injection, and branchFrom propagation
- `chain-execution`: ChainRun controller that orchestrates step execution via Temporal, creates AgentRuns for unblocked steps, passes context between steps
- `schedule-cron`: Schedule CRD with cron expressions, concurrency policy, suspend/resume, referencing Chains or RunTemplates
- `chain-ui`: Chain run visualization as a vertical DAG with live status per node, schedule list view, and "trigger chain" button

### Modified Capabilities
- None

## Impact

- **New CRDs**: `RunTemplate`, `Chain`, `Schedule`, `ChainRun` with controllers
- **New**: `internal/controller/schedule_controller.go` — watches Schedule CRDs, creates ChainRuns or AgentRuns on cron tick
- **New**: `internal/controller/chain_controller.go` — watches ChainRun CRDs, manages step execution DAG
- **New**: `internal/temporal/workflow_chain.go` — Temporal workflow for chain execution (child workflows per step)
- **Modified**: `internal/server/` — REST endpoints for RunTemplate, Chain, Schedule, ChainRun CRUD
- **New**: `web/src/views/ScheduleListView.tsx` — schedule management UI
- **New**: `web/src/views/ChainRunDetailView.tsx` — DAG visualization of chain execution
- **Modified**: `web/src/views/RunListView.tsx` — show chain context (which chain/step a run belongs to)
- **Modified**: `web/src/AppNew.tsx` — new routes for schedules and chain runs
- **Modified**: `deploy/crds/` — new CRD YAML files
- **Modified**: `deploy/helm/aot/` — new Helm templates and RBAC for new CRDs
