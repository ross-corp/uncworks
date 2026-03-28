## REMOVED Requirements

### Requirement: Cluster setup command
**Reason**: The k0s + systemd cluster management model is superseded by the bring-your-own-cluster model. Users now provide their own local Kubernetes cluster (Docker Desktop, OrbStack, Rancher Desktop, Colima, k3d, kind) and `uncworks setup` installs into it. The `task cluster:setup` target and associated systemd unit files are removed.
**Migration**: Run `uncworks setup` instead of `task cluster:setup`. Ensure a local Kubernetes cluster is running first (Docker Desktop with Kubernetes enabled, OrbStack, etc.).

### Requirement: Cluster status command
**Reason**: Replaced by `uncworks status` which queries the Kubernetes API directly for pod health.
**Migration**: Use `uncworks status` instead of `task cluster:status`.

### Requirement: Cluster teardown command
**Reason**: Replaced by `uncworks teardown`.
**Migration**: Use `uncworks teardown` instead of `task cluster:teardown`. Note: `uncworks teardown` does not delete the cluster itself, only the Helm release.

### Requirement: Cluster logs command
**Reason**: Replaced by `uncworks tui` log streaming and direct `kubectl logs` access.
**Migration**: Use `uncworks tui` or `kubectl logs -n uncworks -l app.kubernetes.io/name=aot` instead.

### Requirement: Ollama model pre-pull
**Reason**: Ollama is disabled by default in the local values preset. Users who opt in to Ollama manage model pulls themselves via the Ollama API.
**Migration**: If Ollama is enabled, pull models via `kubectl exec -n uncworks <ollama-pod> -- ollama pull qwen2.5:0.5b`.

### Requirement: Image build and import
**Reason**: End-user installs pull images from `ghcr.io/uncworks/...`. Local development image import is handled by developer-specific tooling (`task k0s:images`), not the cluster management lifecycle.
**Migration**: For development builds, use `task k0s:images` (until k0s is fully removed) or equivalent image import for your cluster tool.
