## 1. Temporal SDK and Tooling Setup

- [x] 1.1 Add `go.temporal.io/sdk` to `go.mod` — v1.41.0
- [x] 1.2 Add `temporal-cli` to `devbox.json` — already present
- [x] 1.3 Add `task temporal:dev` target to `Taskfile.yml` that runs `temporal server start-dev --db-filename .temporal.db`
- [x] 1.4 Add `helm` to `devbox.json` (if not already present) — already present as kubernetes-helm@latest

## 2. Workflow and Activity Definitions

- [ ] 2.1 Create `internal/temporal/` package directory
- [ ] 2.2 Implement `AgentRunWorkflow` in `internal/temporal/workflow.go`: accepts `AgentRunSpec`, orchestrates full lifecycle
- [ ] 2.3 Implement `CreateAgentPod` activity in `internal/temporal/activities.go`: creates agent pod via controller-runtime K8s client (extract pod spec building from controller)
- [ ] 2.4 Implement `WaitForHydration` activity: polls pod init-container status until complete
- [ ] 2.5 Implement `StartAgent` activity: calls sidecar `StartAgent` gRPC RPC
- [ ] 2.6 Implement `GetAgentStatus` activity: calls sidecar `GetStatus` gRPC RPC
- [ ] 2.7 Implement `ForwardHumanInput` activity: calls sidecar `SendInput` gRPC RPC
- [ ] 2.8 Implement `StopAgent` activity: calls sidecar `StopAgent` gRPC RPC
- [ ] 2.9 Implement `CleanupPod` activity: deletes agent pod via K8s client
- [ ] 2.10 Add "human-input" signal handler to workflow: receives input string, calls `ForwardHumanInput` activity
- [ ] 2.11 Add "cancel" signal handler to workflow: calls `StopAgent` then `CleanupPod`
- [ ] 2.12 Add TTL enforcement via `workflow.NewTimer`: on expiry, calls `StopAgent` then `CleanupPod`
- [ ] 2.13 Add "get-state" query handler to workflow: returns current phase, message, pod name
- [ ] 2.14 Implement agent status polling loop: periodic `GetAgentStatus` checks for completion/failure
- [ ] 2.15 Add compensation logic: `CleanupPod` deferred to execute on any workflow failure

## 3. Child Workflows (spawn_junior)

- [ ] 3.1 Implement `SpawnJuniorWorkflow` as a child workflow of `AgentRunWorkflow` in `internal/temporal/workflow.go`
- [ ] 3.2 Add spawn_junior trigger: when agent sidecar notifies of spawn_junior tool call, start child workflow
- [ ] 3.3 Support both blocking (await child completion) and fire-and-forget child workflow modes
- [ ] 3.4 Propagate parent context to child: repo, branch, image, backend

## 4. Temporal Worker Binary

- [ ] 4.1 Create `cmd/temporal-worker/main.go`: connects to Temporal Frontend, registers workflows and activities
- [ ] 4.2 Read `TEMPORAL_HOST` (default: `localhost:7233`), `TEMPORAL_NAMESPACE` (default: `default`), `TEMPORAL_TASK_QUEUE` (default: `aot-agent-runs`) from environment
- [ ] 4.3 Initialize controller-runtime K8s client for pod management activities
- [ ] 4.4 Initialize brain store connection for metadata persistence
- [ ] 4.5 Register `AgentRunWorkflow` and all activities with the worker
- [ ] 4.6 Add graceful shutdown on SIGINT/SIGTERM via `worker.InterruptCh()`
- [ ] 4.7 Add workflow/activity execution logging
- [ ] 4.8 Add `temporal-worker` to `task build` targets in `Taskfile.yml`

## 5. Controller Simplification

- [ ] 5.1 Add Temporal client initialization to controller setup (reads `TEMPORAL_HOST`, `TEMPORAL_NAMESPACE`)
- [ ] 5.2 Refactor `Reconcile()`: on new AgentRun (no `aot.uncworks.io/workflow-id` annotation), start Temporal workflow and annotate CRD
- [ ] 5.3 Refactor `Reconcile()`: on existing AgentRun (has workflow ID), query Temporal workflow state and sync to CRD status
- [ ] 5.4 Refactor `Reconcile()`: on AgentRun deletion, cancel Temporal workflow
- [ ] 5.5 Extract pod spec building logic from controller into shared function (used by `CreateAgentPod` activity)
- [ ] 5.6 Remove direct pod creation/deletion from controller reconcile loop
- [ ] 5.7 Remove TTL checking logic from controller (now handled by workflow timer)
- [ ] 5.8 Remove direct HITL routing from controller (now handled by Temporal signal)

## 6. API Server Updates

- [ ] 6.1 Add Temporal client to API server initialization
- [ ] 6.2 Update `SendHumanInput` handler: send Temporal "human-input" signal to workflow instead of direct sidecar call
- [ ] 6.3 Update `CancelAgentRun` handler: cancel Temporal workflow instead of directly deleting pod
- [ ] 6.4 Update `GetAgentRun` handler: optionally query Temporal workflow state for real-time status

## 7. Brain Store Updates

- [ ] 7.1 Remove queue-related functions from `internal/brain/store.go` (replaced by Temporal task queues)
- [ ] 7.2 Keep metadata storage functions (agent output, trace IDs, completion records)
- [ ] 7.3 Update brain store tests to reflect removed queue functionality

## 8. Deployment Documentation

- [ ] 8.1 Create `deploy/temporal/` directory with deployment documentation
- [ ] 8.2 Document deploying `temporalio/helm-charts` to k0s with PostgreSQL backend
- [ ] 8.3 Document Temporal database setup: separate `temporal` and `temporal_visibility` databases on shared PostgreSQL
- [ ] 8.4 Document shard count recommendation (512 for production)
- [ ] 8.5 Document `TEMPORAL_HOST`, `TEMPORAL_NAMESPACE`, `TEMPORAL_TASK_QUEUE` environment variables
- [ ] 8.6 Update docs/user-guide.md with Temporal architecture section

## 9. Verification

- [ ] 9.1 Run `temporal server start-dev` and verify worker connects
- [ ] 9.2 Create an AgentRun CRD and verify Temporal workflow starts
- [ ] 9.3 Verify HITL flow: agent calls ask_human → workflow pauses → SendHumanInput signal → workflow resumes
- [ ] 9.4 Verify cancel flow: CancelAgentRun → workflow cancelled → pod cleaned up
- [ ] 9.5 Verify TTL: create AgentRun with short TTL → workflow times out → pod cleaned up
- [ ] 9.6 Run `task test:go` -- all Go tests pass
- [ ] 9.7 Verify controller correctly syncs Temporal workflow state to CRD status
