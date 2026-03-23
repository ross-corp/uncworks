## Why

The CI pipeline, local development, and Docker image builds use three different toolchains. CI runs in `golang:1.25-bookworm` and `node:22-bookworm-slim` containers via Dagger. Local dev uses devbox with pinned versions of Go, Node, golangci-lint, buf, etc. Docker images use their own multi-stage Dockerfiles. This means a test can pass locally (devbox Go version) but fail in CI (container Go version), or a lint rule can differ between lefthook (devbox golangci-lint) and Dagger (go install latest).

Devbox should be the single source of truth for tooling. The Dagger module should run inside devbox, and the same `dagger call all` command should work both locally and in GitHub Actions — producing identical results because the environment is identical.

## What Changes

- Replace `goBase()` and `nodeBase()` in the Dagger module with a single `devboxBase()` that uses `jetify/devbox` as the container image and runs `devbox install` with Nix store caching
- All Dagger functions (`Build`, `Lint`, `Test`, `Check`) use `devbox run` instead of raw tool invocations
- Add Docker image building to the Dagger module — `BuildImages()` builds all 5 images using existing Dockerfiles via `DockerBuild`
- Add `PushImages()` for pushing to GHCR from Dagger
- Add Helm chart packaging to Dagger — `PackageChart()` and `PushChart()`
- Simplify GitHub Actions to just `dagger call all` for CI and `dagger call release` for releases
- Keep lefthook for fast local pre-commit feedback (runs devbox tools directly)
- Local usage: `dagger call all --source .` runs the full pipeline in devbox containers

## Capabilities

### New Capabilities
- `dagger-devbox-base`: Dagger container base using jetify/devbox with Nix store caching and devbox.json from the repo
- `dagger-image-build`: Build all 5 Docker images through Dagger using existing Dockerfiles
- `dagger-release`: Full release pipeline (all checks + image build + chart package + push) as a single Dagger function

### Modified Capabilities
- None

## Impact

- **Modified**: `ci/main.go` — replace goBase/nodeBase with devboxBase, add image and chart functions
- **Modified**: `.github/workflows/ci.yml` — simplify to single `dagger call all`
- **Modified**: `.github/workflows/release-images.yaml` — replace matrix docker builds with `dagger call push-images`
- **Modified**: `.github/workflows/release-chart.yaml` — replace helm commands with `dagger call push-chart`
- **Unchanged**: `devbox.json` — already has all required packages
- **Unchanged**: `lefthook.yml` — kept for fast local hooks
- **Unchanged**: `Taskfile.yml` — kept for local dev convenience
- **Unchanged**: `docker/Dockerfile.*` — kept, referenced by Dagger's DockerBuild
