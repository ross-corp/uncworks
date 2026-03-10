## Context

Agent pods currently require direct API keys and endpoint configuration for whatever LLM provider they use. There is no centralized model routing, spend tracking, fallback logic, or ability to swap providers without changing agent configuration. Every agent independently manages its own LLM connectivity.

LiteLLM is an OpenAI-compatible proxy that provides a unified `/v1/chat/completions` endpoint with multi-backend routing, automatic fallbacks, per-agent virtual keys with budget caps, and spend tracking. Since every major LLM SDK respects `OPENAI_BASE_URL`, agents require zero code changes to route through LiteLLM.

The existing `buildAgentPod` in `internal/controller/agentrun_controller.go` already constructs environment variables for agent containers. This change extends that env var list with `OPENAI_BASE_URL` and `OPENAI_API_KEY` to transparently route all LLM calls through the gateway.

## Goals / Non-Goals

### Goals
- Centralize LLM routing through a single proxy (LiteLLM) for all agent pods
- Enable per-agent budget caps and spend tracking via virtual keys
- Provide automatic fallback from local models (Ollama) to cloud free tier (OpenRouter)
- Support model tiers (default, default-cloud, premium) selectable per agent spec
- Require zero code changes in agents -- routing is fully transparent via env vars
- Provide reference deployment configs for LiteLLM and Ollama on k0s

### Non-Goals
- Bundling LiteLLM or Ollama into AOT's Helm chart -- they are explicit infrastructure dependencies
- Building a custom LLM proxy -- LiteLLM is used as-is
- Streaming/SSE passthrough optimization -- standard HTTP proxying is sufficient for now
- Multi-tenancy isolation beyond per-agent virtual keys
- Fine-tuning or model training workflows

## Decisions

### LiteLLM as a transparent proxy
Agents do not know LiteLLM exists. They use standard OpenAI SDK with `OPENAI_BASE_URL` pointing to LiteLLM. This requires zero code changes in agents. Any SDK or tool that respects the OpenAI environment variables works automatically.

### Connection configuration
The controller reads `LITELLM_BASE_URL` (default: `http://litellm:4000`) and injects `OPENAI_BASE_URL=$LITELLM_BASE_URL/v1` into agent pod environment variables. This keeps the LiteLLM endpoint configurable without hardcoding it into agent images or specs.

### Virtual key lifecycle
A Temporal activity (`ProvisionLLMKey`) provisions a LiteLLM virtual key via the LiteLLM Admin API (`POST /key/generate`) at workflow start. The key has a per-agent budget cap (max_budget in USD) and model restrictions based on the agent's model tier. The key is injected as `OPENAI_API_KEY` into the pod env. On workflow completion (success, failure, or cancellation), a `RevokeLLMKey` activity calls `POST /key/delete` to clean up.

### Model tiers
Three tiers are defined:
- `default` -- Ollama local models (e.g., llama3.1:8b). No cost, no external calls.
- `default-cloud` -- OpenRouter free tier models. Free but rate-limited.
- `premium` -- Anthropic/OpenAI via API keys stored in LiteLLM config.

The `AgentRunSpec` gains a `modelTier` field (default: `"default"`) that controls which models the virtual key is authorized to access.

### Fallback chain
LiteLLM is configured with fallback routing: `default` falls back to `default-cloud`. If Ollama is down or overloaded, requests automatically route to OpenRouter free tier. This ensures agent runs do not fail due to local model unavailability.

### LiteLLM configuration
A reference `litellm-config.yaml` is provided in `deploy/litellm/`. It covers `model_list` (Ollama, OpenRouter, premium providers), `litellm_settings` (fallbacks, rate limiting), and `general_settings` (database connection, master key reference).

### Ollama deployment
Ollama is deployed to k0s via Helm chart. Model selection depends on environment:
- CI/testing: `qwen2.5:0.5b` (small, fast inference for integration tests)
- Development: `llama3.1:8b` (capable enough for real agent work)

Model pull is a documented post-deployment step, not automated in the chart.

### PostgreSQL sharing
LiteLLM uses PostgreSQL for virtual key storage and spend tracking. It shares the existing PostgreSQL instance but uses a separate database (`litellm`). This avoids deploying a second PostgreSQL while maintaining data isolation.

### Not bundled
LiteLLM and Ollama are explicit infrastructure dependencies, not bundled in AOT's Helm chart. Deployment guides are provided in `deploy/litellm/` and `deploy/ollama/`. This keeps AOT's chart focused and allows operators to bring their own LiteLLM or model backend.

## Risks / Trade-offs

### Single point of failure
LiteLLM becomes a single point of failure for all LLM calls. Mitigation: LiteLLM supports multiple replicas and has a health check endpoint for Kubernetes probes. The proxy is stateless (state is in PostgreSQL), so horizontal scaling is straightforward.

### Latency overhead
Adding a proxy hop adds latency to every LLM call. For typical LLM response times (seconds), the sub-millisecond proxy overhead is negligible. This trade-off is acceptable for the routing and observability benefits.

### Ollama resource requirements
Running Ollama with even small models (llama3.1:8b requires ~8GB RAM) may be heavy for minimal dev environments. Mitigation: the fallback chain means Ollama is optional -- if not deployed, requests fall through to OpenRouter free tier.

### Virtual key cleanup on crashes
If the Temporal workflow crashes without running the `RevokeLLMKey` activity, orphaned virtual keys may persist. Mitigation: virtual keys have `max_budget` caps limiting blast radius. A periodic cleanup job can revoke keys for completed/failed AgentRuns as a safety net.

### LiteLLM master key security
The LiteLLM master key grants full admin API access (key provisioning, deletion, config changes). It must be stored as a Kubernetes Secret and only accessible to the controller/workflow, not to agent pods.
