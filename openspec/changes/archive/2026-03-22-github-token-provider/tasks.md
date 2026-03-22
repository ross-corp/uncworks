## 1. Token Provider Package

- [x] 1.1 Create `internal/github/provider.go` with `TokenProvider` interface, `PATProvider` implementation
- [x] 1.2 Create `internal/github/app.go` with `AppProvider` stub (returns "not implemented" error)
- [x] 1.3 Create `internal/github/provider_test.go` — test PATProvider returns token, returns error when empty, AppProvider returns not-implemented error

## 2. Replace os.Getenv Calls

- [x] 2.1 Add `GitHubProvider github.TokenProvider` field to `Activities` struct in `internal/temporal/activities.go`
- [x] 2.2 Replace `os.Getenv("GITHUB_TOKEN")` in `internal/temporal/activities_git.go` (CreatePR) with `a.GitHubProvider.Token(ctx)`
- [x] 2.3 Replace `os.Getenv("GITHUB_TOKEN")` in `internal/server/github.go` — inject provider via constructor
- [x] 2.4 Replace `os.Getenv("GITHUB_TOKEN")` in `internal/server/webhook.go` — inject provider via constructor
- [x] 2.5 Wire provider creation in `cmd/apiserver/main.go` and `cmd/temporal-worker/main.go`

## 3. Worker-Side Git Push

- [x] 3.1 Update `PushChanges` in `activities_git.go` to inject token into remote URL via sidecar for authenticated push
- [x] 3.2 Add `InjectTokenInURL` helper in `internal/github/provider.go`
- [x] 3.3 Verified git not in distroless worker container — current approach uses sidecar (trusted) for git operations
- [x] 3.4 Add tests for InjectTokenInURL in provider_test.go and hydrator_test.go

## 4. Init Container Token Scoping

- [x] 4.1 In `BuildAgentPod`: add GITHUB_TOKEN env var to init container ONLY (from Secret ref), not to agent/sidecar
- [x] 4.2 In `hydrator.go`: use GITHUB_TOKEN to inject auth into clone URL for private repos
- [x] 4.3 Add contract test: verify BuildAgentPod init container has GITHUB_TOKEN but agent/sidecar containers do not

## 5. Helm Configuration

- [x] 5.1 Add `github.tokenSecretName` to `deploy/helm/aot/values.yaml`
- [x] 5.2 Add Secret env var reference to worker and apiserver templates (conditional on github.tokenSecretName)
- [x] 5.3 Add GITHUB_TOKEN_SECRET_NAME to controller template for workflow input propagation
- [x] 5.4 Add `github.tokenSecretName` to `dev-values.yaml`

## 6. Verification

- [x] 6.1 Grep for `os.Getenv("GITHUB_TOKEN")` — zero results in internal/temporal/ and internal/server/
- [x] 6.2 Run all tests — 14/14 packages pass
- [ ] 6.3 Deploy and verify: create a run, check init container env, verify agent container has no GITHUB_TOKEN
