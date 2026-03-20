# Helm Values Reference

Chart: `deploy/helm/aot/`

All configurable values for the UNCWORKS Helm chart.

## Temporal

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `temporal.host` | `string` | `""` | Temporal Frontend gRPC address (e.g., `temporal:7233`) |
| `temporal.namespace` | `string` | `default` | Temporal namespace |

## LLM Endpoint

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `llm.baseUrl` | `string` | `""` | LiteLLM/Ollama/OpenAI base URL |
| `llm.apiKey` | `string` | `""` | API key for the LLM endpoint |

## Container Images

All images follow the same structure: `repository`, `tag`, `pullPolicy`.

| Value | Default Repository | Description |
|-------|--------------------|-------------|
| `images.controlplane` | `ghcr.io/uncworks/aot-controlplane` | API server, controller, temporal worker |
| `images.init` | `ghcr.io/uncworks/aot-init` | Workspace hydration init container |
| `images.sidecar` | `ghcr.io/uncworks/aot-sidecar` | RPC gateway sidecar |
| `images.agent` | `ghcr.io/uncworks/aot-agent` | Agent base image |
| `images.web` | `ghcr.io/uncworks/aot-web` | Web dashboard |

All tags default to empty (uses `Chart.appVersion`). All pull policies default to `IfNotPresent`.

## Controller

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `controller.replicas` | `int` | `1` | Controller replica count |
| `controller.metricsPort` | `int` | `8095` | Prometheus metrics port |
| `controller.resources` | `object` | `{}` | Resource requests/limits |

## Temporal Worker

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `worker.replicas` | `int` | `1` | Worker replica count |
| `worker.resources` | `object` | `{}` | Resource requests/limits |

## API Server

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `apiserver.replicas` | `int` | `1` | API server replica count |
| `apiserver.port` | `int` | `50055` | ConnectRPC listen port |
| `apiserver.resources` | `object` | `{}` | Resource requests/limits |
| `apiserver.service.type` | `string` | `ClusterIP` | Kubernetes service type |
| `apiserver.apiKey` | `string` | `""` | API key for request authentication |
| `apiserver.allowedOrigins` | `string` | `""` | CORS origins (comma-separated); defaults to localhost dev ports |

## Pipeline (Spec-Driven)

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `pipeline.maxRetries` | `int` | `3` | Max execute/verify retry attempts |
| `pipeline.planTimeout` | `int` | `120` | Planning stage timeout (seconds) |
| `pipeline.verifyModel` | `string` | `""` | Model for verification LLM judge (defaults to execution model) |

## Web Dashboard

| Value | Type | Default | Description |
|-------|------|---------|-------------|
| `web.enabled` | `bool` | `true` | Enable the web dashboard |
| `web.replicas` | `int` | `1` | Dashboard replica count |
| `web.port` | `int` | `3000` | Dashboard listen port |
| `web.resources` | `object` | `{}` | Resource requests/limits |
| `web.service.type` | `string` | `ClusterIP` | Kubernetes service type |
| `web.service.nodePort` | `string` | `""` | NodePort (only when type is NodePort) |
