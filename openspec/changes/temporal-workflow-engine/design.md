## Context

AOT's agent lifecycle is currently managed by a K8s controller reconcile loop with PostgreSQL as the state store. This hand-rolled orchestration lacks durable execution guarantees -- if the controller restarts during a HITL wait (which can last hours), state must be carefully reconstructed from pod status and database records. Multi-agent workflows (`spawn_junior`) have no built-in parent-child lifecycle management, retry semantics, or saga-pattern compensation. The existing `brain/store.go` queue functionality is a bespoke reimplementation of concerns that Temporal handles natively.

Temporal.io provides durable workflow execution where lifecycle state survives any infrastructure failure, with native support for signals (HITL), child workflows (multi-agent), timers (TTL), and compensation (cleanup on failure). This change introduces Temporal as the orchestration engine while keeping the K8s controller as a thin CRD-to-workflow bridge.

## Goals / Non-Goals

**Goals:**
- Durable agent lifecycle execution that survives controller restarts, node failures, and network partitions.
- Native HITL support via Temporal signals, eliminating the need for polling or fragile direct-to-sidecar routing.
- Parent-child workflow semantics for `spawn_junior` multi-agent coordination.
- Timer-based TTL enforcement owned by the workflow, not the controller reconcile loop.
- Clean separation: K8s controller owns CRD watching and status sync; Temporal owns business logic and execution ordering.

**Non-Goals:**
- Replacing the sidecar gRPC protocol -- the RPC Gateway contract does not change.
- Bundling Temporal server inside AOT's Helm chart -- Temporal is an explicit external dependency.
- Migrating brain store metadata (agent output, trace IDs) into Temporal -- the brain retains non-workflow metadata.
- Supporting workflow versioning or migration strategies in the initial implementation.

## Decisions

### Responsibility split

The K8s controller becomes a thin bridge. It watches AgentRun CRDs, starts Temporal workflows, and syncs workflow state back to CRD status fields. All business logic -- lifecycle state machine, HITL signal handling, TTL timers, multi-agent child workflows, and compensation/saga cleanup -- moves into the Temporal workflow.

The controller reconcile loop changes to three cases:
1. **New CRD (no workflow ID annotation):** Start a Temporal workflow, annotate the CRD with the workflow ID.
2. **Existing CRD (has workflow ID):** Query the Temporal workflow state via the `get-state` query, sync the result to CRD status fields.
3. **CRD deletion:** Cancel the Temporal workflow.

### Workflow design

A single `AgentRunWorkflow` encapsulates the full agent lifecycle. Activities:
- `CreateAgentPod` -- creates the K8s pod with sidecars (uses controller-runtime client).
- `WaitForHydration` -- polls init container status until hydration completes.
- `StartAgent` -- gRPC call to the sidecar to begin agent execution.
- `ForwardHumanInput` -- gRPC call to sidecar to deliver human input.
- `GetAgentStatus` -- gRPC call to sidecar to check agent state.
- `StopAgent` -- gRPC call to sidecar to gracefully stop the agent.
- `CleanupPod` -- deletes the K8s pod.

Signals: `human-input` (delivers HITL input), `cancel` (requests graceful termination).
Queries: `get-state` (returns current phase, message, and pod name for controller status sync).

### Task queue

Single task queue: `aot-agent-runs`. All workers register all workflows and activities on this queue. This keeps the deployment model simple -- a single worker binary handles everything.

### Temporal worker binary

`cmd/temporal-worker/main.go` is a standalone binary that connects to the Temporal Frontend service and runs the worker. It needs access to a controller-runtime K8s client (for pod management activities) and the brain store (for metadata persistence). It is built alongside other binaries in `task build`.

### Connection configuration

Environment variables:
- `TEMPORAL_HOST` (default: `localhost:7233`) -- Temporal Frontend address.
- `TEMPORAL_NAMESPACE` (default: `default`) -- Temporal namespace.
- `TEMPORAL_TASK_QUEUE` (default: `aot-agent-runs`) -- task queue name.

### Dev story

`temporal-cli` is added to `devbox.json`. A new `task temporal:dev` target starts `temporal server start-dev` which runs a single-process Temporal server backed by SQLite with zero external dependencies. This gives developers a fully functional Temporal environment without Docker or Helm.

### Production deployment (k0s)

Temporal is deployed to k0s via the official `temporalio/helm-charts`. It shares the existing PostgreSQL instance but uses a separate database (`temporal`). Recommended shard count for small production: 512. Temporal is NOT bundled in AOT's Helm chart; it is deployed and managed independently.

### spawn_junior as child workflow

Instead of directly creating a CRD, `spawn_junior` triggers a child workflow of `AgentRunWorkflow`. The parent workflow can either await child completion (blocking) or fire-and-forget (non-blocking), depending on the agent's intent. Child workflow failures propagate to the parent via standard Temporal error handling.

### Brain store changes

Queue functionality in `brain/store.go` is replaced by Temporal task queues. The brain store retains responsibility for metadata storage (agent output, trace IDs, run history) that is not owned by Temporal. The queue-related code paths become dead code and should be removed.

## Risks / Trade-offs

- **Operational complexity:** Temporal is a significant new infrastructure dependency. It requires its own database, has its own failure modes, and needs monitoring. Mitigated by using `temporal-cli` for dev (zero-ops) and the official Helm chart for production (well-tested).
- **Latency on HITL path:** Human input now traverses client -> gRPC server -> Temporal signal -> workflow -> activity -> sidecar gRPC, adding one hop (the Temporal signal dispatch). In practice, Temporal signal delivery is sub-100ms, which is negligible for human interaction.
- **Controller-Temporal consistency:** The CRD status is eventually consistent with the Temporal workflow state. There is a window where the CRD status lags behind the actual workflow state, bounded by the controller reconcile interval (default 30s). This is acceptable because the CRD status is informational, not authoritative.
- **Single task queue:** All agent runs share one task queue, meaning a slow activity in one workflow could delay others if worker capacity is saturated. Mitigated by running multiple worker replicas and setting appropriate `MaxConcurrentActivityExecutionSize` on the worker.
- **Temporal server availability:** If Temporal is down, new agent runs cannot start and HITL signals cannot be delivered. Existing workflows that are mid-activity (e.g., pod is running) continue running -- they just cannot advance to the next step. This is strictly better than the current model where controller restarts lose in-flight state.
