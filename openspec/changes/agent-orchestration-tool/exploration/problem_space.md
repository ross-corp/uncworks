# Full Problem Space & Architecture Enumeration

To build a production-grade Agent Orchestration Tool (AOT), we must solve several interlocking problems. This document maps the entire problem space, refines the architectural solutions, and breaks them into discrete, ordered tasks.

## 1. The Problem Space

### A. Infrastructure & Lifecycle Management
*   **Problem:** Agents need to be spun up, monitored, and torn down reliably without leaving zombie processes.
*   **Solution:** Kubernetes (K8s) Custom Resource Definitions (CRDs). An `AgentRun` CRD represents the lifecycle of a task. The Go Orchestrator watches these CRDs and schedules Pods.

### B. Environment Isolation & Reproducibility
*   **Problem:** "It works on my machine" but fails in the agent's environment. Agents need exact dependency matches without requiring custom Dockerfiles for every project.
*   **Solution:** Enforce `devbox.json`. The agent pod boots a lightweight universal base image and uses `devbox shell` to instantly provision the exact Go/Node/Python environment the project requires.

### C. Fast Workspace Syncing
*   **Problem:** `git clone` is too slow for spinning up 10 Junior agents. Traditional volume mounts can have bad I/O performance or locking issues.
*   **Solution:** The "Sesh" Pattern via Git Worktrees. The node maintains a `bare` clone on a Local Path Provisioner volume. Each agent pod creates an instant Git Worktree (`git worktree add`) pointing to that bare repo, giving it an isolated but instantly ready workspace.

### D. Observability & Debugging
*   **Problem:** AI agents are black boxes. When an agent gets stuck in a loop, it's hard to know why or what tools it tried to use.
*   **Solution:** OpenTelemetry (OTel) baked into the Agent Harness. Every LLM call, bash command, and file edit emits a span. These are collected by an in-cluster OTel Collector and visualized in the Web/TUI clients.

### E. Multi-Agent State & Communication
*   **Problem:** A Senior Agent needs to track the progress of 5 Junior Agents without holding the entire context in its prompt.
*   **Solution:** A "Shared Brain" powered by PostgreSQL (with pgvector). The Control Plane manages job queues and state. Junior agents communicate status back via gRPC, and the Senior agent queries the Control Plane for updates.

### F. Human-in-the-Loop (HITL) & Editor Integration
*   **Problem:** If an agent needs an OTP code or hits an ambiguous requirement, it shouldn't just fail or guess.
*   **Solution:** An `/ask_human` tool inside the harness that pauses the execution loop, sends a gRPC event to the Control Plane, and prompts the user via the TUI/Web UI. For deeper inspection, the CLI command `aot open <run-id>` opens the local worktree in the user's `$EDITOR`.

### G. Composability & Harness Swapping
*   **Problem:** Being locked into a single agent framework (`pi-mono`, `goose`, `autogpt`).
*   **Solution:** The Sidecar Pattern. The Control Plane talks to a Go-based `RPC Gateway Sidecar` via standard gRPC (`Agent.proto`). This sidecar translates the commands (via stdin/stdout) to the specific underlying harness.

---

## 2. Refined Architecture Details

### The K8s Pod Layout (The Execution Envelope)
When an `AgentRun` is scheduled, the resulting Pod contains:
1.  **Init Container (Hydration):** Written in Go. Pulls `devbox.json`, clones the bare repo, creates the Git Worktree, and fetches K8s Secrets.
2.  **Sidecar (RPC Gateway):** Written in Go. Exposes gRPC. Pipes commands to the Harness.
3.  **Main Container (Harness & Execution):** Runs the `pi-mono` loop. It executes shell commands within a `devbox` environment against the mounted Git Worktree.

### The Control Plane (Go)
1.  **API Gateway:** Handles TUI/Web websocket connections and REST/gRPC traffic.
2.  **Orchestrator Controller:** Watches `AgentRun` CRDs, manages quotas, and handles the job queue.
3.  **State Manager:** Interfaces with PostgreSQL for long-term memory and cross-run state.

---

## 3. Discrete Implementation Tasks (In Order)

### Phase 1: Foundation & Protocols
1.  **Define Protobufs:** Create `api.proto` (Client <-> Control Plane) and `agent.proto` (Control Plane <-> Pod Sidecar).
2.  **Define CRDs:** Write the Go structs and generate the K8s manifests for `AgentTemplate` and `AgentRun`.
3.  **Database Schema:** Design the PostgreSQL schema for Jobs, Queueing, and Memory.

### Phase 2: The Go Control Plane MVP
1.  **API Server Setup:** Initialize the Go gRPC server and WebSocket handlers.
2.  **K8s Controller:** Build the Go operator that watches `AgentRun` objects and creates basic Pods.
3.  **Job Queue Logic:** Implement the prioritization and max-worker limit logic in the controller.

### Phase 3: The Execution Pod (Hydration & Sidecar)
1.  **Hydration Init-Container:** Build the Go binary that creates Git Worktrees from a mounted Local Path PV.
2.  **RPC Gateway Sidecar:** Build the Go binary that receives gRPC from the Control Plane and translates it to `stdin/stdout` JSONL.
3.  **Base Image:** Create the minimal Dockerfile containing `devbox` and the basic utilities required by the harness.

### Phase 4: The Agent Harness (`pi-mono` Extension)
1.  **AOT Extension Plugin:** Write the TypeScript extension for `pi-mono`.
2.  **OTel Integration:** Hook into `pi.on("tool_call")` to emit OTel traces.
3.  **HITL Tool:** Implement the `/ask_human` tool logic that signals the RPC Gateway.

### Phase 5: Client Interfaces (TUI & Web)
1.  **Go TUI MVP:** Use `bubbletea` to build the fleet dashboard, log tailing, and HITL prompt.
2.  **Next.js Web MVP:** Build the React equivalent with integrated OTel trace visualizations.
3.  **`aot open` CLI:** Implement the logic to locate the K8s Local Path volume and execute `$EDITOR`.

### Phase 6: Advanced Orchestration (Multi-Agent)
1.  **Senior Agent Tools:** Add tools to the harness to allow an agent to request the creation of a child `AgentRun`.
2.  **Shared Memory API:** Implement the pgvector RAG endpoints in the Control Plane for agents to query past runs.
3.  **Review Loop Logic:** Implement the workflow where a Junior agent opens a PR, and a Senior agent is automatically triggered to review it.
