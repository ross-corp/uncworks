## Why

Completed agent runs retain their Deployment (replicas=0) and PVC indefinitely. Over time this fills the cluster with stale K8s resources — hundreds of zero-replica Deployments and unused PVCs that consume etcd storage and clutter `kubectl get` output. There is no automated cleanup; operators must manually delete resources for old runs.

## What Changes

- **Archive cleanup loop**: The controller gains a `cleanupExpiredRuns` method that periodically scans terminal AgentRuns whose `status.completedAt` is older than a configurable retention period (default 7 days). For each expired run it deletes the associated Deployment and PVC, then annotates the CRD as archived.
- **Configurable retention**: `AOT_RETENTION_DAYS` environment variable controls the retention period. Helm chart exposes `controller.retentionDays`.
- **CRD preservation**: The AgentRun CRD itself is never deleted — run history is preserved for auditing and dashboards.

## Capabilities

### New Capabilities
- `archive-cleanup`: Automated deletion of Deployments and PVCs for runs completed beyond the retention period.

## Impact

- `internal/controller/agentrun_controller.go` — new cleanup method, periodic invocation
- `cmd/controller/main.go` — read `AOT_RETENTION_DAYS` env var, pass to reconciler
- `deploy/helm/aot/templates/controller.yaml` — add env var
- `deploy/helm/aot/values.yaml` — add `controller.retentionDays` default
