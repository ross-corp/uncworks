## Context

The AOT system has three independent subsystems — gRPC API server, K8s controller, Temporal workflow engine — that were developed and tested in isolation. The API server uses an in-memory map for state, the controller watches K8s CRDs, and the Temporal workflow has a nil-pointer bug preventing activity execution. No integration path connects them. The web dashboard only sees in-memory API state, so CRD-created runs are invisible and API-created runs never execute.

## Goals / Non-Goals

**Goals:**
- API server creates K8s CRDs so runs flow through the real reconciliation loop
- Fix all Temporal workflow bugs so activities execute correctly
- Fix sidecar output handling (stderr, readiness, drops)
- Proto schema matches CRD schema for all user-facing fields
- E2E tests cover the full API → K8s → Temporal → Pod path
- Web dashboard shows accurate, real-time state

**Non-Goals:**
- Adding ExternalBackend or KubeVirt support to the proto (tracked separately)
- Implementing cursor-based pagination in ListAgentRuns (functional but incomplete)
- Adding a validating admission webhook for CRDs
- Migrating to a persistent database for the API server

## Decisions

### 1. API server reads/writes K8s CRDs directly

The API server will use a K8s client to create AgentRun CRDs on `CreateAgentRun` and list/get them for queries. The in-memory `map[string]*AgentRun` is removed entirely.

**Alternative considered:** Keep in-memory map and add a sync loop to mirror CRDs. Rejected — adds complexity, eventual consistency bugs, and duplicates state that K8s already manages.

**ID scheme:** API server generates a CRD name like `ar-<random-suffix>` (e.g., `ar-7kx2m`) and creates the CRD with that name. The controller picks it up as usual. `GetAgentRun` looks up by CRD name. This unifies the ID namespace.

### 2. API server enriches status from Temporal (keep existing pattern)

`GetAgentRun` reads the CRD from K8s for base status, then optionally queries Temporal for live workflow state (current pattern in `mapWorkflowStateToProto`). The mapping function will be completed to include all fields (TraceId, StartedAt, CompletedAt).

### 3. Fix Temporal workflow by using proper activity function references

Replace `var a *Activities` nil pointer with the correct Temporal SDK pattern: register activity methods on the worker, invoke by function reference. The workflow function should NOT hold an Activities instance — it invokes activities by reference and the SDK routes to the registered worker.

```go
// Before (broken):
var a *Activities  // nil!
workflow.ExecuteActivity(ctx, a.CreateAgentPod, input)

// After (correct):
workflow.ExecuteActivity(ctx, (*Activities).CreateAgentPod, input)
// Or use the activity struct registered on worker:
workflow.ExecuteActivity(ctx, "CreateAgentPod", input)
```

### 4. Share ServiceAccount between controller and API server (or add separate RBAC)

The API server needs permissions to create/list/get/watch AgentRun CRDs. Two options:
- **Option A:** Both controller and apiserver use the same ServiceAccount with full CRD RBAC.
- **Option B:** Separate ServiceAccount for apiserver with only create/list/get/watch (no delete, no pods).

Going with **Option A** (shared ServiceAccount) for simplicity. The Helm chart already has one ServiceAccount — both deployments reference it. The controller already has the needed RBAC.

### 5. Fix sidecar stderr by adding a second goroutine

The sidecar's `monitorProcess` will spawn a second goroutine to read stderr and emit `OUTPUT_TYPE_STDERR` events. Both goroutines feed the same output channels.

### 6. E2E tests use the gRPC API as entry point

New E2E tests will:
1. Connect to the API server via ConnectRPC
2. Call `CreateAgentRun`
3. Verify CRD appears in K8s
4. Watch for phase transitions via `WatchAgentRun` (or poll `GetAgentRun`)
5. Verify agent pod is created and completes
6. Verify final status in both K8s CRD and API response

These run against the live k0s cluster with all components deployed.

## Risks / Trade-offs

- **[Risk] API server now depends on K8s** → This is intentional. The API server is a K8s-native component. It already runs in-cluster.
- **[Risk] Temporal workflow fix could break existing workflows** → There are no existing workflows in production. The nil pointer means nothing has ever run successfully.
- **[Risk] Shared ServiceAccount gives apiserver more permissions than needed** → Acceptable for now; can be split later if multi-tenancy becomes a concern.
- **[Risk] Proto field additions are additive** → No breaking change. Old clients ignore new fields.
