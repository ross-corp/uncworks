# Local development

Local cluster: k3s in a [Colima](https://github.com/abiosoft/colima) VM, driven by Task.

## Prereqs

- [devbox](https://www.jetify.com/devbox) — all tooling lives here.
- Colima — macOS container runtime with k3s.
- [Wails](https://wails.io) — only if you build the macOS desktop app.

## First time

```bash
devbox shell                  # tooling
task install                  # deps + git hooks
task cluster:setup            # Colima VM 'uncworks' + Temporal/LiteLLM/soft-serve + Helm install
task cluster:status           # sanity
```

## Layout

| Workload | What |
|----------|------|
| `apiserver` | ConnectRPC + HTTP on `:50055` |
| `controller` | `AgentRun` + `Project` reconciler |
| `worker` | Temporal workflow + activities |
| `web` | React dashboard, nginx-fronted |
| Temporal | Workflow engine |
| LiteLLM | Proxy → Ollama / OpenRouter |
| Soft-Serve | Per-project git server |

## Day-to-day

```bash
task dev:deploy       # build images into k3s containerd (k8s.io) + rollout
task dev:images       # build only, no rollout
task dev:web          # Vite dev server (hot-reload web)
task build            # all Go binaries → ./bin/
task proto:gen        # regenerate Go + TS from proto
task test:go          # Go tests
task lint             # golangci-lint + tsc --noEmit
task cluster:logs     # tail all UNCWORKS pods
task cluster:teardown # tear down Colima VM
```

`task --list` for the rest.

## Why k3s containerd directly

Images are built into the k3s containerd namespace (`k8s.io`) — no `docker save | docker load` round-trip. This is the main reason image dev cycles are fast.

```bash
task dev:deploy
# or manually:
kubectl rollout restart deploy/aot-apiserver deploy/aot-controller -n aot
kubectl rollout status  deploy/aot-apiserver deploy/aot-controller -n aot
```

## Desktop app

`cmd/uncworks-app/`. Build output is gitignored.

```bash
task app:build       # builds + installs to /Applications/UNCWORKS.app
```

Embeds `web/dist/` via `//go:embed`.

## Proto

```bash
task proto:gen       # buf generate → gen/go/, gen/ts/
task proto:lint      # buf lint
task proto:breaking  # diff vs main
```

Commit generated files alongside the proto change.

## Env (control plane)

| Var | Binary | |
|-----|--------|---|
| `LISTEN_ADDR` | apiserver | gRPC listen (default `:50055`) |
| `TEMPORAL_HOST` | all | Frontend address |
| `LITELLM_BASE_URL` | apiserver, worker | Proxy URL |
| `LITELLM_MASTER_KEY` | apiserver, worker | Auth |
| `GITHUB_TOKEN` | controller | PR creation |
| `SOFT_SERVE_ADDR` | controller, worker | SSH address |
| `AOT_API_KEY` | apiserver | Required client header when set |
| `LOG_FORMAT` | all | `text` (default) / `json` |
| `LOG_LEVEL` | all | `debug` / `info` / `warn` / `error` |

Defaults live in `deploy/helm/aot/values.yaml`.
