## Context

Agent runs currently create bare Pods with emptyDir volumes. The Temporal workflow's `defer` block deletes the Pod on completion, destroying all workspace data. The observability features (log viewer, file explorer, shell) we built depend on the Pod existing — they use K8s exec and container logs. A retention timer delays cleanup but wastes resources and doesn't scale.

The k0s cluster has no storage provisioner. PVCs don't work yet. The API server, controller, and worker run inside the cluster. The sidecar captures agent stdout/stderr via pipe and has a `StreamOutput` RPC.

## Goals / Non-Goals

**Goals:**
- Workspace data (files, logs) persists after Pod deletion — seamless transition
- Compute is on-demand — scale Deployment 0→1 when needed, 0 when idle
- "Debug Run" brings back a shell into the exact workspace state the agent left
- VS Code Remote Containers can attach to live or debug pods
- Same observability UI (tabs, log viewer, file explorer, shell) works in all states
- E2E tests validate the full lifecycle including post-completion access
- Architecture is simple: 4 K8s objects per run (CRD, PVC, Deployment, Temporal Workflow)

**Non-Goals:**
- Multi-node PVC storage (local-path is single-node; S3 backend is future)
- Real-time collaborative editing between multiple VS Code users
- Workspace snapshots or version history
- Running multiple agents simultaneously in the same workspace

## Decisions

### 1. Deployment (replicas 0/1) instead of bare Pod

**Decision**: Each agent run creates a Deployment with `replicas: 1`. The Deployment spec is identical to the current Pod spec (init container, agent container, sidecar). On completion, the workflow sets `replicas: 0` instead of deleting. For debug mode, the API server patches `replicas: 1` with a modified command (shell instead of agent).

**Rationale**: Deployments provide declarative scaling. Scale to 0 = compute freed. Scale to 1 = compute back. The Deployment object persists as the run's compute identity. No custom "resurrect" logic — just a replicas patch. K8s handles Pod scheduling, restart, and lifecycle.

**Alternative considered**: Jobs — rejected because completed Jobs can't be "resumed." StatefulSets — rejected as overkill for a single replica.

### 2. One PVC per run via local-path-provisioner

**Decision**: Install Rancher's local-path-provisioner in k0s. Each run gets a PVC (`aot-ws-{run-id}`, 2Gi, `ReadWriteOnce`). The PVC is mounted at `/workspace` in all containers (init, agent, sidecar). Data persists at `/opt/local-path-provisioner/` on the host.

**Rationale**: PVCs are the K8s-native way to persist data across Pod restarts. local-path-provisioner is the simplest provisioner for single-node — no CSI driver, no cloud integration, just host directories. The API server can read PVC data directly from the host path when the Pod is scaled to 0.

**Alternative considered**: hostPath volumes — simpler but no quota management, no K8s lifecycle integration, fragile path management. S3/MinIO — overkill for local dev.

### 3. Sidecar tees logs to PVC

**Decision**: The sidecar's `startAgentProcess` function tees agent stdout/stderr to `/workspace/.aot/logs/agent.log` using `io.TeeReader` in addition to the existing pipe capture. The log file is on the PVC and persists after Pod deletion.

**Rationale**: No separate log collection activity needed. The log file grows as the agent runs. After Pod deletion, the API server reads the log file directly from the PVC's host path. Same content, zero additional infrastructure.

### 4. File/log API endpoints: exec when running, disk when not

**Decision**: The file and log API endpoints check whether the run's Deployment has `replicas > 0`:
- **Pod running**: exec into pod (current behavior, fast)
- **Pod scaled to 0**: read from PVC host path (`/opt/local-path-provisioner/pvc-{uid}/...`)

The PVC's host path is discoverable by reading the PV's `spec.hostPath.path` from the K8s API.

**Rationale**: Seamless API — same endpoints, same response format. The UI doesn't need to know whether it's talking to a live Pod or reading from disk. The transition is invisible.

### 5. Debug pod: scale Deployment 0→1 with debug entrypoint

**Decision**: `POST /api/v1/runs/{id}/debug` patches the Deployment:
- Sets `replicas: 1`
- Adds annotation `aot.uncworks.io/mode: debug`
- The sidecar checks this annotation on startup — if `debug`, it skips agent launch and just serves the RPC gateway (shell access via exec).

`DELETE /api/v1/runs/{id}/debug` sets `replicas: 0`.

An idle timeout controller scales debug pods to 0 after 30 minutes of no WebSocket/exec activity.

**Rationale**: Reuses the existing Deployment. No new Pod spec. The sidecar already handles shell access. The debug annotation is the simplest way to distinguish "run agent" from "just provide shell."

### 6. devcontainer.json generation

**Decision**: The hydrator generates `/workspace/.devcontainer/devcontainer.json` during workspace setup:

```json
{
  "name": "aot-run-{id}",
  "image": "aot-agent:local",
  "workspaceFolder": "/workspace",
  "postStartCommand": "devbox install",
  "remoteUser": "root",
  "forwardPorts": [50052]
}
```

For VS Code attachment, users use `kubectl port-forward` to the pod's SSH port, then connect via Remote-SSH. The API server provides connection info at `GET /api/v1/runs/{id}/connect`.

**Rationale**: devcontainer.json is the standard. VS Code reads it to configure the remote environment. The image is the same agent base image. `postStartCommand` ensures devbox packages are available.

### 7. Archive cleanup: configurable retention

**Decision**: A background goroutine in the controller (or a CronJob) scans for runs where `completedAt` is older than the retention period (default 7 days). It deletes the Deployment and PVC. The CRD remains with status `phase: Archived`.

Before deletion, the last 1MB of `agent.log` is stored on the CRD status `logOutput` field as a permanent record. Files are gone after archival.

**Rationale**: Disk space management. 7-day default is generous for debugging. The CRD and log summary persist indefinitely for audit.

### 8. E2E test strategy

**Go E2E tests:**
- Create run → verify PVC created
- Wait for completion → verify Deployment replicas=0, PVC still exists
- Read files from disk endpoint → verify workspace contents
- Read logs from disk endpoint → verify agent output
- POST /debug → verify Deployment scales to 1
- Exec into debug pod → verify shell works
- DELETE /debug → verify scales back to 0

**Playwright tests:**
- Completed run → Logs tab shows content (from disk)
- Completed run → Files tab shows tree (from disk)
- Completed run → Shell tab shows "Debug Run" button
- Click "Debug Run" → Shell tab activates with terminal
- Running run → all tabs work as before

## Risks / Trade-offs

**Disk usage** — PVCs persist until archived. 2Gi × 100 runs = 200Gi before cleanup. → Mitigation: 7-day default retention, configurable, cleanup runs daily.

**local-path-provisioner single-node** — Data is local to one node. → Mitigation: Acceptable for dev/single-node. Multi-node would need a distributed storage backend (future).

**Direct host path reading** — API server reads PVC data by discovering the host path from the PV object. This couples to local-path-provisioner's storage layout. → Mitigation: Abstracted behind a helper function. Easy to swap for a different read strategy.

**Debug pod idle detection** — Detecting "no activity" requires tracking WebSocket/exec connections. → Mitigation: Simple approach — set a 30-minute timer on debug pod creation, reset on each connection. If timer expires, scale to 0.

**Migration from current bare-pod approach** — Existing runs use bare Pods and emptyDir. → Mitigation: New architecture applies to new runs only. Existing runs continue with current behavior until they're cleaned up.
