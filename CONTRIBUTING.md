# Contributing to UNCWORKS

## Setup

```bash
# 1. Install devbox (https://www.jetify.com/devbox)
curl -fsSL https://get.jetify.com/devbox | bash

# 2. Enter the dev environment. This installs Go, Node, kubectl, helm, and buf
devbox shell

# 3. Install the project dependencies and the git hooks
task install

# 4. Build every binary
task build
```

The macOS desktop app in `cmd/uncworks-app/` needs Wails. Install it with
`go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

Run `git commit` from inside `devbox shell`. The hooks call `gofmt`,
`golangci-lint`, and `npx`, and none of them are on the bare PATH.

## Local cluster

UNCWORKS runs on a local k3s cluster that Colima manages.

```bash
task cluster:setup    # create the Colima VM and deploy everything. Run this once
task cluster:status   # report pod health
task dev:deploy       # rebuild the images and roll out every deployment
task dev:web          # Vite dev server for the web dashboard only
```

## Daily cycle

```bash
# After you edit Go code:
task dev:deploy       # build images into the k8s.io namespace and roll out

# After you edit web code only:
task dev:web

# After you edit a proto file:
task proto:gen        # regenerate the Go and TypeScript bindings

# Before you push:
task test:go          # Go unit, contract, and layer 2 tests
task test:web         # Playwright tests
task test:e2e         # end-to-end tests. Needs a running cluster
```

## Testing

| Command | What it tests |
|---------|--------------|
| `task test:go` | Go unit tests across every package |
| `task test:contract` | API contract tests |
| `task test:layer2` | Pipeline integration tests |
| `task test:regression` | Regression suite, tagged `//go:build regression` |
| `task test:web` | Playwright end-to-end tests for the dashboard |
| `task test:extension` | pi-aot-extension TypeScript tests |
| `task test:shared` | `@aot/shared` TypeScript tests |
| `task test:e2e` | End-to-end tests. Needs a cluster |

Run `task --list` to see every task.

Run controller tests with `-p 1`. Parallel test binaries race to extract the same
envtest etcd binary, and the loser fails with `text file busy`.

## Code style

Go:

- Run `golangci-lint run`. The config is `.golangci.yml`.
- Wrap every error you return from another package: `fmt.Errorf("doing X: %w", err)`.
- Use `slog` for structured logging. Do not use `fmt.Println` or `log.Printf`.
- Give every exported symbol a godoc comment.

TypeScript and React:

- Do not use `any` without an `// eslint-disable` comment that says why.
- Return `{ data, loading, error }` from a hook. Never swallow an error.
- Use the Tailwind CSS variables, such as `text-foreground` and `bg-background`.
  Do not hardcode a color.

Proto:

- Use snake_case fields, PascalCase messages, and UPPER_SNAKE enum values.
- Give every enum a zero value named `FOO_UNSPECIFIED = 0`.
- Run `task proto:lint` before you push a proto change.

Documentation:

- Write in Simplified Technical English. Use one instruction per sentence, active
  voice, and one term for one concept.
- Do not use em-dashes, emoji, callout blocks, or a bolded label at the start of
  a sentence or list item.
- Use Mermaid for every diagram.

## Commit style

Use conventional commits. A body is optional.

```
feat: add webhook retry backoff
fix: handle nil project ref in list handler
chore: bump temporal SDK to v1.47
refactor: extract rate limit middleware
test: add layer2 HITL flow tests
```

Do not create merge commits. Rebase onto `main` before you open a pull request.

## Pull requests

- Keep each pull request to one logical change.
- Every CI check MUST pass.
- Add tests for new behavior. Do not reduce coverage.
- Update `docs/` when you change user-facing behavior.
- Name the related OpenSpec change in the description.

## OpenSpec workflow

Significant changes go through [OpenSpec](openspec/).

```bash
openspec new change <name>   # scaffold proposal, design, and tasks
openspec status <name>       # show task completion
openspec validate <name>     # validate the artifacts
openspec archive <name>      # archive when every task is done
```

Active changes live in `openspec/changes/`. Completed changes live in
`openspec/changes/archive/`.

## Repository layout

```
cmd/
  apiserver/          ConnectRPC API server (gRPC and HTTP)
  controller/         Kubernetes controller, reconciles AgentRun
  temporal-worker/    Temporal workflow worker
  hydration/          Init container, clones the repo
  sidecar/            RPC gateway inside the agent pod
  uncworks/           End-user CLI
  aot/                Internal workspace CLI
  bff/                Backend-for-frontend for the desktop app
  uncworks-app/       macOS desktop app (Wails). Build output is gitignored

internal/
  server/             HTTP and gRPC handlers
  temporal/           Workflows and activities
  controller/         Reconciliation logic
  softserve/          Soft-Serve git client
  brain/              PostgreSQL state store and semantic search
  embeddings/         Embedding generation through Ollama
  litellm/            LiteLLM admin API client
  github/             GitHub App client, webhooks, pull requests

web/src/
  views/              Page-level React components
  components/         Reusable UI components
  hooks/              React hooks for data fetching and state
  lib/                Utilities

proto/                Protobuf service definitions
gen/                  Generated Go and TypeScript code. Do not edit
deploy/helm/aot/      Helm chart
docker/               Dockerfiles
ci/                   Dagger CI pipeline, written in Go
test/                 Integration, contract, and regression tests
```

## Getting help

- Open a [GitHub issue](https://github.com/ross-corp/uncworks/issues) for a bug
  or a feature request.
- Read `docs/` for guides on specific topics.
