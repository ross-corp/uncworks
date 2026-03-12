## Why

The E2E test pipeline is 50/55 tasks complete but the final 5 tests (lifecycle, HITL, multi-agent with real LLM) cannot run because the agent pod images have never been built and the LLM infrastructure (Ollama + LiteLLM) is not deployed to the k0s cluster. The Dockerfiles, Helm charts, and Taskfile targets all exist but have never been exercised end-to-end.

## What Changes

- Build all 3 agent pod Docker images locally and import into k0s
- Deploy Ollama with a small model (qwen2.5:0.5b) for deterministic CI testing
- Deploy LiteLLM proxy pointing at Ollama
- Add Taskfile targets for image build + import
- Write 5 E2E tests: agent lifecycle with real LLM, HITL flow, multi-agent spawn, and deterministic prompt design for each
- Make image names configurable via env vars so tests can use locally-built images

## Capabilities

### New Capabilities

- `local-image-pipeline`: Build, tag, and import Docker images (hydration, sidecar, agent-base) into k0s without a registry
- `e2e-llm-tests`: E2E tests that exercise real agent pods with a local LLM (lifecycle, HITL, multi-agent) using deterministic prompts

### Modified Capabilities

## Impact

- `docker/Dockerfile.*` — may need Go version fix (1.25-bookworm availability)
- `internal/temporal/activities.go` — image names need to be configurable (env vars)
- `e2e/` — new test files with `//go:build e2e` tag
- `Taskfile.yml` — new build/import targets
- `deploy/` — Ollama + LiteLLM deployment verified working
