// UNCWORKS CI pipeline — Dagger module
//
// All checks run inside devbox containers for reproducible tooling.
// The devbox.json at the repo root defines exact tool versions.
//
// Usage:
//   dagger call build --source .
//   dagger call lint --source .
//   dagger call test --source .
//   dagger call check --source .
//   dagger call all --source .
//   dagger call build-images --source .
//   dagger call push-images --source . --version v1.0.0 --registry-user USER --registry-pass env:TOKEN
//   dagger call release --source . --version v1.0.0 --registry-user USER --registry-pass env:TOKEN

package main

import (
	"context"
	"dagger/ci/internal/dagger"
	"fmt"
	"strings"
)

type Ci struct{}

// imageSpec defines a Docker image to build.
type imageSpec struct {
	Name       string
	Dockerfile string
	Context    string // relative to source root
}

var images = []imageSpec{
	{Name: "aot-controlplane", Dockerfile: "docker/Dockerfile.controlplane", Context: "."},
	{Name: "aot-init", Dockerfile: "docker/Dockerfile.hydration", Context: "."},
	{Name: "aot-sidecar", Dockerfile: "docker/Dockerfile.sidecar", Context: "."},
	{Name: "aot-agent", Dockerfile: "docker/Dockerfile.agent-base", Context: "."},
	{Name: "aot-web", Dockerfile: "docker/Dockerfile.web", Context: "web"},
	{Name: "aot-bff", Dockerfile: "docker/Dockerfile.bff", Context: "."},
	{Name: "aot-cudgel-shim", Dockerfile: "docker/Dockerfile.cudgel-shim", Context: "."},
}

// goBase returns a Go container with the source mounted and modules cached.
// TODO: Replace with devbox-in-Dagger once Nix daemon-less containers are solved.
// Versions are kept in sync with devbox.json manually.
func (m *Ci) goBase(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithEnvVariable("CGO_ENABLED", "0").
		// Stub dist/ so //go:embed dist/* in cmd/bff compiles without the real frontend build
		WithExec([]string{"bash", "-c", "mkdir -p cmd/bff/dist && echo placeholder > cmd/bff/dist/index.html"})
}

// nodeBase returns a Node.js container with the source mounted and npm cached.
func (m *Ci) nodeBase(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("node:22-bookworm-slim").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/root/.npm", dag.CacheVolume("npm-cache"))
}

// helmBase returns a Go container with Helm installed for chart operations.
func (m *Ci) helmBase(source *dagger.Directory) *dagger.Container {
	return m.goBase(source).
		WithExec([]string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"})
}

// Build compiles all Go binaries.
func (m *Ci) Build(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := m.goBase(source).
		WithExec([]string{"go", "build", "./cmd/...", "./internal/..."}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}
	return "go build: ok", nil
}

// Lint runs golangci-lint with timeout and reduced concurrency for CI runners.
func (m *Ci) Lint(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := m.goBase(source).
		WithExec([]string{"go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"}).
		WithExec([]string{"golangci-lint", "run", "--timeout", "5m", "--concurrency", "2"}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("golangci-lint failed: %w", err)
	}
	return "golangci-lint: ok", nil
}

// Test runs Go unit tests with envtest.
func (m *Ci) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	out, err := m.goBase(source).
		WithExec([]string{"go", "install", "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"}).
		WithExec([]string{"bash", "-c", `
			ENVTEST_PATH=$(setup-envtest use --print path 2>/dev/null || echo "")
			if [ -n "$ENVTEST_PATH" ]; then
				export KUBEBUILDER_ASSETS="$ENVTEST_PATH"
			fi
			go test $(go list ./api/... ./internal/... | grep -v /brain | grep -v /embeddings) -count=1
		`}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("go test failed: %w", err)
	}
	return out, nil
}

// Check runs TypeScript type checking.
// Installs deps for shared first (web imports from it via relative path),
// then checks all three packages sequentially.
func (m *Ci) Check(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := m.nodeBase(source).
		WithExec([]string{"bash", "-c", `
			set -e
			# Install @bufbuild/protobuf at repo root so gen/ts/ proto files
			# can resolve it (they live outside any package's node_modules).
			cd /src && npm install --no-save @bufbuild/protobuf@^2.0.0
			cd /src/packages/shared && npm ci
			cd /src/packages/pi-aot-extension && npm ci
			cd /src/web && npm ci
			cd /src/packages/shared && npx tsc --noEmit
			cd /src/packages/pi-aot-extension && npx tsc --noEmit
			cd /src/web && npx tsc --noEmit
		`}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("typescript check failed: %w", err)
	}
	return "tsc: ok", nil
}

// All runs all checks in parallel: build, lint, test, typescript.
func (m *Ci) All(ctx context.Context, source *dagger.Directory) (string, error) {
	type result struct {
		name string
		err  error
	}
	ch := make(chan result, 4)

	go func() { _, err := m.Build(ctx, source); ch <- result{"build", err} }()
	go func() { _, err := m.Lint(ctx, source); ch <- result{"lint", err} }()
	go func() { _, err := m.Test(ctx, source); ch <- result{"test", err} }()
	go func() { _, err := m.Check(ctx, source); ch <- result{"check", err} }()

	var failures []string
	for range 4 {
		r := <-ch
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.name, r.err))
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("CI failed:\n%s", joinLines(failures))
	}
	return "all checks passed", nil
}

// BuildImage builds a single Docker image by name.
func (m *Ci) BuildImage(source *dagger.Directory, name string) *dagger.Container {
	for _, img := range images {
		if img.Name == name {
			contextDir := source
			if img.Context != "." {
				contextDir = source.Directory(img.Context)
			}
			return contextDir.DockerBuild(dagger.DirectoryDockerBuildOpts{
				Dockerfile: img.Dockerfile,
			})
		}
	}
	return nil
}

// BuildImages builds all 5 Docker images in parallel.
func (m *Ci) BuildImages(ctx context.Context, source *dagger.Directory) (string, error) {
	type result struct {
		name string
		err  error
	}
	ch := make(chan result, len(images))

	for _, img := range images {
		img := img
		go func() {
			contextDir := source
			if img.Context != "." {
				contextDir = source.Directory(img.Context)
			}
			_, err := contextDir.DockerBuild(dagger.DirectoryDockerBuildOpts{
				Dockerfile: img.Dockerfile,
			}).Sync(ctx)
			ch <- result{img.Name, err}
		}()
	}

	var failures []string
	for range len(images) {
		r := <-ch
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.name, r.err))
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("image build failed:\n%s", joinLines(failures))
	}
	return fmt.Sprintf("built %d images", len(images)), nil
}

// PushImages builds and pushes all images to a container registry.
func (m *Ci) PushImages(
	ctx context.Context,
	source *dagger.Directory,
	version string,
	registryUser string,
	registryPass *dagger.Secret,
	// +optional
	// +default="ghcr.io/uncworks"
	registry string,
) (string, error) {
	type result struct {
		name string
		ref  string
		err  error
	}
	ch := make(chan result, len(images))

	for _, img := range images {
		img := img
		go func() {
			contextDir := source
			if img.Context != "." {
				contextDir = source.Directory(img.Context)
			}
			container := contextDir.DockerBuild(dagger.DirectoryDockerBuildOpts{
				Dockerfile: img.Dockerfile,
			})

			ref := fmt.Sprintf("%s/%s:%s", registry, img.Name, version)
			addr, err := container.
				WithRegistryAuth(registry, registryUser, registryPass).
				Publish(ctx, ref)
			ch <- result{img.Name, addr, err}
		}()
	}

	var published []string
	var failures []string
	for range len(images) {
		r := <-ch
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.name, r.err))
		} else {
			published = append(published, r.ref)
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("push failed:\n%s", joinLines(failures))
	}
	return fmt.Sprintf("pushed %d images:\n%s", len(published), joinLines(published)), nil
}

// PushEdge builds all images and publishes them with :edge and :sha-{sha7} tags.
// Called on every push to main for a rolling "latest main" image channel.
func (m *Ci) PushEdge(
	ctx context.Context,
	source *dagger.Directory,
	registryUser string,
	registryPass *dagger.Secret,
	// +optional
	// +default="ghcr.io/uncworks"
	registry string,
) (string, error) {
	// Derive short SHA from git inside a container.
	sha7, err := dag.Container().
		From("alpine/git:latest").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"git", "rev-parse", "--short=7", "HEAD"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	sha7 = strings.TrimSpace(sha7)

	type result struct {
		name string
		refs []string
		err  error
	}
	ch := make(chan result, len(images))

	for _, img := range images {
		img := img
		go func() {
			contextDir := source
			if img.Context != "." {
				contextDir = source.Directory(img.Context)
			}
			container := contextDir.DockerBuild(dagger.DirectoryDockerBuildOpts{
				Dockerfile: img.Dockerfile,
			})

			var refs []string
			var pushErr error
			for _, tag := range []string{"edge", "sha-" + sha7} {
				ref := fmt.Sprintf("%s/%s:%s", registry, img.Name, tag)
				if _, err := container.
					WithRegistryAuth(registry, registryUser, registryPass).
					Publish(ctx, ref); err != nil {
					pushErr = err
					break
				}
				refs = append(refs, ref)
			}
			ch <- result{img.Name, refs, pushErr}
		}()
	}

	var published []string
	var failures []string
	for range len(images) {
		r := <-ch
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.name, r.err))
		} else {
			published = append(published, r.refs...)
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("edge push failed:\n%s", joinLines(failures))
	}
	return fmt.Sprintf("pushed %d refs (sha: %s):\n%s", len(published), sha7, joinLines(published)), nil
}

// PackageChart packages the Helm chart with the given version.
func (m *Ci) PackageChart(ctx context.Context, source *dagger.Directory, version string) *dagger.File {
	return m.helmBase(source).
		WithExec([]string{"bash", "-c", fmt.Sprintf(`
			cd /src/deploy/helm/aot
			sed -i "s/^version:.*/version: %s/" Chart.yaml
			sed -i "s/^appVersion:.*/appVersion: \"%s\"/" Chart.yaml
			helm package .
		`, version, version)}).
		File(fmt.Sprintf("/src/deploy/helm/aot/aot-%s.tgz", version))
}

// PushChart packages and pushes the Helm chart to an OCI registry.
func (m *Ci) PushChart(
	ctx context.Context,
	source *dagger.Directory,
	version string,
	registryUser string,
	registryPass *dagger.Secret,
	// +optional
	// +default="oci://ghcr.io/uncworks/charts"
	registry string,
) (string, error) {
	out, err := m.helmBase(source).
		WithSecretVariable("HELM_REGISTRY_PASS", registryPass).
		WithExec([]string{"bash", "-c", fmt.Sprintf(`
			cd /src/deploy/helm/aot
			sed -i "s/^version:.*/version: %s/" Chart.yaml
			sed -i "s/^appVersion:.*/appVersion: \"%s\"/" Chart.yaml
			helm package .
			echo "$HELM_REGISTRY_PASS" | helm registry login ghcr.io --username %s --password-stdin
			helm push aot-%s.tgz %s
		`, version, version, registryUser, version, registry)}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("chart push failed: %w", err)
	}
	return out, nil
}

// Release runs all checks, then builds and pushes images and chart.
// Fails fast if checks fail — no images or charts are published.
func (m *Ci) Release(
	ctx context.Context,
	source *dagger.Directory,
	version string,
	registryUser string,
	registryPass *dagger.Secret,
	// +optional
	// +default="ghcr.io/uncworks"
	imageRegistry string,
	// +optional
	// +default="oci://ghcr.io/uncworks/charts"
	chartRegistry string,
) (string, error) {
	// Gate: all checks must pass first
	if _, err := m.All(ctx, source); err != nil {
		return "", fmt.Errorf("checks failed, release aborted: %w", err)
	}

	// Publish images and chart in parallel
	type result struct {
		name string
		out  string
		err  error
	}
	ch := make(chan result, 2)

	go func() {
		out, err := m.PushImages(ctx, source, version, registryUser, registryPass, imageRegistry)
		ch <- result{"images", out, err}
	}()
	go func() {
		out, err := m.PushChart(ctx, source, version, registryUser, registryPass, chartRegistry)
		ch <- result{"chart", out, err}
	}()

	var failures []string
	for range 2 {
		r := <-ch
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.name, r.err))
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("release failed:\n%s", joinLines(failures))
	}
	return fmt.Sprintf("release %s complete", version), nil
}

// BuildBinaries cross-compiles the uncworks CLI for all supported platforms.
// Returns a directory containing binaries named uncworks-<os>-<arch>.
func (m *Ci) BuildBinaries(ctx context.Context, source *dagger.Directory, version string) *dagger.Directory {
	type platform struct{ goos, goarch string }
	platforms := []platform{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	}

	out := dag.Directory()
	for _, p := range platforms {
		name := fmt.Sprintf("uncworks-%s-%s", p.goos, p.goarch)
		binary := m.goBase(source).
			WithEnvVariable("GOOS", p.goos).
			WithEnvVariable("GOARCH", p.goarch).
			WithEnvVariable("CGO_ENABLED", "0").
			WithExec([]string{"go", "build", "-ldflags", fmt.Sprintf("-X main.version=%s", version),
				"-o", "/out/" + name, "./cmd/uncworks"}).
			File("/out/" + name)
		out = out.WithFile(name, binary)
	}
	return out
}

// ReleaseBinaries builds cross-platform binaries and the Helm chart for a release.
// Used by the GitHub Actions release workflow to produce release assets.
func (m *Ci) ReleaseBinaries(ctx context.Context, source *dagger.Directory, version string) *dagger.Directory {
	return m.BuildBinaries(ctx, source, version)
}

func joinLines(ss []string) string {
	out := ""
	for _, s := range ss {
		out += "  - " + s + "\n"
	}
	return out
}
