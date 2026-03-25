## Context

The CI pipeline currently uses two separate container bases: `golang:1.25-bookworm` for Go operations and `node:22-bookworm-slim` for TypeScript checks. These containers install tools at runtime (`go install golangci-lint@latest`, `go install setup-envtest@latest`), meaning CI tool versions drift from what devbox provides locally. Docker image builds happen entirely outside Dagger via GitHub Actions matrix builds with `docker/build-push-action`. Helm chart packaging also runs outside Dagger in a separate workflow.

The result: three different environments (devbox local, Dagger CI, GitHub Actions release) that can disagree on tool versions and behavior.

## Goals / Non-Goals

**Goals:**
- Single environment: devbox is the source of truth for all tooling, used inside Dagger containers.
- `dagger call all --source .` produces identical results locally and in CI.
- Docker image builds and Helm chart packaging move into the Dagger module.
- GitHub Actions workflows become thin wrappers around `dagger call`.
- Nix store and Go/npm caches are persisted across Dagger runs for fast iteration.

**Non-Goals:**
- Changing the existing Dockerfiles (they are reused as-is via `DockerBuild`).
- Removing lefthook or Taskfile (kept for fast local pre-commit and dev convenience).
- Multi-platform image builds (single linux/amd64 for now).
- Moving Release Please itself into Dagger (it stays as a GitHub Action).

## Decisions

### 1. devboxBase() replaces goBase() and nodeBase()

A single `devboxBase(source)` function replaces both container bases. It uses the `jetify/devbox` image, copies in `devbox.json` and `devbox.lock`, runs `devbox install`, then mounts the full source. All downstream functions call `devbox run -- <command>` to execute within the devbox environment.

```go
func (m *Ci) devboxBase(source *dagger.Directory) *dagger.Container {
    return dag.Container().
        From("jetify/devbox:latest").
        // Cache the Nix store across runs
        WithMountedCache("/nix/store", dag.CacheVolume("nix-store")).
        // Copy devbox config first (for layer caching)
        WithMountedFile("/src/devbox.json", source.File("devbox.json")).
        WithMountedFile("/src/devbox.lock", source.File("devbox.lock")).
        WithWorkdir("/src").
        WithExec([]string{"devbox", "install"}).
        // Now mount the full source
        WithMountedDirectory("/src", source).
        // Cache Go and npm
        WithMountedCache("/home/devbox/.cache/go/mod", dag.CacheVolume("go-mod")).
        WithMountedCache("/home/devbox/.cache/go-build", dag.CacheVolume("go-build")).
        WithMountedCache("/home/devbox/.npm", dag.CacheVolume("npm-cache")).
        WithEnvVariable("CGO_ENABLED", "0")
}
```

**Rationale**: The `jetify/devbox` image already has Nix installed and the `devbox` CLI. By copying `devbox.json`/`devbox.lock` first and running `devbox install` before mounting the full source, Dagger's layer cache ensures the Nix install step is skipped when only source code changes. This keeps incremental builds fast.

### 2. Nix store cache strategy

The `/nix/store` directory is large (often 1-2 GB after installing all packages) and immutable by design -- Nix store paths are content-addressed and never modified. This makes it a perfect candidate for a Dagger `CacheVolume`:

- **First run**: `devbox install` downloads and installs all Nix packages. The entire `/nix/store` is captured in the `nix-store` cache volume.
- **Subsequent runs**: The cache volume is mounted before `devbox install` runs. Nix sees all packages already present and completes in seconds.
- **devbox.lock changes**: If a developer updates `devbox.lock` (e.g., bumps Go version), `devbox install` installs only the new/changed packages. Old packages remain in the cache volume (Nix garbage collection is not run in CI).

The Go module cache and Go build cache are mounted separately at the devbox user's home paths, matching where `devbox run -- go` writes them.

### 3. devbox run replaces raw tool calls

Every CI function wraps its commands in `devbox run --`:

```go
// Before (raw tool call, tool installed at runtime):
WithExec([]string{"go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"}).
WithExec([]string{"golangci-lint", "run"})

// After (devbox provides the tool):
WithExec([]string{"devbox", "run", "--", "golangci-lint", "run"})
```

This ensures all tools come from the Nix packages declared in `devbox.json`, pinned by `devbox.lock`. No `go install` at runtime. The devbox environment also sets `GOPATH`, `GOROOT`, `NODE_PATH`, etc. correctly.

For multi-command steps (like the Test function that sets up envtest then runs tests), a bash wrapper is used:

```go
WithExec([]string{"devbox", "run", "--", "bash", "-c", `
    ENVTEST_PATH=$(setup-envtest use --print path 2>/dev/null || echo "")
    if [ -n "$ENVTEST_PATH" ]; then
        export KUBEBUILDER_ASSETS="$ENVTEST_PATH"
    fi
    go test ./api/... ./internal/... -count=1
`})
```

### 4. DockerBuild for images

Dagger's `Container.DockerBuild()` method builds images from existing Dockerfiles without requiring `docker` in the container. Each image is defined as a struct:

```go
type imageSpec struct {
    name       string
    dockerfile string
    context    string
}

var images = []imageSpec{
    {"aot-controlplane", "docker/Dockerfile.controlplane", "."},
    {"aot-init", "docker/Dockerfile.hydration", "."},
    {"aot-sidecar", "docker/Dockerfile.sidecar", "."},
    {"aot-agent", "docker/Dockerfile.agent-base", "."},
    {"aot-web", "docker/Dockerfile.web", "web/"},
}
```

`BuildImages` iterates over the specs and calls `dag.Container().DockerBuild(contextDir, dagger.ContainerDockerBuildOpts{Dockerfile: spec.dockerfile})` for each. All 5 builds run in parallel using goroutines.

`PushImages` builds all images, then calls `container.WithRegistryAuth(...).Publish(ctx, ref)` for each, tagging with semver, major.minor, and git SHA.

### 5. Helm chart packaging in Dagger

`PackageChart` uses the devbox base (which includes `helm`) to:
1. Rewrite the version in `Chart.yaml` using `sed`.
2. Run `helm package deploy/helm/aot`.
3. Return the `.tgz` file.

`PushChart` extends this by running `helm push` inside the devbox container with registry credentials injected via `WithSecretVariable`.

### 6. Release pipeline composition

The `Release` function composes everything:

```
Release(source, version, registryAuth)
  |
  +-- All(source)                    [fail-fast: abort if checks fail]
  |     +-- Build (parallel)
  |     +-- Lint  (parallel)
  |     +-- Test  (parallel)
  |     +-- Check (parallel)
  |
  +-- PushImages(source, version, registryAuth)   [parallel after All]
  +-- PushChart(source, version, registryAuth)    [parallel after All]
```

The checks gate the release. If any check fails, no artifacts are published. After checks pass, image pushes and chart push run in parallel.

### 7. Simplified GitHub Actions

**ci.yml** stays almost the same -- it already calls `dagger call all --source .`.

**release-images.yaml** is simplified from a 5-job matrix to:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: dagger/dagger-for-github@v8.4.0
    with:
      version: latest
      module: ./ci
      call: push-images --source . --version ${{ github.ref_name }} --registry-auth env:GITHUB_TOKEN
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**release-chart.yaml** is simplified similarly:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: dagger/dagger-for-github@v8.4.0
    with:
      version: latest
      module: ./ci
      call: push-chart --source . --version ${{ github.ref_name }} --registry-auth env:GITHUB_TOKEN
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Alternatively, both can be replaced with a single `dagger call release` invocation. This is deferred until the per-function approach is validated.

## Risks / Trade-offs

- **[Nix store cache size]** The Nix store can grow large as packages are added but never garbage-collected. In Dagger Cloud or self-hosted runners with persistent caches, this is a non-issue. On ephemeral GitHub Actions runners, the cache volume is rebuilt each run unless Dagger's built-in caching persists it. Mitigation: the `devbox.lock` file ensures deterministic installs, and the layer-cache strategy (devbox config before source) minimizes rebuilds.
- **[devbox install cold-start time]** First run on a clean cache takes 60-120 seconds to install all Nix packages. Subsequent runs with a warm cache take 2-5 seconds. This is slower than pulling `golang:1.25-bookworm` (~15s) but eliminates version drift.
- **[jetify/devbox image updates]** The `jetify/devbox:latest` tag may change. Pinning to a specific digest or version tag (e.g., `jetify/devbox:0.13`) is recommended once the initial implementation is stable.
- **[DockerBuild Dockerfile.web context]** The `aot-web` image uses `web/` as its context, not the repo root. The source directory passed to `DockerBuild` must be `source.Directory("web/")` for that image, while all others use the full source.
- **[Secret handling for GHCR]** Dagger secrets are passed by reference and never written to container layers or logs. The `registryAuth` parameter uses `dagger.Secret` type, and GitHub Actions passes it via `env:GITHUB_TOKEN` which Dagger reads from the environment without exposing it.
