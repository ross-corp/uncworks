# Testing

UNCWORKS uses lefthook for git hooks and Task for running test suites.

## Git Hooks

Install hooks via:

```
task hooks:install
```

### Pre-Commit (parallel)

| Hook | Scope | Description |
|------|-------|-------------|
| `go-fmt` | `*.go` | Formats Go files and re-stages |
| `golangci-lint` | `*.go` | Lints new Go changes |
| `buf-lint` | `*.proto` | Lints protobuf files |
| `tsc-web` | `*.{ts,tsx}` | TypeScript check on `web/` |
| `tsc-shared` | `*.{ts,tsx}` | TypeScript check on `packages/shared/` |
| `tsc-extension` | `*.{ts,tsx}` | TypeScript check on `packages/pi-aot-extension/` |

### Commit Message

Uses `commitlint` to enforce conventional commit format.

### Pre-Push (parallel)

| Hook | Description |
|------|-------------|
| `go-test` | Runs Go unit + integration tests |
| `buf-breaking` | Checks for breaking proto changes against `main` |

## Test Suites

### Quick Tests

```
task test          # Run Go, web, and extension tests in parallel
task test:unit     # Go unit tests only (fast, no Docker)
task test:go       # Go tests (unit + integration)
```

### Full Pipeline

```
task test:all      # Sequential: proto lint -> unit -> contract -> temporal -> integration -> e2e
```

### By Category

| Command | Description | Requirements |
|---------|-------------|--------------|
| `task test:unit` | Go unit tests (`-short` flag) | `kubebuilder` envtest assets |
| `task test:contract` | ConnectRPC + protovalidate tests | -- |
| `task test:temporal` | Temporal workflow tests | temporal-workflow-engine |
| `task test:integration` | Integration tests | Docker (testcontainers) |
| `task test:extension` | pi-aot-extension TypeScript tests | npm |
| `task test:shared` | @aot/shared TypeScript tests | npm |
| `task test:web` | Playwright E2E for web dashboard | npm, Playwright browsers |

### E2E Tests

E2E tests run against a live cluster:

```
task test:e2e           # Go E2E tests against k0s cluster (30min timeout)
task test:e2e:api       # API-focused E2E tests
task test:e2e:infra     # Build images, import, run LLM E2E tests
task test:e2e:full      # Full suite: setup Soft-Serve, run Go + Playwright, teardown
task test:e2e:playwright # Playwright browser tests only
```

The full E2E suite (`test:e2e:full`) uses [Soft-Serve](https://github.com/charmbracelet/soft-serve) as a local git server with fixture repositories.

## Linting

```
task lint
```

Runs `golangci-lint` on Go code and `tsc --noEmit` on all TypeScript packages (web, shared, pi-aot-extension).

## Proto Checks

```
task proto:lint      # Lint proto files with buf
task proto:breaking  # Check for breaking changes against main branch
```
