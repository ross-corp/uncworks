# LiteLLM Proxy Deployment

LiteLLM is an OpenAI-compatible proxy that provides centralized LLM routing, per-agent virtual keys, and spend tracking. All agent pods route LLM calls through LiteLLM transparently via `OPENAI_BASE_URL`.

## Prerequisites

- k0s cluster running (`sudo ./hack/k0s-setup.sh`)
- Helm 3 installed
- PostgreSQL accessible in-cluster (for virtual key storage)

## Deploy to k0s

```bash
# Add the LiteLLM Helm repo
helm repo add litellm https://litellm-helm.github.io/litellm-helm
helm repo update

# Create the litellm database in PostgreSQL
kubectl exec -it deploy/postgres -- psql -U postgres -c "CREATE DATABASE litellm;"
kubectl exec -it deploy/postgres -- psql -U postgres -c "CREATE USER litellm WITH PASSWORD 'litellm'; GRANT ALL ON DATABASE litellm TO litellm;"

# Create the master key secret
kubectl create secret generic litellm-master-key \
  --from-literal=LITELLM_MASTER_KEY="sk-aot-$(openssl rand -hex 16)"

# Deploy LiteLLM
helm install litellm litellm/litellm-helm \
  --set masterKeySecretName=litellm-master-key \
  --set masterKeySecretKey=LITELLM_MASTER_KEY \
  --set-file configFile=litellm-config.yaml

# Or with Taskfile
task k0s:litellm
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `LITELLM_BASE_URL` | `http://litellm:4000` | LiteLLM proxy base URL. Controller and worker read this to construct `OPENAI_BASE_URL` for agent pods. |
| `LITELLM_MASTER_KEY` | (required) | Admin API key for virtual key provisioning. Stored as a Kubernetes Secret. Must NOT be exposed to agent pods. |

### Model Tiers

| Tier | Model | Provider | Cost |
|---|---|---|---|
| `default` | llama3.1:8b | Ollama (local) | Free |
| `default-cloud` | llama-3.1-8b-instruct:free | OpenRouter | Free (rate-limited) |
| `premium` | claude-sonnet-4-20250514 | Anthropic | Paid |
| `ci` | qwen2.5:0.5b | Ollama (local) | Free |

### Fallback Chain

`default` → `default-cloud`: If Ollama is unavailable, requests fall through to OpenRouter free tier automatically.

### Master Key Management

The LiteLLM master key grants full admin API access (key provisioning, deletion, config changes). It MUST be:

1. Stored as a Kubernetes Secret (`litellm-master-key`)
2. Only accessible to the controller and Temporal worker (via `LITELLM_MASTER_KEY` env var)
3. Never mounted into agent pods — agents only receive their per-run virtual key

```bash
# Rotate the master key
kubectl create secret generic litellm-master-key \
  --from-literal=LITELLM_MASTER_KEY="sk-aot-$(openssl rand -hex 16)" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart LiteLLM to pick up the new key
kubectl rollout restart deploy/litellm
```

## Verify

```bash
# Check LiteLLM is running
kubectl get pods -l app=litellm

# Health check
curl http://litellm:4000/health

# List configured models
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" http://litellm:4000/model/info
```
