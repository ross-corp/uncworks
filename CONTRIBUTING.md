# Contributing to UNCWORKS

## Setup

```bash
# 1. Install devbox (https://www.jetify.com/devbox)
curl -fsSL https://get.jetify.com/devbox | bash

# 2. Enter the dev environment (installs Go, Node, kubectl, helm, buf, etc.)
devbox shell

# 3. Install project dependencies and git hooks
task install

# 4. Build all binaries
task build
```

> [!NOTE]
> The macOS desktop app (`cmd/uncworks-app/`) requires Wails: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## Local Cluster

UNCWORKS runs on a local k3s cluster managed by Colima:

```bash
task cluster:setup    # Create colima VM + deploy everything (one-time)
task cluster:status   # Check pod health
task dev:deploy       # Rebuild images + rollout all deployments
task dev:web          # Vite dev server (web dashboard only)
```

## Daily Development Cycle

```bash
# After editing Go code:
task dev:deploy       # build images into k8s.io namespace + rollout

# After editing web code only:
cd web && npm run dev  # or: task dev:web

# After editing proto files:
task proto:gen        # regenerate Go + TypeScript bindings

# Run tests before pushing:
task test:go          # Go unit + contract + layer2
task test:web         # Vitest
task test:e2e         # End-to-end (requires running cluster)
```

## Testing

| Command | What it tests |
|---------|--------------|
| `task test:go` | Go unit tests across all packages |
| `task test:contract` | API contract tests |
| `task test:layer2` | Pipeline integration tests |
| `task test:regression` | Regression suite (tagged `//go:build regression`) |
| `task test:web` | React component + hook unit tests |
| `task test:e2e` | End-to-end tests (needs cluster) |

Run `task --list` to see all available tasks.

## Code Style

**Go:**
- `golangci-lint run` (config in `.golangci.yml`)
- Wrap errors: `fmt.Errorf("doing X: %w", err)`
- Use `slog` for structured logging — no `fmt.Println` or `log.Printf`
- All exported symbols need godoc comments

**TypeScript/React:**
- No `any` types without an explicit `// eslint-disable` comment explaining why
- Hooks expose `{ data, loading, error }` — never swallow errors silently
- Use Tailwind CSS variables (`text-foreground`, `bg-background`) — no hardcoded colors

**Proto:**
- snake_case fields, PascalCase messages, UPPER_SNAKE enums
- Zero-value enum entry required: `FOO_UNSPECIFIED = 0`
- Run `task proto:lint` before pushing proto changes

## Commit Style

Conventional commits, no body required:

```
feat: add webhook retry backoff
fix: handle nil project ref in list handler
chore: bump temporal SDK to v1.31
refactor: extract rate limit middleware
test: add layer2 HITL flow tests
```

No merge commits. Rebase onto main before opening a PR.

## Pull Requests

- Keep PRs focused — one logical change per PR
- All CI checks must pass
- Add tests for new behavior; don't reduce coverage
- Update `docs/` if you change user-facing behavior
- Reference any related OpenSpec change in the PR description

## OpenSpec Workflow

Significant changes use [OpenSpec](openspec/) for structured proposals:

```bash
/opsx:propose <change-name>   # create proposal + design + tasks
/opsx:apply                   # implement tasks
/opsx:archive                 # archive when done
```

Active changes are in `openspec/changes/`. Archived changes are in `openspec/changes/archive/`.

## Architecture Overview

```
cmd/
  apiserver/          ConnectRPC API server (gRPC + HTTP)
  controller/         Kubernetes controller (AgentRun CRD reconciler)
  worker/             Temporal workflow worker
  uncworks/           CLI tool
  uncworks-app/       macOS desktop app (Wails — gitignored build output)

internal/
  server/             HTTP/gRPC handlers
  temporal/           Workflows + activities
  controller/         Reconciliation logic
  softserve/          Soft-serve git client
  brain/              LLM inference client
  ratelimit/          Per-IP rate limiting

web/src/
  views/              Page-level React components
  components/         Reusable UI components
  hooks/              React hooks (data fetching, state)
  lib/                Utilities

proto/                Protobuf service definitions
gen/                  Generated code (Go + TypeScript) — do not edit

deploy/helm/aot/      Helm chart for Kubernetes deployment
docker/               Dockerfiles
ci/                   Dagger CI pipeline (Go)
test/                 Integration + contract + regression tests
```

## Getting Help

- Open a [GitHub Issue](https://github.com/ross-corp/uncworks/issues) for bugs or feature requests
- Check `docs/` for guides on specific topics
