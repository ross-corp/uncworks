# Ollama Deployment

Ollama serves local LLM models as the primary backend for LiteLLM. Running models locally means zero API costs and no external dependencies for development.

## Prerequisites

- k0s cluster running
- Helm 3 installed

## Deploy to k0s

```bash
# Add the Ollama Helm repo
helm repo add ollama-helm https://otwld.github.io/ollama-helm
helm repo update

# Deploy Ollama
helm install ollama ollama-helm/ollama \
  --set ollama.gpu.enabled=false \
  --set resources.requests.memory=8Gi

# Or with Taskfile
task k0s:ollama
```

## Pull Models

Models must be pulled after deployment. The pull is a post-deployment step to avoid blocking on large downloads.

```bash
# CI/testing model (tiny, fast inference, ~400MB)
kubectl exec -it deploy/ollama -- ollama pull qwen2.5:0.5b

# Development model (capable, ~4.7GB)
kubectl exec -it deploy/ollama -- ollama pull llama3.1:8b

# Or with Taskfile
task k0s:ollama:pull
```

## Service DNS

Ollama is accessible within the cluster at:

```
http://ollama:11434
```

This is the `api_base` value used in `deploy/litellm/litellm-config.yaml` for Ollama-backed models.

## Verify

```bash
# Check Ollama is running
kubectl get pods -l app.kubernetes.io/name=ollama

# List available models
kubectl exec -it deploy/ollama -- ollama list

# Test inference
kubectl exec -it deploy/ollama -- ollama run qwen2.5:0.5b "Hello"
```

## Resource Requirements

| Model | RAM | Disk |
|---|---|---|
| qwen2.5:0.5b | ~1GB | ~400MB |
| llama3.1:8b | ~8GB | ~4.7GB |

If resources are limited, skip Ollama — the LiteLLM fallback chain routes to OpenRouter free tier automatically.
