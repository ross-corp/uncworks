# Local Development

UNCWORKS uses a k0s-based local cluster managed through the `aot-local/` directory (located at `../aot-local` relative to the uncworks repo).

## Architecture

The local dev environment runs on a single k0s node with:
- **Temporal** -- Workflow engine (SQLite-backed dev server)
- **Ollama** -- Local LLM inference
- **UNCWORKS Control Plane** -- API server, controller, temporal worker (single image)
- **UNCWORKS Web** -- Dashboard (Vite + React, served via nginx)
- **Agent Pods** -- Ephemeral pods with init, sidecar, and agent containers

## Taskfile Commands (aot-local/)

Run these from the `aot-local/` directory:

| Command | Description |
|---------|-------------|
| `task up` | Build, import, and deploy everything |
| `task build` | Build all Docker images |
| `task build:agents` | Build init, sidecar, agent images |
| `task build:controlplane` | Build control plane image |
| `task build:web` | Build web dashboard image |
| `task import` | Import images into k0s runtime |
| `task deploy` | Apply CRDs, manifests, and Helm chart |
| `task status` | Show pod status and access URLs |
| `task logs` | Tail logs from all UNCWORKS pods |
| `task down` | Remove all UNCWORKS resources |
| `task pull-model` | Pull `qwen2.5:0.5b` into Ollama |

## Taskfile Commands (uncworks/)

Run these from the uncworks repo root:

| Command | Description |
|---------|-------------|
| `task build` | Build all Go binaries to `./bin/` |
| `task docker:build` | Build all Docker images |
| `task k0s:images` | Build and import images into k0s |
| `task deploy:all` | Build, import, restart all deployments |
| `task dev:web` | Start Vite dev server for web dashboard |
| `task install` | Install Go and npm dependencies |
| `task lint` | Run golangci-lint and TypeScript checks |
| `task proto:gen` | Generate Go + TS code from proto files |
| `task proto:lint` | Lint proto files with buf |
| `task temporal:dev` | Start local Temporal dev server |

## Go Binaries

The control plane builds six binaries:

| Binary | Description |
|--------|-------------|
| `apiserver` | ConnectRPC API server |
| `controller` | Kubernetes controller for AgentRun CRD |
| `temporal-worker` | Temporal workflow/activity worker |
| `hydration` | Init container for workspace setup |
| `sidecar` | RPC gateway running in agent pods |
| `aot` | CLI tool |

## Persistent Dev Cluster

For a systemd-managed persistent cluster (survives reboots):

```
task cluster:setup    # Install units, build, deploy, start
task cluster:status   # Check service health
task cluster:teardown # Stop and remove
task cluster:logs     # View combined logs
```

## Image Development Cycle

1. Edit source code
2. `task docker:build` (or specific target like `task build:agents`)
3. `task k0s:images` to import into k0s
4. `kubectl rollout restart deploy/<name> -n aot` to pick up new images
