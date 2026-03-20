# Model Configuration

UNCWORKS routes all LLM requests through a LiteLLM proxy, providing a unified interface for local and cloud models. Configuration lives in `deploy/litellm/litellm-config.yaml`.

## Architecture

```
Agent Pod --> LiteLLM Proxy --> Ollama (local)
                            --> OpenRouter (cloud)
```

The agent selects a model by name (e.g., `qwen3:8b`, `deepseek-v3.1`). LiteLLM handles routing, retries, and fallback chains.

## Available Models

### Local Models (Ollama)

| Model Name | Backend | Notes |
|------------|---------|-------|
| `qwen3:8b` | Ollama | Default local model, strong coding |
| `llama3.1:8b` | Ollama | Alternative local model |
| `qwen2.5:0.5b` | Ollama | Tiny model for CI/testing only |
| `default` | Ollama | Alias for `qwen3:8b` |
| `ci` | Ollama | Alias for `qwen2.5:0.5b` |

### Cloud Models (OpenRouter)

| Model Name | Provider | Cost (in/out per M tokens) | Context |
|------------|----------|---------------------------|---------|
| `deepseek-v3.1` | DeepSeek | $0.15 / $0.75 | 32K |
| `deepseek-v3.2` | DeepSeek | $0.26 / $0.38 | 164K |
| `qwen3-coder` | Qwen | $0.22 / $1.00 | 262K |
| `mistral-medium` | Mistral | $0.40 / $2.00 | 131K |
| `default-cloud` | DeepSeek | Alias for `deepseek-v3.1` | -- |
| `premium` | Qwen | Alias for `qwen3-coder` | -- |

### Free Tier (rate-limited)

| Model Name | Provider |
|------------|----------|
| `qwen3-coder-free` | Qwen (via OpenRouter) |
| `mistral-small-free` | Mistral (via OpenRouter) |

## Fallback Chains

LiteLLM is configured with fallback chains so that if a cloud model is unavailable, requests fall through to alternatives:

- `qwen3-coder` --> `deepseek-v3.2` --> `deepseek-v3.1` --> `qwen3:8b`
- `deepseek-v3.2` --> `qwen3-coder` --> `deepseek-v3.1` --> `qwen3:8b`
- `default-cloud` --> `deepseek-v3.2` --> `qwen3:8b`

## Adding a Model

Add an entry to the `model_list` in `deploy/litellm/litellm-config.yaml`:

```yaml
- model_name: "my-model"
  litellm_params:
    model: "openrouter/provider/model-name"
```

For Ollama models, use the `ollama_chat/` prefix and specify `api_base`:

```yaml
- model_name: "my-local-model"
  litellm_params:
    model: "ollama_chat/model:tag"
    api_base: "http://ollama:11434"
```

## OpenRouter Setup

Cloud models route through [OpenRouter](https://openrouter.ai). Set your API key as an environment variable accessible to LiteLLM:

```
OPENROUTER_API_KEY=sk-or-...
```

## Model Selection in Runs

When creating a run, the `modelTier` field (or model selector in the UI) determines which model the agent uses. This maps directly to a `model_name` in the LiteLLM config. For spec-driven runs, each pipeline stage can use a different model via `pipelineConfig`.
