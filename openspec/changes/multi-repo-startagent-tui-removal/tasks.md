## 1. Proto Schema Changes

- [x] 1.1 Add `Repository` message to `proto/aot/api/v1/api.proto` with `url`, `branch`, `path` fields
- [x] 1.2 Replace `repo_url` (field 2) and `branch` (field 3) in `AgentRunSpec` with `repeated Repository repos = 2`; renumber remaining fields
- [x] 1.3 Add `buf.validate` rule: `repos` must have at least one entry
- [x] 1.4 Run `buf generate` for Go and TS codegen; verify `buf lint` passes

## 2. CRD Type Changes

- [x] 2.1 Add `Repository` struct to `api/v1alpha1/agentrun_types.go` with `URL`, `Branch`, `Path` fields
- [x] 2.2 Replace `RepoURL`/`Branch` fields in `AgentRunSpec` with `Repos []Repository`
- [x] 2.3 Update CRD validation markers if any exist
- [x] 2.4 Update controller code that reads `agentRun.Spec.RepoURL` / `.Branch` to use `.Repos`

## 3. API Server Updates

- [x] 3.1 Update `specProtoToCRD` in `internal/server/grpc.go` to map `repeated Repository` → CRD `[]Repository`
- [x] 3.2 Update `crdToProto` to map CRD repos back to proto
- [x] 3.3 Update unit tests in `internal/server/grpc_test.go`
- [x] 3.4 Update contract tests in `test/contract/server_aot_test.go`

## 4. Hydration Multi-Repo Support

- [x] 4.1 Change `Config` struct in `internal/hydration/hydrator.go`: replace `RepoURL`/`Branch` with `Repos []RepoConfig`
- [x] 4.2 Update `ConfigFromEnv` to parse `AOT_REPOS` JSON env var (list of `{url, branch, path}`)
- [x] 4.3 Update `cloneRepo` to loop over all repos, cloning each into `/workspace/.bare/<name>/`
- [x] 4.4 Update `createWorktree` to loop over repos, creating worktrees at `/workspace/src/<path>/`
- [x] 4.5 Derive repo name from URL when `path` is empty (e.g., `https://github.com/org/foo.git` → `foo`)
- [x] 4.6 Add `PrimaryWorktreePath()` method returning the first repo's worktree path
- [x] 4.7 Update hydration tests for multi-repo scenarios
- [x] 4.8 Update `cmd/hydration/main.go` to use new config

## 5. Workflow and Activity Updates

- [x] 5.1 Add `Repos []Repository` to `WorkflowInput` (replace `RepoURL`/`Branch`)
- [x] 5.2 Add `Repos []Repository` to `CreateAgentPodInput` (replace `RepoURL`/`Branch`)
- [x] 5.3 Update `BuildAgentPod` to encode repos as `AOT_REPOS` JSON env var instead of `AOT_REPO_URL`/`AOT_BRANCH`
- [x] 5.4 Add `WorkspacePath string` to `WaitForHydrationOutput`
- [x] 5.5 Add `RepoPath string` to `StartAgentInput`; pass it through workflow after hydration
- [x] 5.6 Update workflow to pass `hydrationOutput.WorkspacePath` to `StartAgentInput.RepoPath`
- [x] 5.7 Update workflow tests in `test/temporal/workflow_test.go`
- [x] 5.8 Update integration test mocks in `test/temporal/integration_test.go`

## 6. Sidecar StartAgent Fix

- [x] 6.1 Update `startAgentProcess` in `internal/sidecar/gateway.go`: use `req.RepoPath` for `cmd.Dir`, default to `/workspace/src` if empty
- [x] 6.2 Update sidecar tests in `internal/sidecar/gateway_test.go` if they exist
- [x] 6.3 Rebuild sidecar Docker image and import into k0s

## 7. Controller Updates

- [x] 7.1 Update `startWorkflow` in `internal/controller/agentrun_controller.go` to pass `Repos` from CRD to workflow input
- [x] 7.2 Update controller tests

## 8. TUI Removal

- [x] 8.1 Delete `packages/tui/` directory
- [x] 8.2 Remove `test:tui` and `dev:tui` tasks from `Taskfile.yml`
- [x] 8.3 Remove TUI from `test` aggregate task deps in `Taskfile.yml`
- [x] 8.4 Remove TUI npm install from deps task in `Taskfile.yml`
- [x] 8.5 Remove `dashboard` subcommand from `cmd/aot/main.go`
- [x] 8.6 Remove `test:tui` from `devbox.json` scripts
- [x] 8.7 Update proto comment: "Web UI, TUI, CLI" → "Web UI, CLI" in `api.proto`; regenerate
- [x] 8.8 Update `README.md`: remove TUI references, mermaid diagram, task table entries
- [x] 8.9 Update `AGENTS.md`: remove TUI test/reference lines
- [x] 8.10 Update `docs/user-guide.md`: remove TUI section and references

## 9. Verification

- [x] 9.1 Run `buf lint` and `buf generate` — proto valid and codegen up to date
- [x] 9.2 Run `go test ./api/... ./internal/...` — all unit tests pass
- [x] 9.3 Run `go test ./test/contract/...` — all contract tests pass
- [x] 9.4 Run `go test ./test/temporal/...` — all workflow tests pass
- [x] 9.5 Run `npx tsc --noEmit` in web/ and packages/shared/ — TS type checks pass
- [ ] 9.6 Rebuild Docker images, import into k0s, create agent run — verify full lifecycle
- [ ] 9.7 Commit and push; verify CI passes
