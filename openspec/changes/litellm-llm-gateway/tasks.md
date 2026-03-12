## 1. LiteLLM Configuration

- [x] 1.1 Create `deploy/litellm/` directory
- [x] 1.2 Create `deploy/litellm/litellm-config.yaml`: model_list with Ollama primary (ollama_chat/llama3.1:8b), OpenRouter fallback (openrouter/meta-llama/llama-3.1-8b-instruct:free), premium tier (anthropic/claude-sonnet-4-20250514), fallback chain, rate limiting
- [x] 1.3 Create `deploy/litellm/README.md`: deployment instructions for LiteLLM Helm chart to k0s
- [x] 1.4 Document `LITELLM_BASE_URL` environment variable (default: `http://litellm:4000`)
- [x] 1.5 Document LiteLLM master key management via Kubernetes Secret

## 2. Ollama Configuration

- [x] 2.1 Create `deploy/ollama/` directory
- [x] 2.2 Create `deploy/ollama/README.md`: deployment instructions for Ollama Helm chart to k0s
- [x] 2.3 Document model pull commands for CI model (qwen2.5:0.5b) and dev model (llama3.1:8b)
- [x] 2.4 Document Ollama service DNS (http://ollama:11434) for LiteLLM configuration

## 3. Virtual Key Lifecycle Activities

- [x] 3.1 Create `internal/litellm/` package directory
- [x] 3.2 Implement `Client` in `internal/litellm/client.go`: HTTP client for LiteLLM Admin API (key/generate, key/delete, spend tracking)
- [x] 3.3 Implement `ProvisionLLMKey` Temporal activity in `internal/temporal/activities.go`: calls LiteLLM `/key/generate` with budget cap and model restrictions based on model tier
- [x] 3.4 Implement `RevokeLLMKey` Temporal activity: calls LiteLLM `/key/delete` to revoke the virtual key
- [x] 3.5 Update `AgentRunWorkflow`: call `ProvisionLLMKey` after pod creation, inject key into pod env
- [x] 3.6 Update `AgentRunWorkflow`: call `RevokeLLMKey` in cleanup (success, failure, cancel, TTL expiry)
- [x] 3.7 Add retry policy with exponential backoff to `ProvisionLLMKey` activity (handle LiteLLM unavailability)

## 4. Agent Pod Environment Injection

- [x] 4.1 Add `LITELLM_BASE_URL` to controller/worker configuration (env var, default `http://litellm:4000`)
- [x] 4.2 Update pod spec construction: add `OPENAI_BASE_URL` env var (value: `$LITELLM_BASE_URL/v1`) to agent container
- [x] 4.3 Update pod spec construction: add `OPENAI_API_KEY` env var (value: provisioned virtual key) to agent container
- [x] 4.4 Add optional `model_tier` field to `AgentRunSpec` proto message (default: `"default"`)
- [x] 4.5 Regenerate proto code after adding `model_tier` field
- [x] 4.6 Update CRD types in `api/v1alpha1/types.go` to include `ModelTier` field

## 5. k0s Deployment Tasks

- [x] 5.1 Add `task k0s:litellm` target: deploys LiteLLM Helm chart to k0s cluster with reference config
- [x] 5.2 Add `task k0s:ollama` target: deploys Ollama Helm chart to k0s cluster
- [x] 5.3 Add `task k0s:ollama:pull` target: pulls models into running Ollama instance
- [x] 5.4 Update `task k0s:deps` (or create it): orchestrates deploying all dependencies (PostgreSQL, Temporal, LiteLLM, Ollama)

## 6. Testing

- [x] 6.1 Write unit tests for `internal/litellm/client.go`: mock HTTP responses for key/generate, key/delete
- [x] 6.2 Write unit tests for `ProvisionLLMKey` and `RevokeLLMKey` activities with mocked LiteLLM client
- [x] 6.3 Write integration test: provision key → verify via LiteLLM API → revoke key → verify revoked
- [x] 6.4 Verify agent pod receives correct `OPENAI_BASE_URL` and `OPENAI_API_KEY` env vars

## 7. Documentation

- [x] 7.1 Update docs/user-guide.md: add LiteLLM section covering configuration, model tiers, spend tracking
- [x] 7.2 Update README.md: add LiteLLM to architecture diagram and dependencies list
- [x] 7.3 Document model tier options and how they map to LiteLLM model routing
