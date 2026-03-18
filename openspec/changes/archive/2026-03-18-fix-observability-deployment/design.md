## Context

The AOT system has four Docker images — controlplane, sidecar, hydration, and agent-base — that must all be in sync for observability features to work. The controlplane was recently rebuilt after code changes, but the other three images are stale and missing critical functionality (log tee, trace collection, `.aot` directory creation). Additionally, the API server pod cannot access PVC data because the local-path-provisioner stores volumes on the host filesystem at `/opt/local-path-provisioner/`, and the apiserver pod has no mount for that path.

## Goals / Non-Goals

**Goals:**
- API server can read PVC-backed data (logs, files, traces) from any completed or running run
- All four Docker images are rebuilt from current source and deployed to k0s
- A single Taskfile command rebuilds, imports, and restarts everything
- Logs stream in real-time in the Logs tab
- Files are browsable in the Files tab after run completion
- Shell/Debug Run works for running and completed runs
- Traces appear in the traces view

**Non-Goals:**
- Changing the storage backend from local-path-provisioner to something else
- Adding new observability features beyond what is already coded
- Setting up external monitoring (Prometheus, Grafana, etc.)
- Modifying the sidecar or hydration code (only rebuilding the images with existing code)

## Decisions

### 1. Mount `/opt/local-path-provisioner/` as a hostPath volume in the apiserver pod

The local-path-provisioner creates PVC-backed directories under `/opt/local-path-provisioner/` on the host. The API server needs read access to these directories to serve file/log/trace data. A hostPath volume mount is the simplest and most direct solution.

**Alternative considered:** Use a shared NFS volume or a sidecar proxy. Rejected — adds unnecessary complexity for a single-node k0s deployment. The local-path-provisioner is already host-bound by design.

**Mount mode:** ReadOnly. The API server only reads PVC data; it never writes.

### 2. Rebuild ALL images, not just the ones we know are stale

Even though only sidecar and hydration are known to be stale, rebuilding all four images (controlplane, sidecar, hydration, agent-base) ensures consistency. The cost is a few extra minutes of build time. The risk of a partially-rebuilt system is much worse.

### 3. Add a `deploy:all` Taskfile task

The current workflow requires manually running separate build, import, and restart commands for each image. This is error-prone and is the root cause of the current staleness problem. A single `deploy:all` task prevents future drift.

The task will:
1. Build all Docker images (`docker build` for each)
2. Import all images into k0s (`k0s ctr images import` for each)
3. Rollout restart all deployments (`kubectl rollout restart deployment/...`)

### 4. Verify each feature end-to-end, not just "pods are running"

Previous deployments verified that pods started successfully but did not verify that the features actually worked. This change explicitly requires creating a test run and checking each observability feature against it.

## Risks / Trade-offs

- **[Risk] hostPath mount ties the apiserver to a specific node** — Acceptable for single-node k0s. If the system moves to multi-node, this must be revisited with a shared storage solution.
- **[Risk] ReadOnly mount may not work if the API server needs to write temp files** — Unlikely given the current code only reads. If discovered during testing, the mount can be changed to ReadWrite.
- **[Risk] Rebuilding all images may introduce unrelated regressions** — Mitigated by end-to-end verification of each feature. If a regression appears, it indicates code that was never tested in-cluster.
- **[Trade-off] `deploy:all` rebuilds everything even if only one image changed** — Acceptable. Correctness over speed. A future optimization could add per-image tasks.
