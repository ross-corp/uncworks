## 1. Rewrite ci/main.go: devboxBase

- [ ] 1.1 Remove `goBase()` function from `ci/main.go`
- [ ] 1.2 Remove `nodeBase()` function from `ci/main.go`
- [ ] 1.3 Add `devboxBase(source *dagger.Directory) *dagger.Container` that uses `jetify/devbox:latest`, copies `devbox.json` and `devbox.lock`, runs `devbox install`, mounts source, and attaches cache volumes for `/nix/store`, Go mod, Go build, and npm
- [ ] 1.4 Verify `dagger call build --source .` works with the new `devboxBase`

## 2. Rewrite ci/main.go: CI functions to use devbox run

- [ ] 2.1 Update `Build` to use `devbox run -- go build ./cmd/... ./internal/...`
- [ ] 2.2 Update `Lint` to use `devbox run -- golangci-lint run` (remove `go install golangci-lint`)
- [ ] 2.3 Update `Test` to use `devbox run -- bash -c '...'` for envtest setup and `go test` (remove `go install setup-envtest`)
- [ ] 2.4 Update `Check` to use `devbox run -- bash -c '...'` for npm ci and tsc --noEmit across web, shared, and extension packages
- [ ] 2.5 Verify `dagger call all --source .` passes with all four functions using devboxBase

## 3. Add image build functions to ci/main.go

- [ ] 3.1 Define `imageSpec` struct and `images` slice with all 5 images (aot-controlplane, aot-init, aot-sidecar, aot-agent, aot-web) mapping to their Dockerfiles and contexts
- [ ] 3.2 Add `BuildImage(ctx, source, name) *dagger.Container` that builds a single image via `dag.Container().DockerBuild()` using the matching Dockerfile and context directory
- [ ] 3.3 Add `BuildImages(ctx, source) []*dagger.Container` that builds all 5 images in parallel
- [ ] 3.4 Add `PushImages(ctx, source, version, registryAuth)` that builds all images and pushes each to `ghcr.io/uncworks/<name>` with semver, major.minor, and SHA tags using `WithRegistryAuth` and `Publish`
- [ ] 3.5 Verify `dagger call build-images --source .` completes without errors (images built but not pushed)

## 4. Add Helm chart functions to ci/main.go

- [ ] 4.1 Add `PackageChart(ctx, source, version) *dagger.File` that uses devboxBase to update Chart.yaml version, run `helm package`, and return the `.tgz` file
- [ ] 4.2 Add `PushChart(ctx, source, version, registryAuth)` that packages the chart and runs `helm push` to `oci://ghcr.io/uncworks/charts`
- [ ] 4.3 Verify `dagger call package-chart --source . --version 0.0.0-test` produces a valid `.tgz`

## 5. Add Release function to ci/main.go

- [ ] 5.1 Add `Release(ctx, source, version, registryAuth)` that runs `All` first, then `PushImages` and `PushChart` in parallel
- [ ] 5.2 Ensure `Release` fails fast if `All` fails, without starting any pushes
- [ ] 5.3 Verify `dagger call release --source . --version 0.0.0-test --registry-auth env:DUMMY` fails gracefully when auth is invalid (no partial publish)

## 6. Update GitHub Actions workflows

- [ ] 6.1 Confirm `.github/workflows/ci.yml` already calls `dagger call all --source .` and needs no changes
- [ ] 6.2 Rewrite `.github/workflows/release-images.yaml` to replace the matrix build with a single `dagger call push-images --source . --version <tag> --registry-auth env:GITHUB_TOKEN`
- [ ] 6.3 Rewrite `.github/workflows/release-chart.yaml` to replace helm install/package/push with `dagger call push-chart --source . --version <tag> --registry-auth env:GITHUB_TOKEN`
- [ ] 6.4 Verify both release workflows pass YAML lint

## 7. Local testing

- [ ] 7.1 Run `dagger call all --source .` locally and confirm all checks pass
- [ ] 7.2 Run `dagger call build-images --source .` locally and confirm all 5 images build
- [ ] 7.3 Run `dagger call package-chart --source . --version 0.0.0-test` locally and verify the output tgz
- [ ] 7.4 Run `dagger call all --source .` a second time to confirm Nix store cache hit (should be significantly faster)

## 8. CI verification

- [ ] 8.1 Push branch and verify CI workflow (`dagger call all`) passes in GitHub Actions
- [ ] 8.2 Create a test tag and verify `release-images.yaml` builds and pushes via Dagger
- [ ] 8.3 Verify `release-chart.yaml` packages and pushes via Dagger on the test tag
- [ ] 8.4 Delete the test tag and any test images/charts from GHCR
