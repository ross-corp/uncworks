# Local development

The local cluster is k3s inside a [Colima](https://github.com/abiosoft/colima)
VM. Task drives it.

## Prereqs

- [devbox](https://www.jetify.com/devbox) holds every tool.
- Colima provides the container runtime and k3s on macOS.
- [Wails](https://wails.io) is needed only to build the macOS desktop app.

## First time

```bash
devbox shell                  # enter the toolchain
task install                  # dependencies and git hooks
task cluster:setup            # Colima VM, Temporal, LiteLLM, soft-serve, Helm install
task cluster:status           # confirm every pod is healthy
```

## Layout

| Workload | What |
|----------|------|
| `apiserver` | Serves ConnectRPC and HTTP on `:50055` |
| `controller` | Reconciles `AgentRun` and `Project` |
| `worker` | Runs the Temporal workflow and its activities |
| `web` | React dashboard behind nginx |
| Temporal | Workflow engine |
| LiteLLM | Proxies to Ollama and OpenRouter |
| Soft-Serve | Git server, one repo per project |

## Day-to-day

```bash
task dev:deploy       # build images into k3s containerd and roll out
task dev:images       # build only, no rollout
task dev:web          # Vite dev server with hot reload
task build            # build every Go binary into ./bin/
task proto:gen        # regenerate Go and TypeScript from the proto files
task test:go          # Go tests
task lint             # golangci-lint and tsc --noEmit
task cluster:logs     # tail every UNCWORKS pod
task cluster:teardown # tear down the Colima VM
```

Run `task --list` for the rest.

## Why k3s containerd directly

The build writes images straight into the k3s containerd namespace `k8s.io`, so
there is no `docker save | docker load` round trip. This is what makes the image
cycle fast.

```bash
task dev:deploy
# or manually:
kubectl rollout restart deploy/aot-apiserver deploy/aot-controller -n aot
kubectl rollout status  deploy/aot-apiserver deploy/aot-controller -n aot
```

## Desktop app

`cmd/uncworks-app/`. Build output is gitignored.

```bash
task app:build       # build and install to /Applications/UNCWORKS.app
```

The app embeds `web/dist/` through `//go:embed`.

## Proto

```bash
task proto:gen       # buf generate into gen/go/ and gen/ts/
task proto:lint      # buf lint
task proto:breaking  # compare against main
```

Commit the generated files in the same commit as the proto change.

## Control-plane environment variables

| Variable | Binary | Purpose |
|-----|--------|---|
| `LISTEN_ADDR` | apiserver | gRPC listen address. Defaults to `:50055` |
| `TEMPORAL_HOST` | all | Frontend address |
| `LITELLM_BASE_URL` | apiserver, worker | Proxy URL |
| `LITELLM_MASTER_KEY` | apiserver, worker | Authenticates to the proxy |
| `GITHUB_TOKEN` | controller | Creates pull requests |
| `SOFT_SERVE_ADDR` | controller, worker | SSH address |
| `AOT_API_KEY` | apiserver | When set, every client call MUST carry it as a header |
| `LOG_FORMAT` | all | Either `text`, the default, or `json` |
| `LOG_LEVEL` | all | One of `debug`, `info`, `warn`, or `error` |

The defaults live in `deploy/helm/aot/values.yaml`.
