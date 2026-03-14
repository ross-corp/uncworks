## Why

Agent run workspaces currently use emptyDir volumes and bare Pods. When a run completes, the Pod is deleted and everything is gone — logs, files, the entire workspace. The retention timer we added is a band-aid that wastes resources keeping idle Pods alive. Users can't inspect completed runs, debug agent output, or attach VS Code to pair-program with the agent. The observability features we built (log viewer, file explorer, shell) only work while the Pod exists — which is a narrow window.

We need the workspace to survive Pod deletion. We need compute to be on-demand (scale up when needed, scale down when not). And we need VS Code dev container support so users can attach a full IDE to any run — live or completed.

## What Changes

### Infrastructure: Deployment + PVC per Run
- **Replace bare Pods with Deployments** (replicas 0 or 1) — the Deployment is the run's compute identity, scalable on demand. Scale to 1 = agent runs or debug session. Scale to 0 = compute freed, data persists.
- **Replace emptyDir with PersistentVolumeClaims** — one PVC per run mounted at `/workspace`. Data survives Pod deletion. Logs written to PVC at `/workspace/.aot/logs/agent.log`.
- **Install local-path-provisioner** in k0s for PVC support on single-node clusters.

### Three-Layer Observability
- **Layer 1 (Live)**: While Deployment replicas=1 — stream logs from sidecar, exec for files/shell, full interactivity. Same as current.
- **Layer 2 (Archive)**: After scale to 0 — logs and files readable directly from PVC on disk. Same API endpoints, same UI tabs, seamless transition. No exec needed.
- **Layer 3 (Resurrect)**: "Debug Run" scales Deployment 0→1 with a debug entrypoint (shell, no agent). User gets shell access to the exact workspace state the agent left. Auto-expires after 30min idle.

### VS Code Dev Container Support
- **Generate `.devcontainer/devcontainer.json`** in workspace during hydration — describes the workspace for VS Code Remote Containers.
- **SSH server in debug pods** — enables VS Code Remote SSH attachment.
- **Pair programming**: attach VS Code to a running agent pod to work alongside the agent in real-time.

### API Changes
- **File/log endpoints detect pod state** — exec into pod when running, read from disk when scaled to 0.
- **`POST /api/v1/runs/{id}/debug`** — scales Deployment 0→1 in debug mode.
- **`DELETE /api/v1/runs/{id}/debug`** — scales back to 0.
- **Connection info endpoint** — returns SSH/devcontainer details for VS Code attachment.

### Workflow Changes
- **CreateAgentPod → CreateAgentDeployment** — creates Deployment + PVC instead of bare Pod.
- **CleanupPod → ScaleDown** — sets replicas=0 instead of deleting.
- **Log tee** — sidecar tees agent output to `/workspace/.aot/logs/agent.log` in addition to stdout.
- **Archive cleanup** — background process deletes Deployment + PVC after configurable retention (default 7 days).

### E2E Test Coverage
- Tests for PVC persistence after scale-down, file/log reading from disk, debug pod creation, VS Code connection info.
- Playwright tests for seamless transition between live and archived states.

## Capabilities

### New Capabilities
- `deployment-lifecycle`: Deployment + PVC per run replacing bare Pods — create, scale up/down, cleanup. Includes local-path-provisioner setup.
- `persistent-workspace`: PVC-backed workspace that survives Pod deletion — log tee to disk, file persistence, direct disk reads when Pod is gone.
- `debug-pod`: On-demand debug pod creation by scaling Deployment 0→1 in debug mode — shell access, auto-expiry, same workspace state.
- `devcontainer-support`: VS Code dev container integration — devcontainer.json generation, SSH server in debug pods, connection info API, pair programming with live agents.

### Modified Capabilities
<!-- No existing spec-level requirements change — the observability UI (tabs, log viewer, file explorer, shell) remains the same, only the backend data source changes -->

## Impact

- **Temporal activities** (`internal/temporal/activities.go`): Replace `BuildAgentPod` with `BuildAgentDeployment` creating Deployment + PVC. Replace `CleanupPod` with `ScaleDownDeployment`. Add `CreateDebugPod` activity.
- **Temporal workflow** (`internal/temporal/workflow.go`): Update to use new activities. Scale-down instead of delete on completion.
- **Sidecar** (`internal/sidecar/gateway.go`): Tee agent stdout/stderr to `/workspace/.aot/logs/agent.log`.
- **Hydrator** (`internal/hydration/hydrator.go`): Generate `.devcontainer/devcontainer.json` during workspace setup.
- **API server** (`internal/server/files.go`, `exec.go`): Detect pod state for file/log endpoints — exec when running, disk read when scaled down. New debug endpoint.
- **API server** (`cmd/apiserver/main.go`): Register debug and connection-info endpoints.
- **Web UI** (`web/src/components/AgentRunDetailPanel.tsx`): Replace "Pod expired" with "Debug Run" button. Show VS Code connection info.
- **Infrastructure**: Install local-path-provisioner. Update RBAC for Deployment/PVC management.
- **CRD types** (`api/v1alpha1/types.go`): Add `DeploymentName` to status. Remove `RetainPodMinutes` from spec (replaced by archive retention).
- **Proto** (`api.proto`): Update status fields for deployment name, archive state.
- **E2E tests** (`e2e/`, `web/e2e/`): Tests for PVC persistence, debug pod, file/log reading after scale-down.
- **Documentation**: Update AGENTS.md, README with new architecture.
