# Helm values

Chart: `deploy/helm/aot/`. Selected values; see `values.yaml` for the full surface.

## Temporal

| Value | Default | |
|-------|---------|---|
| `temporal.host` | `""` | Frontend gRPC (`temporal:7233`) |
| `temporal.namespace` | `default` | |

## LLM

| Value | Default | |
|-------|---------|---|
| `llm.baseUrl` | `""` | LiteLLM / Ollama / OpenAI base URL |
| `llm.apiKey` | `""` | |

## Images

Each: `repository`, `tag` (default empty → `Chart.appVersion`), `pullPolicy` (default `IfNotPresent`).

| Value | Default repository |
|-------|--------------------|
| `images.controlplane` | `ghcr.io/uncworks/aot-controlplane` |
| `images.init` | `ghcr.io/uncworks/aot-init` |
| `images.sidecar` | `ghcr.io/uncworks/aot-sidecar` |
| `images.agent` | `ghcr.io/uncworks/aot-agent` |
| `images.web` | `ghcr.io/uncworks/aot-web` |

## Controller / worker / API server

| Value | Default | |
|-------|---------|---|
| `controller.replicas` | `1` | |
| `controller.metricsPort` | `8095` | Prom metrics |
| `worker.replicas` | `1` | |
| `apiserver.replicas` | `1` | |
| `apiserver.port` | `50055` | |
| `apiserver.service.type` | `ClusterIP` | |
| `apiserver.apiKey` | `""` | Required header when set |
| `apiserver.allowedOrigins` | `""` | CORS; defaults to localhost dev ports |

All have a `.resources` object for requests/limits.

## Pipeline

| Value | Default | |
|-------|---------|---|
| `pipeline.maxRetries` | `3` | Execute/verify cap |
| `pipeline.planTimeout` | `120` | Seconds |
| `pipeline.verifyModel` | `""` | Override judge model (defaults to execution model) |

## Web

| Value | Default | |
|-------|---------|---|
| `web.enabled` | `true` | |
| `web.replicas` | `1` | |
| `web.port` | `3000` | |
| `web.service.type` | `ClusterIP` | |
| `web.service.nodePort` | `""` | Only when `type: NodePort` |
