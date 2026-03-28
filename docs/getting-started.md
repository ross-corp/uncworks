# Getting Started

This guide walks through setting up a local UNCWORKS instance using `uncworks setup`.

## Prerequisites

| Tool | Purpose |
|------|---------|
| **uncworks CLI** | UNCWORKS setup and management (`brew install uncworks/tap/uncworks`) |
| **kubectl** | Kubernetes CLI |
| **helm** | Kubernetes package manager |
| **A local Kubernetes cluster** | See options below |

### Local Kubernetes Cluster

UNCWORKS runs on Kubernetes. You need a local cluster — any of the following work:

**macOS:**
- [Docker Desktop](https://docs.docker.com/desktop/kubernetes/) — enable Kubernetes in Preferences > Kubernetes
- [OrbStack](https://orbstack.dev/) — `brew install orbstack` (fastest)
- [Rancher Desktop](https://rancherdesktop.io/) — `brew install --cask rancher`

**Linux:**
- [k3d](https://k3d.io/) — `curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash` (requires Docker)
- [kind](https://kind.sigs.k8s.io/) — `go install sigs.k8s.io/kind@latest` (requires Docker)

## Install the CLI

```bash
# macOS / Linux (Homebrew)
brew install uncworks/tap/uncworks

# Or download a binary directly from GitHub Releases:
# https://github.com/uncworks/uncworks/releases
```

## Setup

Run the interactive setup wizard:

```bash
uncworks setup
```

The wizard will:
1. Detect your local Kubernetes context and let you select one
2. Check that the cluster has sufficient resources (min 2 CPU / 2Gi memory, recommended 4/4Gi)
3. Prompt for required configuration (LLM API key, GitHub token, Temporal address)
4. Deploy UNCWORKS via Helm (`helm upgrade --install`)
5. Print the web UI URL

For non-interactive / scripted setup:

```bash
uncworks setup \
  --context docker-desktop \
  --llm-key sk-... \
  --github-token ghp_... \
  --temporal-host temporal:7233
```

### Using the Local Values Preset

For local clusters, pass the included values preset for lighter resource usage and NodePort exposure:

```bash
uncworks setup --values deploy/helm/values.local.yaml
```

This sets NodePort 30300 for the web UI, reduces resource requests, and disables Ollama by default.

## Access the UI

After setup:

```bash
uncworks open    # starts port-forward + opens browser
```

Or navigate directly to `http://localhost:30300` if your cluster exposes NodePorts on localhost (Docker Desktop, OrbStack, Rancher Desktop).

## Terminal UI

```bash
uncworks tui     # launch the Bubble Tea terminal UI
```

The TUI shows active runs, streams logs, and lets you submit new runs — all from the terminal.

## Connecting to a Remote Server

```bash
uncworks connect grpc.example.com:50055   # store remote address
uncworks tui                               # TUI connects to remote server
```

## Status and Teardown

```bash
uncworks status      # show pod health
uncworks teardown    # uninstall UNCWORKS (keeps PVCs by default)
uncworks teardown --purge   # also delete PVCs (destroys workspace data)
```

## Create Your First Run

1. Open the web dashboard via `uncworks open`
2. Click "New Run"
3. Fill in the form:
   - **Prompt**: Describe the task (e.g., "Add a health check endpoint to the API")
   - **Repository URL**: GitHub repo URL (e.g., `https://github.com/owner/repo.git`)
   - **Branch**: Branch to check out (defaults to main)
   - **Model**: Select a model tier (`default` for local Ollama, `default-cloud` for OpenRouter)
   - **Mode**: `single` for a simple task or `spec-driven` for the full plan/execute/verify pipeline
4. Click "Create Run"

## Configure Cloud Models (Optional)

Pass your API key during setup:

```bash
uncworks setup --llm-key sk-or-...   # OpenRouter or OpenAI key
```

## Next Steps

- [Creating Runs](guides/creating-runs.md) — Detailed guide on run creation options
- [Model Configuration](guides/models.md) — Add and configure LLM models
- [Spec-Driven Runs](guides/spec-driven.md) — Using the plan/execute/verify pipeline
- [API Reference](reference/api.md) — ConnectRPC and REST endpoint documentation
- [CRD Reference](reference/crd.md) — AgentRun and Project CRD field reference
- [macOS App](guides/macos-app.md) — Installing the native UNCWORKS.app
