# Local Development

UNCWORKS runs on a local k3s cluster managed by [Colima](https://github.com/abiosoft/colima).

## Prerequisites

- [devbox](https://www.jetify.com/devbox) — installs all tooling in an isolated shell
- [Colima](https://github.com/abiosoft/colima) — macOS container runtime with k3s support
- [Wails](https://wails.io) (optional) — required for the macOS desktop app only

## First-time Setup

```bash
# Install devbox, then enter the dev shell
devbox shell

# Install dependencies + git hooks
task install

# Create the Colima k3s VM and deploy everything
task cluster:setup

# Verify the cluster is healthy
task cluster:status
```

`task cluster:setup` creates a Colima VM named `uncworks` with k3s, deploys all infrastructure dependencies (Temporal, LiteLLM, soft-serve), and deploys the UNCWORKS Helm chart.

## Architecture

The local dev environment runs a k3s cluster inside a Colima VM:

| Component | Description |
|-----------|-------------|
| `apiserver` | ConnectRPC + HTTP API server |
| `controller` | Kubernetes controller for AgentRun CRD |
| `worker` | Temporal workflow + activity worker |
| `web` | React dashboard (nginx) |
| Temporal | Workflow engine |
| LiteLLM | LLM proxy (routes to Ollama or cloud) |
| soft-serve | Per-project git server (config + specs) |

## Common Tasks

```bash
task dev:deploy       # Build images into k8s.io namespace + rollout all deployments
task dev:web          # Start Vite dev server (hot-reload web dashboard)
task build            # Build all Go binaries to ./bin/
task proto:gen        # Regenerate Go + TypeScript from .proto files
task test:go          # Run Go tests
task lint             # Run golangci-lint + TypeScript checks
task cluster:logs     # Tail logs from all UNCWORKS pods
task cluster:teardown # Tear down the Colima cluster
```

Run `task --list` for the full list.

## Image Development Cycle

Images are built directly into the k3s containerd namespace (`k8s.io`) — no `docker save | load` step needed:

```bash
# Build and rollout everything
task dev:deploy

# Or just rebuild images (without rollout)
task dev:images

# Manual rollout after image rebuild
kubectl rollout restart deploy/aot-apiserver deploy/aot-controller -n aot
kubectl rollout status  deploy/aot-apiserver deploy/aot-controller -n aot
```

## Desktop App

The macOS desktop app lives in `cmd/uncworks-app/` (gitignored — build output is not committed).

```bash
task app:build        # Build and install to /Applications/UNCWORKS.app
```

The app embeds the compiled web frontend from `web/dist/` via `//go:embed`.

## Proto Changes

After editing any `.proto` file:

```bash
task proto:gen    # runs buf generate — updates gen/go/ and gen/ts/
task proto:lint   # check for breaking changes and style issues
```

Commit the generated files alongside the proto changes.

## Environment Variables

Key env vars consumed by the control plane binaries (see `deploy/helm/aot/values.yaml` for defaults):

| Variable | Binary | Description |
|----------|--------|-------------|
| `LISTEN_ADDR` | apiserver | gRPC listen address (default `:50055`) |
| `TEMPORAL_HOST` | all | Temporal frontend address |
| `LITELLM_BASE_URL` | apiserver, worker | LiteLLM proxy base URL |
| `LITELLM_MASTER_KEY` | apiserver, worker | LiteLLM authentication key |
| `GITHUB_TOKEN` | controller | GitHub API token for PR creation |
| `SOFT_SERVE_ADDR` | controller, worker | soft-serve SSH address |
| `AOT_API_KEY` | apiserver | API authentication key |
| `LOG_FORMAT` | all | `text` (default) or `json` |
| `LOG_LEVEL` | all | `debug`, `info` (default), `warn`, `error` |
