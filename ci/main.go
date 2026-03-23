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
}

// goBase returns a Go container with the source mounted and modules cached.
// Versions match devbox.json: go@latest (1.25), golangci-lint@latest.
func (m *Ci) goBase(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithEnvVariable("CGO_ENABLED", "0")
}

// nodeBase returns a Node.js container with the source mounted and npm cached.
// Versions match devbox.json: nodejs@22.
func (m *Ci) nodeBase(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("node:22-bookworm-slim").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/root/.npm", dag.CacheVolume("npm-cache"))
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

// Lint runs golangci-lint.
func (m *Ci) Lint(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := m.goBase(source).
		WithExec([]string{"go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"}).
		WithExec([]string{"golangci-lint", "run"}).
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
			go test ./api/... ./internal/... -count=1
		`}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("go test failed: %w", err)
	}
	return out, nil
}

// Check runs TypeScript type checking for web, shared, and extension packages.
func (m *Ci) Check(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := m.nodeBase(source).
		WithExec([]string{"bash", "-c", `
			cd /src/web && npm ci --ignore-scripts && npx tsc --noEmit &&
			cd /src/packages/shared && npm ci --ignore-scripts && npx tsc --noEmit &&
			cd /src/packages/pi-aot-extension && npm ci --ignore-scripts && npx tsc --noEmit
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

// PackageChart packages the Helm chart with the given version.
func (m *Ci) PackageChart(ctx context.Context, source *dagger.Directory, version string) *dagger.File {
	return m.goBase(source).
		WithExec([]string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"}).
		WithExec([]string{"devbox", "run", "--", "bash", "-c", fmt.Sprintf(`
			cd /src/deploy/helm/aot
			sed -i "s/^version:.*/version: %s/" Chart.yaml
			sed -i "s/^appVersion:.*/appVersion: %s/" Chart.yaml
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
	out, err := m.goBase(source).
		WithExec([]string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"}).
		WithExec([]string{"devbox", "run", "--", "bash", "-c", fmt.Sprintf(`
			cd /src/deploy/helm/aot
			sed -i "s/^version:.*/version: %s/" Chart.yaml
			sed -i "s/^appVersion:.*/appVersion: %s/" Chart.yaml
			helm package .
			helm push aot-%s.tgz %s
		`, version, version, version, registry)}).
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

func joinLines(ss []string) string {
	out := ""
	for _, s := range ss {
		out += "  - " + s + "\n"
	}
	return out
}
