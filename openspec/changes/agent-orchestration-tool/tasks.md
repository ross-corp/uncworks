## 1. Protocols & CRD Definitions

- [ ] 1.1 Define Protobufs: Create `api.proto` (Client <-> Control Plane) and `agent.proto` (Control Plane <-> Pod Sidecar)
- [ ] 1.2 Generate Go gRPC code from Protobufs
- [ ] 1.3 Define Kubernetes CRDs for `AgentTemplate` and `AgentRun`
- [ ] 1.4 Generate Go client and controller boilerplate for CRDs

## 2. Go Control Plane Foundation

- [ ] 2.1 Set up the API Server (gRPC/REST) in Go
- [ ] 2.2 Implement the K8s Controller to watch `AgentRun` CRDs and manage Pod lifecycle
- [ ] 2.3 Implement the PostgreSQL database schema and connection logic (Shared Brain)
- [ ] 2.4 Build the basic Job Queuing and priority logic in the Orchestrator

## 3. Execution Pod Components

- [ ] 3.1 Build the Go-based Hydration Init-Container for Git Worktree provisioning
- [ ] 3.2 Build the Go-based RPC Gateway Sidecar (gRPC to stdin/stdout)
- [ ] 3.3 Create the Base Docker Image with `devbox` and `pi-mono` runtime dependencies

## 4. Agent Harness (pi-mono Extension)

- [ ] 4.1 Create the TypeScript `pi-aot-extension` for the `pi-mono` harness
- [ ] 4.2 Implement OTel tracing logic within the extension (Span propagation)
- [ ] 4.3 Implement the `/ask_human` tool and RPC signal for HITL workflow

## 5. Client Interfaces (TUI & Web)

- [ ] 5.1 Build the Bubbletea-based TUI Fleet Dashboard in Go
- [ ] 5.2 Build the Next.js Web UI for OTel trace visualization and agent monitoring
- [ ] 5.3 Implement the `aot open` CLI command to locate and open local worktrees

## 6. Advanced Multi-Agent Orchestration

- [ ] 6.1 Implement the `spawn_junior` tool for the Senior Agent harness
- [ ] 6.2 Add pgvector RAG endpoints to the Control Plane for cross-run memory
- [ ] 6.3 Implement the Multi-Agent Review Loop workflow (Junior PR -> Senior Review)
