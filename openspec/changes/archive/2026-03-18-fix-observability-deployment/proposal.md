## Why

The observability features (logs, files, shell, traces) were implemented in code but do not work in the deployed k0s cluster. Five root causes prevent them from functioning:

1. **API server can't read PVC data**: The API server runs inside k0s but the local-path-provisioner stores PVC data on the host at `/opt/local-path-provisioner/`. The apiserver pod has no hostPath mount, so file/log endpoints that fall back to disk reads fail with "no such file or directory."
2. **Stale sidecar image**: The sidecar image in the cluster predates the log tee and trace collection code. Runs execute but produce no observable output.
3. **Stale hydration image**: The hydration image doesn't generate `.devcontainer` or `.aot` directories, so the sidecar has nothing to read for file browsing.
4. **Partial rebuild**: The controlplane image was rebuilt after code changes, but the sidecar and hydration images were not redeployed, leaving them out of sync.
5. **No automated full-rebuild task**: There is no single command to rebuild all images, import them into k0s, and restart all deployments — making it easy to forget a component.

## What Changes

- **Mount host path in apiserver pod**: Add a hostPath volume mount for `/opt/local-path-provisioner/` to the API server deployment so it can read PVC data directly.
- **Rebuild all Docker images**: Rebuild controlplane, sidecar, hydration, and agent-base images from current source to pick up all code changes.
- **Import and redeploy**: Import all rebuilt images into k0s and rollout restart all deployments.
- **Add `deploy:all` Taskfile target**: A single task that builds all images, imports them into k0s, and restarts all deployments.
- **End-to-end verification**: Create a test run and verify each observability feature works: log streaming, file browsing, shell access, and trace viewing.
- **Fix code bugs**: Fix any bugs discovered during end-to-end verification.

## Capabilities

### New Capabilities
- `working-observability`: All observability features (logs, files, shell, traces) verified working end-to-end in the deployed cluster.

### Modified Capabilities

## Impact

- `deploy/helm/aot/templates/apiserver-deployment.yaml` — add hostPath volume and volumeMount for `/opt/local-path-provisioner/`
- `Taskfile.yml` (or `Taskfile.yaml`) — add `deploy:all` task that builds all images, imports into k0s, and restarts deployments
- `Dockerfile.sidecar` (or equivalent) — rebuild with current source including log tee and trace collection
- `Dockerfile.hydration` (or equivalent) — rebuild with current source including `.devcontainer` and `.aot` directory generation
- `Dockerfile.controlplane` (or equivalent) — rebuild with current source
- `Dockerfile.agent-base` (or equivalent) — rebuild with current source
