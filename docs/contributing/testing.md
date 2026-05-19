# Testing

Test runners are Task targets. Git hooks live in lefthook.

## Hooks

```bash
task hooks:install
```

### pre-commit (parallel)

| Hook | Scope | |
|------|-------|---|
| `go-fmt` | `*.go` | Formats + re-stages |
| `golangci-lint` | `*.go` | Lints |
| `buf-lint` | `*.proto` | |
| `tsc-web` | `*.{ts,tsx}` | `web/` |
| `tsc-shared` | `*.{ts,tsx}` | `packages/shared/` |
| `tsc-extension` | `*.{ts,tsx}` | `packages/pi-aot-extension/` |

### commit-msg

`commitlint` — conventional commits.

### pre-push (parallel)

| Hook | |
|------|---|
| `go-test` | Go unit + integration |
| `buf-breaking` | Proto breaking-change check vs `main` |

## Suites

| Command | What |
|---------|------|
| `task test` | Go + web + extension, parallel |
| `task test:unit` | Go unit only (`-short`); fast, no Docker |
| `task test:go` | Go unit + integration |
| `task test:contract` | ConnectRPC + protovalidate |
| `task test:temporal` | Workflow tests |
| `task test:layer2` | Pipeline integration (LLM stubbed, no cluster) |
| `task test:regression` | Regression suite — gates releases and PRs to main |
| `task test:integration` | Docker (testcontainers) |
| `task test:extension` | pi-aot-extension TS |
| `task test:shared` | `@aot/shared` TS |
| `task test:web` | Playwright |
| `task test:all` | Sequential: proto lint → unit → contract → temporal → integration → e2e |

Single Go test: `go test ./internal/server/... -run TestCreateAgentRun -count=1`.

## E2E

Against a live cluster:

| Command | |
|---------|---|
| `task test:e2e` | Go E2E (30m timeout) |
| `task test:e2e:api` | API-focused |
| `task test:e2e:infra` | Build + import + LLM E2E |
| `task test:e2e:playwright` | Browser only |
| `task test:e2e:full` | Setup Soft-Serve → Go + Playwright → teardown |

Full E2E uses [Soft-Serve](https://github.com/charmbracelet/soft-serve) for fixture repos.

## Lint / proto

```bash
task lint            # golangci-lint + tsc --noEmit (web, shared, extension)
task proto:lint
task proto:breaking
```
