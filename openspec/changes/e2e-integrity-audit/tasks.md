## 1. Fix Temporal Workflow (Critical — nothing works without this)

- [x] 1.1 Fix nil `*Activities` pointer in `AgentRunWorkflow` — use proper Temporal activity invocation pattern (function references or string names)
- [x] 1.2 Fix defer cleanup block to use correct activity invocation (currently calls methods on nil pointer)
- [x] 1.3 Move signal channel registration (`workflow.GetSignalChannel`) to workflow start, before provisioning/pod creation
- [x] 1.4 Add `activity.RecordHeartbeat()` in `WaitForHydration` and `GetAgentStatus` polling loops
- [x] 1.5 Fix `SpawnJuniorWorkflow` to pass `ModelTier`, `MaxBudget`, and `LiteLLMBaseURL` to child workflow input
- [x] 1.6 Reuse `http.Client` in Activities struct for sidecar RPC instead of creating per-call
- [x] 1.7 Add sidecar readiness check (retry loop) in `StartAgent` before calling sidecar RPC
- [x] 1.8 Handle errors from `StopAgent` and `ForwardHumanInput` activities instead of discarding with `_`
- [x] 1.9 Remove unused activity function pointer variables (`CreateAgentPodActivity`, etc.)
- [x] 1.10 Verify workflow tests pass with fixed activity invocation pattern

## 2. Fix Sidecar

- [x] 2.1 Add stderr streaming — second goroutine in `monitorProcess` that reads stderr pipe and emits `OUTPUT_TYPE_STDERR` events
- [x] 2.2 Log dropped messages when output channel buffer is full instead of silent discard
- [x] 2.3 Explicitly close stdin/stdout/stderr pipes on process exit
- [x] 2.4 Add timeout to `cmd.Wait()` — force-kill process after configurable duration
- [x] 2.5 Add tests for process lifecycle (start, stdout/stderr streaming, termination)

## 3. Wire API Server to K8s

- [x] 3.1 Add K8s client (`client.Client` from controller-runtime) to `AOTServiceHandler` struct
- [x] 3.2 Rewrite `CreateAgentRun` to create an AgentRun CRD in K8s (generate name `ar-<random>`)
- [x] 3.3 Rewrite `GetAgentRun` to read AgentRun CRD from K8s, enrich with Temporal state
- [x] 3.4 Rewrite `ListAgentRuns` to list AgentRun CRDs from K8s, sorted by creation time
- [x] 3.5 Rewrite `CancelAgentRun` to signal Temporal workflow cancellation (remove in-memory state flip)
- [x] 3.6 Fix `SendHumanInput` to return error if Temporal signal fails (don't swallow)
- [x] 3.7 Fix `WatchAgentRun` to return error (not nil) when EventBus is not configured
- [x] 3.8 Complete `mapWorkflowStateToProto` to map all fields (TraceId, StartedAt, CompletedAt)
- [x] 3.9 Remove in-memory `map[string]*AgentRun` and `sync.RWMutex`
- [x] 3.10 Update `cmd/apiserver/main.go` to initialize K8s client and pass to handler
- [x] 3.11 Update API server tests for K8s-backed handler (use envtest)

## 4. Proto and Type Alignment

- [x] 4.1 Add `string image = 9` to proto `AgentRunSpec` message
- [x] 4.2 Add `double max_budget = 10` to proto `AgentRunSpec` message
- [x] 4.3 Add `string worktree_path = 6` to proto `AgentRunStatus` message
- [x] 4.4 Regenerate Go proto types (`buf generate --template buf.gen.go.yaml`)
- [x] 4.5 Regenerate TypeScript proto types (`buf generate --template buf.gen.ts.yaml --include-imports`)
- [x] 4.6 Update `mapWorkflowStateToProto` and CRD-to-proto conversion to include new fields

## 5. Helm Chart RBAC Update

- [x] 5.1 Add `serviceAccountName` to apiserver Deployment template (reference shared ServiceAccount)
- [x] 5.2 Verify `helm template` renders apiserver with correct ServiceAccount
- [x] 5.3 Rebuild and reimport controlplane image with API server K8s client changes

## 6. Web Dashboard Fixes

- [x] 6.1 Preserve previous run list on API error instead of returning empty array
- [x] 6.2 Add loading state indicator while fetching
- [x] 6.3 Add error state indicator with retry button

## 7. Controller Hardening

- [x] 7.1 Fix workflow start race — write annotation before status update (or use single update with both)
- [x] 7.2 Update status with error message when Temporal is unreachable (instead of silent requeue)
- [x] 7.3 Use Temporal `CloseTime` for `CompletedAt` instead of reconciliation timestamp
- [x] 7.4 Log when EventBus is nil at startup (warn once, not per-event)

## 8. E2E Tests via API

- [x] 8.1 Create `e2e/api_test.go` with ConnectRPC client setup targeting in-cluster API server
- [x] 8.2 Test: `CreateAgentRun` via API → verify CRD exists in K8s
- [x] 8.3 Test: Full lifecycle via API → Pending → Running → Succeeded
- [x] 8.4 Test: `CancelAgentRun` via API → verify Cancelled phase
- [x] 8.5 Test: `SendHumanInput` via API → verify run completes after input
- [x] 8.6 Add `test:e2e:api` Taskfile target

## 9. Verification

- [x] 9.1 Run `task test:go` — all unit tests pass
- [x] 9.2 Run `task test:contract` — contract tests pass with K8s-backed server
- [x] 9.3 Run `task test:temporal` — workflow tests pass with fixed activity pattern
- [x] 9.4 Run `helm template` — chart renders correctly with apiserver ServiceAccount
- [x] 9.5 Deploy to dev cluster and create runs via web dashboard — verify they execute
