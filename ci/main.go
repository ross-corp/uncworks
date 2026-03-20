// UNCWORKS CI pipeline — Dagger module
//
// All checks run in parallel:
//   - Go build
//   - Go lint (golangci-lint v2)
//   - Go tests (with envtest for controller)
//   - TypeScript checks (web, shared, extension)
//
// Usage:
//   dagger call check --source .
//   dagger call lint --source .
//   dagger call test --source .
//   dagger call all --source .

package main

import (
	"context"
	"dagger/ci/internal/dagger"
	"fmt"
)

type Ci struct{}

// goBase returns a Go container with the source mounted and modules cached.
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

// Lint runs golangci-lint v2.
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

// Test runs Go unit tests with envtest for controller tests.
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
	// Run all checks in parallel using goroutines
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

func joinLines(ss []string) string {
	out := ""
	for _, s := range ss {
		out += "  - " + s + "\n"
	}
	return out
}
