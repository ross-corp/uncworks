# Testing

Every test runner is a Task target. Lefthook holds the git hooks.

## Hooks

```bash
task hooks:install
```

### pre-commit, run in parallel

| Hook | Scope | What it does |
|------|-------|---|
| `go-fmt` | `*.go` | Formats the file and re-stages it |
| `golangci-lint` | `*.go` | Lints the new changes only |
| `buf-lint` | `*.proto` | Lints the proto files |
| `tsc-web` | `*.{ts,tsx}` | Type-checks `web/` |
| `tsc-shared` | `*.{ts,tsx}` | Type-checks `packages/shared/` |
| `tsc-extension` | `*.{ts,tsx}` | Type-checks `packages/pi-aot-extension/` |

### commit-msg

`commitlint` enforces conventional commits.

### pre-push, run in parallel

| Hook | What it does |
|------|---|
| `go-test` | Runs the Go unit and integration tests |
| `buf-breaking` | Checks the proto files for breaking changes against `main` |

## Suites

| Command | What |
|---------|------|
| `task test` | Runs the Go, web, and extension tests in parallel |
| `task test:unit` | Go unit tests only, with `-short`. Fast, and needs no Docker |
| `task test:go` | Go unit and integration tests |
| `task test:contract` | ConnectRPC and protovalidate contract tests |
| `task test:temporal` | Workflow tests |
| `task test:layer2` | Pipeline integration with the LLM stubbed, and no cluster |
| `task test:regression` | Regression suite. It gates releases and pull requests to main |
| `task test:integration` | Integration tests through testcontainers, which needs Docker |
| `task test:extension` | pi-aot-extension TypeScript tests |
| `task test:shared` | `@aot/shared` TypeScript tests |
| `task test:web` | Playwright browser tests |
| `task test:all` | Runs proto lint, unit, contract, temporal, integration, and e2e in that order |

Run one Go test with `go test ./internal/server/... -run TestCreateAgentRun -count=1`.

Run controller tests with `-p 1`. Parallel test binaries race to extract the same
envtest etcd binary, and the loser fails with `text file busy`.

## E2E

These run against a live cluster.

| Command | What it runs |
|---------|---|
| `task test:e2e` | The Go E2E suite, with a 30 minute timeout |
| `task test:e2e:api` | The API-focused subset |
| `task test:e2e:infra` | Build, import, and the LLM E2E subset |
| `task test:e2e:playwright` | The browser tests only |
| `task test:e2e:full` | Sets up Soft-Serve, runs the Go and Playwright suites, then tears down |

The full E2E suite serves its fixture repos from
[Soft-Serve](https://github.com/charmbracelet/soft-serve).

## Lint and proto

```bash
task lint            # golangci-lint, then tsc --noEmit for web, shared, and extension
task proto:lint
task proto:breaking
```
