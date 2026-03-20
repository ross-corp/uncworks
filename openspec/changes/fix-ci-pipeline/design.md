## CI Pipeline Design

### Current Issues

1. **envtest setup swallows errors**: `setup-envtest use --print path -p env > "$GITHUB_ENV" || true` — the `|| true` masks failures, and piping to `$GITHUB_ENV` doesn't actually set `KUBEBUILDER_ASSETS` correctly
2. **Missing npm installs**: Web UI TypeScript check not in CI. Shared package does `npm install` inline but web doesn't
3. **buf on push**: `buf breaking` only runs on PRs but `buf lint` runs on all pushes — if buf isn't installed properly it fails silently
4. **No Go build check**: Compile errors only caught by `go test`, which is slower and less clear
5. **No caching**: Every run downloads all Go modules and npm packages from scratch
6. **Contract tests fragile**: Same envtest setup issue

### Fixed Pipeline

```
push/PR to main
    │
    ├── build (parallel group 1)
    │   ├── Go build (compile check)
    │   ├── Go lint (golangci-lint)
    │   └── TypeScript check (web + shared + extension)
    │
    └── test (parallel group 2, after build)
        ├── Go unit tests (with envtest for controller)
        └── Contract tests (with envtest)
```

### Key Fixes

1. **envtest**: Use the official `setup-envtest` action pattern — run `setup-envtest use` and capture the path via command output, then export as env var in the same step
2. **npm caching**: Use `actions/setup-node` with `cache: 'npm'` and explicit cache paths
3. **Go caching**: Use `actions/setup-go` which caches Go modules by default
4. **Web TypeScript**: Add explicit `cd web && npm ci && npx tsc --noEmit`
5. **buf**: Make buf optional — skip if not installed (it's already correctly handled)
6. **Simplify stages**: Two parallel groups (build+lint, then tests) instead of serial chain
