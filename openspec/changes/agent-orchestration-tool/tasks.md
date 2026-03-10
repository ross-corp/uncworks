## 1. Foundation, Testing Infra & Protocols

- [x] 1.1 Set up Local Testing Environment: Install `k0s` and initialize a single-node cluster with `kine` (SQLite).
- [x] 1.2 Initialize Playwright E2E suite and verify against a dummy SolidJS app.
- [x] 1.3 Define Protobufs: Create `api.proto` (Client <-> Control Plane) and `agent.proto` (Control Plane <-> Sidecar).
- [x] 1.4 Define `AgentRun` CRD with support for `Pod`, `KubeVirt`, and `External` backends (Golang).
- [x] 1.5 Write integration tests for CRD lifecycle using Go's `envtest`.

## 2. Go Control Plane & Shared Logic

- [x] 2.1 Set up the Go API Server with gRPC and WebSocket support.
- [x] 2.2 Implement the K8s Controller to watch `AgentRun` CRDs (Pod-only initially, stubs for others).
- [ ] 2.3 Set up PostgreSQL Shared Brain and write unit tests for agent state persistence.
- [ ] 2.4 Create a shared TypeScript logic package (`@aot/shared`) for gRPC clients and Solid stores.

## 3. Execution Pod & Devbox

- [ ] 3.1 Build Go-based Hydration Init-Container and verify with integration tests.
- [ ] 3.2 Build Go-based RPC Gateway Sidecar and verify with gRPC contract tests.
- [ ] 3.3 Create a base Docker image with `devbox` and `bun` runtime pre-installed.
- [ ] 3.4 Implement automated tests for the `devbox shell` execution context.

## 4. Agent Harness (pi-mono Extension)

- [ ] 4.1 Develop the `pi-aot-extension` (TypeScript) and verify with Bun test runner.
- [ ] 4.2 Implement OTel tracing and verify span emission via an OTel Collector sidecar.
- [ ] 4.3 Implement `/ask_human` tool and verify HITL signaling via a mock gRPC client.

## 5. Client Interfaces (SolidJS + OpenTUI)

- [ ] 5.1 Build the SolidJS Web UI and write Playwright tests for agent monitoring.
- [ ] 5.2 Build the SolidJS TUI using OpenTUI and verify terminal rendering.
- [ ] 5.3 Integrate shared `@aot/shared` logic into both UIs and verify reactive state sync.
- [ ] 5.4 Implement the `aot open` CLI command (Go) to find local worktrees and open `$EDITOR`.

## 6. Verification & Roadmap

- [ ] 6.1 Execute full system E2E test suite (Local k0s + Playwright + Agent fixed PR).
- [ ] 6.2 Implement Multi-Agent "Senior" tool (`spawn_junior`) and verify child pod creation.
- [ ] 6.3 Final regression test pass across all "Testing Taxonomy" layers.
