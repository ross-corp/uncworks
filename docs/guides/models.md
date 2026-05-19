# Models

All LLM calls route through LiteLLM (`:4000`). The agent picks a model by name; LiteLLM handles auth, routing, retries, and fallbacks. Config: `deploy/litellm/litellm-config.yaml`.

```
agent → LiteLLM → Ollama (local) | OpenRouter (cloud)
```

## Built-in models

### Local (Ollama)

| Name | Backend |
|------|---------|
| `qwen3:8b` | Default local. |
| `llama3.1:8b` | Alternative. |
| `qwen2.5:0.5b` | CI only. |
| `default` | Alias → `qwen3:8b`. |
| `ci` | Alias → `qwen2.5:0.5b`. |

### Cloud (OpenRouter)

| Name | Provider | $/M in / out | Context |
|------|----------|-------------:|--------:|
| `deepseek-v3.1` | DeepSeek | 0.15 / 0.75 | 32K |
| `deepseek-v3.2` | DeepSeek | 0.26 / 0.38 | 164K |
| `qwen3-coder` | Qwen | 0.22 / 1.00 | 262K |
| `mistral-medium` | Mistral | 0.40 / 2.00 | 131K |
| `default-cloud` | DeepSeek | alias → `deepseek-v3.2` | — |
| `premium` | Qwen | alias → `qwen3-coder` | — |
| `llm-judge` | DeepSeek | alias → `deepseek-v3.1` (cheap judge) | — |

### Free tier

| Name | Provider |
|------|----------|
| `qwen3-coder-free` | Qwen via OpenRouter |
| `mistral-small-free` | Mistral via OpenRouter |
| `gpt-oss-120b-free` | OpenRouter |

Free models are rate-limited. The CLI marks them with a `(free)` indicator in `uncworks runs credits`.

## Fallbacks

LiteLLM falls through on provider failure:

- `qwen3-coder` → `deepseek-v3.2` → `deepseek-v3.1` → `qwen3:8b`
- `deepseek-v3.2` → `qwen3-coder` → `deepseek-v3.1` → `qwen3:8b`
- `default-cloud` → `deepseek-v3.2` → `qwen3:8b`

## Adding a model

Edit `deploy/litellm/litellm-config.yaml`:

```yaml
- model_name: "my-model"
  litellm_params:
    model: "openrouter/provider/model-name"
```

Ollama needs the `ollama_chat/` prefix and an `api_base`:

```yaml
- model_name: "my-local-model"
  litellm_params:
    model: "ollama_chat/model:tag"
    api_base: "http://ollama:11434"
```

## OpenRouter

```
OPENROUTER_API_KEY=sk-or-...
```

Set during `uncworks setup` or pass via Helm values. The key is held by LiteLLM, never the agent — agent pods get a scoped LiteLLM virtual key with budget caps and per-key model allowlists.

## Per-run / per-stage selection

`modelTier` on the `AgentRun` spec selects the model. For spec-driven, `pipelineConfig.{plan,execute,verify}.model` overrides per stage. The LLM judge always uses `deepseek-v3.1` independent of the agent — judge cost is decoupled from agent cost.
