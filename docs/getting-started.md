# Getting Started

This guide walks through setting up a local UNCWORKS instance using the `aot-local` development environment with k0s.

## Prerequisites

| Tool | Purpose |
|------|---------|
| **k0s** | Single-binary Kubernetes distribution (the local dev cluster) |
| **Docker** | Building container images |
| **Task** | Task runner (`go-task/task`) -- used instead of Make |
| **Helm** | Kubernetes package manager |
| **kubectl** | Kubernetes CLI |
| **Go 1.25+** | Building control plane binaries |
| **Node.js 22+** | Building the web dashboard |

## Clone the Repository

```
git clone <repo-url>
cd uncworks
```

The `aot-local/` directory (located at `../aot-local` relative to uncworks) contains the local cluster configuration, Taskfile, and Helm value overrides.

## Start the Cluster

From the `aot-local/` directory:

```
task up
```

This single command:
1. Builds all Docker images (hydration, sidecar, agent-base, controlplane, web/BFF)
2. Imports them into the k0s container runtime
3. Applies the AgentRun and Project CRDs
4. Deploys Temporal, Ollama, LiteLLM, Soft-Serve, and the UNCWORKS Helm chart
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

This shows all pods, services, and access URLs. Expected pods in the `aot` namespace:

```
aot-apiserver-*        API Server (ConnectRPC + REST)
aot-controller-*       K8s Controller (AgentRun + Project reconcilers)
aot-worker-*           Temporal Worker
aot-web-*              Web Dashboard (React + nginx BFF)
temporal-*             Temporal Server
ollama-*               Local LLM inference
litellm-*              LLM routing proxy
soft-serve-*           In-cluster Git server
```

## Create Your First Run

1. Open the web dashboard at `http://<host-ip>:30300`
2. Click "New Run" in the top navigation
3. Fill in the form:
   - **Prompt**: Describe the task (e.g., "Add a health check endpoint to the API")
   - **Repository URL**: GitHub repo URL (e.g., `https://github.com/owner/repo.git`)
   - **Branch**: Branch to check out (defaults to the repo's default branch)
   - **Model**: Select a model tier (`default` for local Ollama, `default-cloud` for OpenRouter)
   - **Mode**: Choose `single` for a simple task or `spec-driven` for the full plan/execute/verify pipeline
4. Click "Create Run"
5. The run detail page shows real-time progress:
   - Activity feed with agent output
   - File browser showing workspace changes
   - Trace timeline showing tool calls and stage transitions
   - Verification panel (spec-driven mode)

## Create a Project

Projects group related runs and provide default configuration.

1. Navigate to the Projects page in the dashboard
2. Click "New Project"
3. Configure:
   - **Name**: Kebab-case identifier (e.g., `my-api`)
   - **Display Name**: Human-readable name
   - **Repositories**: Add one or more GitHub repos
   - **Devbox Packages**: Nix packages to install in every workspace (e.g., `go@1.22`, `nodejs@20`)
   - **Defaults**: Default model tier, TTL, auto-push/PR settings
4. The controller automatically creates a soft-serve config repo for the project
5. Future runs can reference the project via `projectRef` to inherit defaults

## Configure Cloud Models (Optional)

To use cloud LLMs via OpenRouter, add your API key to the LiteLLM configuration:

```
kubectl -n aot edit configmap litellm-config
```

Add an OpenRouter model entry and restart the LiteLLM pod.

## Configure GitHub Integration (Optional)

For auto-push, PR creation, and CI autofix:

1. Create a GitHub personal access token with `repo` scope
2. Store it as a Kubernetes secret:
   ```
   kubectl -n aot create secret generic github-token --from-literal=token=ghp_...
   ```
3. Set `GITHUB_TOKEN_SECRET_NAME=github-token` on the API server
4. For webhooks, set `GITHUB_WEBHOOK_SECRET` and configure the webhook URL in your GitHub repo settings

## Next Steps

- [Creating Runs](guides/creating-runs.md) -- Detailed guide on run creation options
- [Model Configuration](guides/models.md) -- Add and configure LLM models
- [Spec-Driven Runs](guides/spec-driven.md) -- Using the plan/execute/verify pipeline
- [API Reference](reference/api.md) -- ConnectRPC and REST endpoint documentation
- [CRD Reference](reference/crd.md) -- AgentRun and Project CRD field reference
