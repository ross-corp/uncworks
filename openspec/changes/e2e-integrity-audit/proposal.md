## Why

The API server, K8s controller, and Temporal workflow engine were built and tested in isolation. The API server stores runs in an in-memory map and never creates K8s CRDs. The controller only watches CRDs and has no connection to the API. The Temporal workflow code has a nil pointer bug that prevents any activity from executing. The web dashboard calls the API but only ever sees in-memory state. No test creates a run via the API and verifies it executes end-to-end. The system does not work as an integrated product.

## What Changes

- **BREAKING**: API server `CreateAgentRun` creates a K8s CRD instead of storing in memory. `ListAgentRuns`/`GetAgentRun` read from K8s + Temporal. The in-memory map is removed.
- Fix Temporal workflow nil pointer and activity invocation pattern so workflows can actually execute activities.
- Fix sidecar to stream stderr, handle readiness, and not silently drop output.
- Add activity heartbeats in long-running polling loops (WaitForHydration, GetAgentStatus).
- Register signal channels at workflow start instead of after agent startup.
- Fix SpawnJunior to pass LLM config to child workflows.
- Reuse HTTP client for sidecar RPC instead of creating one per call.
- Add proto fields for `image`, `worktree_path`, and `max_budget` to align proto with CRD.
- Regenerate TypeScript proto types (`buf generate`).
- Add E2E tests that exercise the full API → K8s → Temporal → Pod path.
- Fix web dashboard to preserve state on API errors and show loading/error states properly.

## Capabilities

### New Capabilities
- `api-k8s-bridge`: API server creates and reads K8s CRDs, bridging the gRPC API to the controller reconciliation loop.
- `e2e-api-tests`: End-to-end tests that create runs via the gRPC API and verify CRD creation, workflow execution, and event streaming.

### Modified Capabilities
- `helm-chart`: API server deployment needs RBAC to create/list/watch AgentRun CRDs (currently only controller has this).

## Impact

- `internal/server/grpc.go` — rewrite to use K8s client instead of in-memory map
- `internal/server/grpc_test.go` — rewrite tests for K8s-backed API
- `internal/temporal/workflow.go` — fix nil pointer, activity invocation, signal registration, cleanup defer
- `internal/temporal/activities.go` — add heartbeats, fix HTTP client reuse, add sidecar readiness check
- `internal/sidecar/gateway.go` — add stderr streaming, fix output drops
- `proto/aot/api/v1/api.proto` — add missing fields
- `gen/` — regenerate from proto
- `web/src/App.tsx` — fix error handling, stale state
- `deploy/helm/aot/templates/rbac.yaml` — add CRD permissions for apiserver ServiceAccount (or share with controller)
- `e2e/` — new API-driven E2E tests
- `test/contract/` — update for K8s-backed server
