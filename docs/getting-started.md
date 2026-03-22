# Getting Started

This guide walks through setting up a local UNCWORKS instance using the `aot-local` development environment.

## Prerequisites

- **k0s** -- Single-binary Kubernetes distribution (the local dev cluster)
- **Docker** -- For building container images
- **Task** -- Task runner (`go-task/task`, not GNU Make)
- **Helm** -- Kubernetes package manager
- **kubectl** -- Kubernetes CLI

## Clone the Repository

```
git clone <repo-url>
cd uncworks
```

The `aot-local/` directory (located at `../aot-local` relative to uncworks) contains the local cluster configuration.

## Start the Cluster

From the `aot-local/` directory:

```
task up
```

This single command:
1. Builds all Docker images (init, sidecar, agent, controlplane, web)
2. Imports them into the k0s container runtime
3. Applies the AgentRun CRD
4. Deploys Temporal, Ollama, and the UNCWORKS Helm chart
5. Waits for all deployments to become ready

## Pull a Model

After the cluster is running, pull at least one Ollama model:

```
kubectl -n aot exec deploy/ollama -- ollama pull qwen2.5:0.5b
```

For better results, pull the default model:

```
kubectl -n aot exec deploy/ollama -- ollama pull qwen3:8b
```

## Access the UI

The web dashboard is exposed via NodePort:

```
http://<host-ip>:30300
```

The Temporal UI is available at:

```
http://<host-ip>:30823
```

## Verify the Installation

```
task status
```

This shows all pods, services, and access URLs.

## Next Steps

- [Creating Runs](guides/creating-runs.md) -- Submit your first agent run
- [Model Configuration](guides/models.md) -- Add cloud models via OpenRouter
