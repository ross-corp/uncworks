## ADDED Requirements

### Requirement: Dagger module uses devbox as the single container base
The Dagger module SHALL use a single `devboxBase()` function that returns a container based on the `jetify/devbox` image with the repo's `devbox.json` installed. The separate `goBase()` and `nodeBase()` functions SHALL be removed.

#### Scenario: devboxBase container has all repo tools
- **WHEN** the `devboxBase()` function is called with the source directory
- **THEN** the returned container SHALL be based on the `jetify/devbox` image
- **THEN** the repo's `devbox.json` and `devbox.lock` SHALL be copied into the container
- **THEN** `devbox install` SHALL have been run, making all declared packages (Go, Node, golangci-lint, buf, helm, setup-envtest, etc.) available on PATH

#### Scenario: devboxBase caches the Nix store
- **WHEN** the `devboxBase()` function builds the container
- **THEN** the `/nix/store` path SHALL be backed by a Dagger `CacheVolume` named `nix-store`
- **THEN** subsequent calls to `devboxBase()` within the same Dagger session SHALL reuse the cached Nix store and skip re-downloading packages

#### Scenario: Go module and build caches are mounted
- **WHEN** the `devboxBase()` function builds the container
- **THEN** the Go module cache (`/home/devbox/.cache/go/mod`) SHALL be backed by a Dagger `CacheVolume` named `go-mod`
- **THEN** the Go build cache (`/home/devbox/.cache/go-build`) SHALL be backed by a Dagger `CacheVolume` named `go-build`

#### Scenario: npm cache is mounted
- **WHEN** the `devboxBase()` function builds the container
- **THEN** the npm cache directory SHALL be backed by a Dagger `CacheVolume` named `npm-cache`

### Requirement: All Dagger CI functions use devbox run
All CI functions (Build, Lint, Test, Check) SHALL invoke tools through `devbox run` instead of calling binaries directly, ensuring the devbox environment (PATH, env vars) is active.

#### Scenario: Build uses devbox run
- **WHEN** `Build` is called
- **THEN** the container SHALL execute `devbox run -- go build ./cmd/... ./internal/...`

#### Scenario: Lint uses devbox run
- **WHEN** `Lint` is called
- **THEN** the container SHALL execute `devbox run -- golangci-lint run`
- **THEN** golangci-lint SHALL NOT be installed via `go install` at runtime

#### Scenario: Test uses devbox run
- **WHEN** `Test` is called
- **THEN** the container SHALL execute envtest setup and `go test` through `devbox run`
- **THEN** `setup-envtest` SHALL come from the devbox environment, NOT from `go install`

#### Scenario: Check uses devbox run
- **WHEN** `Check` is called
- **THEN** the container SHALL execute `npm ci` and `tsc --noEmit` for web, shared, and extension packages through `devbox run`

### Requirement: All function produces identical results locally and in CI
The `All` function SHALL produce the same pass/fail result regardless of whether it is invoked locally via `dagger call all --source .` or in GitHub Actions via the dagger-for-github action, because both paths use the same devbox environment.

#### Scenario: Local invocation
- **WHEN** a developer runs `dagger call all --source .` from the repo root
- **THEN** the function SHALL run Build, Lint, Test, and Check in parallel using the devbox container
- **THEN** the result SHALL match what CI produces

#### Scenario: CI invocation
- **WHEN** GitHub Actions runs `dagger call all --source .` via `dagger/dagger-for-github`
- **THEN** the function SHALL use the same devbox container and produce the same result as local invocation
