## Context

Completed agent runs leave behind a Deployment (replicas=0) and a PVC that are never cleaned up. The existing TODO in `agentrun_controller.go` notes this should be addressed via either a CronJob or a controller reconciliation pass. We choose the controller approach because it reuses existing RBAC and avoids deploying a separate CronJob.

## Goals / Non-Goals

**Goals:**
- Automatically delete Deployments and PVCs for runs older than the retention period
- Make the retention period configurable via env var
- Preserve the AgentRun CRD for history

**Non-Goals:**
- Deleting the AgentRun CRD itself
- Archiving logs or artifacts to external storage before cleanup
- Adding a separate CronJob resource

## Decisions

### 1. Cleanup runs inside the controller manager via a background goroutine

The controller manager starts a `Runnable` that runs the cleanup loop every 5 minutes. This avoids modifying the per-resource Reconcile method with unrelated list-all logic, and leverages the manager's lifecycle (start/stop with leader election).

**Alternative considered:** Run cleanup inside `Reconcile()` on every terminal-state reconciliation. Rejected — cleanup is a cluster-wide scan, not specific to a single resource event.

### 2. Annotate archived runs instead of adding a status field

We use annotation `aot.uncworks.io/archived: "true"` rather than a new `status.archived` field. This avoids a CRD schema change and deepcopy regeneration while still being queryable.

### 3. PVC naming convention

PVCs are named `workspace-<agentrun-name>` based on the existing convention in the Temporal activities. The cleanup method uses this pattern to find and delete the PVC.
