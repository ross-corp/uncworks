## Why

AOT's agent lifecycle is managed by a K8s controller reconcile loop with PostgreSQL as the state store. This hand-rolled orchestration lacks durable execution guarantees -- if the controller restarts during a HITL wait (which can last hours), state must be carefully reconstructed. Multi-agent workflows (`spawn_junior`) have no built-in parent-child lifecycle management, retry semantics, or saga-pattern compensation. Temporal.io provides durable workflow execution where the lifecycle state survives any infrastructure failure, with native support for signals (HITL), child workflows (multi-agent), timers (TTL), and compensation (cleanup on failure).

## What Changes

- **Add Temporal Go SDK**: Introduce `go.temporal.io/sdk` as a dependency. Define `AgentRunWorkflow` and associated activities.
- **Thin controller**: The K8s controller becomes a bridge -- it watches `AgentRun` CRDs and starts/cancels Temporal workflows. Pod creation, HITL signal handling, TTL enforcement, and multi-agent coordination move into the Temporal workflow.
- **Temporal worker binary**: New `cmd/temporal-worker/` binary that registers workflows and activities, connects to the Temporal Frontend service.
- **Durable HITL**: `SendHumanInput` gRPC RPC sends a Temporal Signal instead of directly routing to the sidecar. The workflow receives the signal and forwards it via an activity.
- **Durable multi-agent**: `spawn_junior` triggers a Temporal child workflow instead of directly creating a CRD. Parent workflow can await child completion.
- **Temporal as explicit dependency**: Not bundled in AOT's Helm chart. Documented as a required external service with connection via `TEMPORAL_HOST` and `TEMPORAL_NAMESPACE` env vars.
- **Temporal dev server for local development**: Add `temporal-cli` to `devbox.json` for `temporal server start-dev` (single binary, SQLite, zero deps).
- **k0s deployment option**: Document deploying the official `temporalio/helm-charts` to k0s for a full cluster setup.

## Capabilities

### New Capabilities
- `temporal-workflows`: Temporal workflow definitions for AgentRun lifecycle, including activities for pod management, sidecar communication, and state synchronization.
- `temporal-deployment`: Temporal server deployment configuration for k0s (Helm) and connection configuration for external/cloud Temporal instances.
- `temporal-worker`: Standalone worker binary that executes workflow and activity code, connects to Temporal Frontend service.

### Modified Capabilities
- `k8s-orchestrator`: Controller reconcile logic reduced to CRD→workflow bridge and workflow-state→CRD-status sync. Pod creation logic moved to Temporal activities but still uses controller-runtime K8s client.
- `agent-harness`: HITL input routing changes from direct sidecar call to Temporal signal → activity → sidecar call. No proto changes required.
- `client-interfaces`: `SendHumanInput` and `CancelAgentRun` RPCs now route through Temporal signals instead of direct pod communication.

## Impact

- **`internal/controller/`**: `agentrun_controller.go` dramatically simplified. `multi_agent.go` logic moves to child workflow.
- **`internal/server/`**: `SendHumanInput` handler sends Temporal Signal. `CancelAgentRun` handler cancels Temporal workflow.
- **`internal/brain/`**: Queue functionality may be replaced by Temporal task queues. State store remains for metadata not owned by Temporal.
- **`cmd/temporal-worker/`**: New binary.
- **`go.mod`**: Add `go.temporal.io/sdk`.
- **`devbox.json`**: Add `temporal-cli`.
- **`Taskfile.yml`**: New targets: `temporal:dev` (start dev server), `build` updated to include temporal-worker binary.
- **Infrastructure**: Requires Temporal server accessible at `TEMPORAL_HOST:7233`. For k0s dev, deployed via Helm. Temporal needs its own PostgreSQL database (can share the same PostgreSQL instance, separate database).
