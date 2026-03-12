## Context

The AOT platform has 3 Dockerfiles (`docker/Dockerfile.{agent-base,hydration,sidecar}`) that have never been built. The k0s cluster is running but agent pods fail with `ImagePullBackOff` because images aren't available. Ollama and LiteLLM have Taskfile deployment targets (`k0s:ollama`, `k0s:litellm`) but haven't been deployed. The E2E test framework exists in `e2e/` with working CRD and Temporal tests, but no tests exercise a real agent pod with a real LLM.

## Goals / Non-Goals

**Goals:**
- Build all 3 Docker images locally and make them available in k0s
- Deploy Ollama + LiteLLM to k0s for local LLM inference
- Make image names configurable so tests don't depend on ghcr.io
- Write E2E tests that verify the full agent lifecycle with a real (local) LLM
- Use deterministic prompts that produce verifiable output regardless of model

**Non-Goals:**
- CI/CD image publishing to ghcr.io (future work)
- GPU acceleration for Ollama (CPU-only for local dev)
- Testing with premium/cloud models
- Building a full container registry

## Decisions

1. **Local image import via k0s ctr**: Build images with Docker, export as tar, import via `sudo k0s ctr images import`. No registry needed. Images tagged as `docker.io/library/aot-*:local` to avoid pull attempts.

2. **Image name configuration via environment variables**: Add env var overrides in `internal/temporal/activities.go` (`AOT_AGENT_IMAGE`, `AOT_SIDECAR_IMAGE`, `AOT_INIT_IMAGE`). The temporal-worker reads these at startup. E2E tests set them to point at locally-built images.

3. **qwen2.5:0.5b for E2E**: The smallest Ollama model (~400MB). Fast inference, deterministic enough for structured prompts. LiteLLM's `ci` model alias already points to it.

4. **Deterministic E2E prompts**: Use prompts that ask the agent to create a specific file with specific content (e.g., "create a file called RESULT.txt containing exactly 'PASS'"). Verify the file exists in the pod's workspace. This works regardless of LLM quality.

5. **imagePullPolicy: Never**: Set on pod specs when using locally-imported images to prevent k0s from trying to pull from a registry.

## Risks / Trade-offs

- **qwen2.5:0.5b may not follow complex prompts**: Mitigated by using extremely simple, structured prompts. If the model can't follow "echo PASS > RESULT.txt", the test catches real issues.
- **Image build adds ~2 min to E2E setup**: Acceptable for local dev. Taskfile target caches via Docker layer cache.
- **k0s ctr requires sudo**: Same as k0s setup. Already accepted trade-off.
- **CPU-only Ollama is slow**: First inference may take 30-60s. E2E test timeout is 5 min, sufficient.
