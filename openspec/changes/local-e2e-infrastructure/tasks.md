## 1. Docker Image Build Pipeline

- [x] 1.1 Verify `docker/Dockerfile.hydration` builds successfully (fix Go version if needed)
- [x] 1.2 Verify `docker/Dockerfile.sidecar` builds successfully
- [x] 1.3 Verify `docker/Dockerfile.agent-base` builds successfully
- [x] 1.4 Add `task docker:build` target to Taskfile that builds all 3 images tagged as `aot-{agent,init,sidecar}:local`
- [x] 1.5 Add `task k0s:images` target that exports Docker images to tar and imports via `sudo k0s ctr images import`

## 2. Configurable Image Names

- [x] 2.1 Add env var overrides (`AOT_AGENT_IMAGE`, `AOT_SIDECAR_IMAGE`, `AOT_INIT_IMAGE`) to `internal/temporal/activities.go`
- [x] 2.2 Set `imagePullPolicy: Never` when image name has no registry prefix
- [x] 2.3 Update temporal-worker startup to log which images are configured

## 3. LLM Infrastructure Deployment

- [x] 3.1 Deploy Ollama to k0s via `task k0s:ollama`
- [x] 3.2 Pull qwen2.5:0.5b model into Ollama via port-forward (DNS issues in cluster — kube-router CrashLoopBackOff)
- [x] 3.3 Deploy LiteLLM proxy via `task k0s:litellm` — skipped, E2E tests use Ollama directly via OpenAI-compatible endpoint
- [x] 3.4 Verify Ollama serves inference requests with qwen2.5:0.5b — confirmed via port-forward

## 4. E2E Tests with Real LLM

- [x] 4.1 Design deterministic E2E prompt for lifecycle test (simple file creation task)
- [x] 4.2 Write E2E lifecycle test: create AgentRun → agent pod starts → LLM processes prompt → workflow completes
- [x] 4.3 Design deterministic E2E prompt for HITL test
- [x] 4.4 Write E2E HITL test: agent waits for input → send human input → agent resumes → completes
- [x] 4.5 Write E2E multi-agent test: parent spawns junior → junior completes → parent completes
- [x] 4.6 Add `task test:e2e:infra` target that builds images, imports, deploys LLM deps before running E2E tests
