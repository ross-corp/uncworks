## Why

Agent pods currently need direct API keys and endpoint configuration for whatever LLM provider they use. There is no centralized model routing, no spend tracking per agent, no fallback logic, and no way to swap providers without changing agent configuration. LiteLLM is an OpenAI-compatible proxy that sits between agents and LLM providers, providing a unified `/v1/chat/completions` endpoint with multi-backend routing, automatic fallbacks, per-agent virtual keys with budget caps, and spend tracking -- all without any code changes in the agent since every LLM SDK respects `OPENAI_BASE_URL`.

## What Changes

- **LiteLLM as first-class infrastructure component**: Documented as an explicit dependency alongside PostgreSQL and Temporal. All agent pods route LLM calls through LiteLLM.
- **Ollama backend for local/dev**: In-cluster Ollama deployment serves small models (e.g., `qwen2.5:0.5b` for testing, `llama3.1:8b` for dev) as the primary LiteLLM backend.
- **OpenRouter free tier as cloud fallback**: LiteLLM fallback routing sends requests to OpenRouter's free models when Ollama is unavailable or overloaded.
- **Per-agent virtual keys**: Each `AgentRun` workflow provisions a LiteLLM virtual key with budget caps and model restrictions, injected as `OPENAI_API_KEY` into the agent pod. Key is revoked on workflow completion.
- **Agent pod environment configuration**: Controller/workflow injects `OPENAI_BASE_URL=http://litellm.<namespace>:4000/v1` and the provisioned virtual key into every agent pod.
- **k0s deployment option**: Document deploying the official LiteLLM Helm chart (`deploy/charts/litellm-helm`) and Ollama to k0s.
- **LiteLLM proxy configuration**: Provide a reference `litellm-config.yaml` with Ollama primary, OpenRouter fallback, and premium model tiers.

## Capabilities

### New Capabilities
- `llm-gateway`: LiteLLM proxy deployment, configuration, and connection management. Covers model routing, fallback chains, and provider abstraction.
- `llm-virtual-keys`: Per-agent virtual key lifecycle -- provisioning on workflow start, budget enforcement, spend tracking, revocation on completion.
- `ollama-backend`: Ollama deployment configuration for in-cluster local model serving as the primary LiteLLM backend.

### Modified Capabilities
- `k8s-orchestrator`: Agent pod spec gains `OPENAI_BASE_URL` and `OPENAI_API_KEY` environment variables injected by the controller/workflow.
- `agent-harness`: No code changes -- agents already use OpenAI-compatible SDKs. The routing is transparent.

## Impact

- **`internal/controller/`**: Pod spec construction adds LiteLLM env vars to agent container.
- **Agent pod runtime**: All LLM API calls route through `http://litellm:4000/v1` instead of direct provider endpoints.
- **Infrastructure**: Requires LiteLLM proxy accessible in-cluster. For k0s dev, deployed via Helm. LiteLLM needs PostgreSQL for virtual key storage (can share instance, separate database). Ollama deployment optional but recommended for local dev.
- **Cost**: Centralizes API key management -- only LiteLLM needs provider API keys, not individual agents.
- **Testing**: E2E tests can use Ollama with a tiny model via LiteLLM, making full agent lifecycle tests possible without paid API calls.
