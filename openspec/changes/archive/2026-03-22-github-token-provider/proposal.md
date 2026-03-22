## Why

GitHub auth is hardcoded as `os.Getenv("GITHUB_TOKEN")` in 4 files with no abstraction, no secret management, and a critical security gap: `git push` runs inside the agent pod via `execInSidecar` but the pod has no credentials — push would fail. Meanwhile, `git clone` only works for public repos. The token isn't wired into Helm at all.

Worse, naively injecting `GITHUB_TOKEN` into the agent pod would expose it to the agent process — an LLM-driven coding agent should NOT have access to credentials it doesn't need.

## What Changes

- Extract a `GitHubTokenProvider` interface replacing all `os.Getenv("GITHUB_TOKEN")` calls
- Implement `PATProvider` (static token from k8s Secret) for current single-user flow
- Implement `AppProvider` stub for future GitHub App integration
- **Move git push from agent pod to worker** — the worker does commit+push using PVC host path, token never enters the agent pod
- **Pass token to init container only** for private repo cloning — agent/sidecar containers never see it
- Wire GitHub credentials into Helm chart as k8s Secret references
- Add contract tests for the provider interface and the security boundary

## Capabilities

### New Capabilities
- `github-auth-provider`: Pluggable token provider interface with PAT and App implementations. Token is scoped — init container gets it for clone, worker gets it for push/PR, agent pod never sees it.

### Modified Capabilities
- None

## Impact

- **New package**: `internal/github/` — provider interface, PAT impl, App stub
- **Modified**: `internal/temporal/activities_git.go` — PushChanges uses worker-side git (PVC host path) instead of execInSidecar
- **Modified**: `internal/temporal/activities.go` — BuildAgentPod passes token to init container only
- **Modified**: `internal/server/github.go`, `webhook.go` — inject provider
- **Modified**: `cmd/apiserver/main.go`, `cmd/temporal-worker/main.go` — create provider
- **Helm**: Add `github.tokenSecretName` to values, reference in worker + apiserver deployments
- **Security**: Agent pods never have GitHub credentials in their environment
