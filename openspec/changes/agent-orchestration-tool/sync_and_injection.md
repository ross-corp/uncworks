# Design: AOT Go Backend & Workspace Sync Architecture

## 1. Core Philosophy: The Go-First Backend
The AOT Control Plane and its associated services (Orchestrator, API Gateway, RPC Gateway) are built in **Go**. This ensures high performance, static typing for complex CRD management, and excellent support for gRPC and K8s client-go.

### Key Backend Components (Go)
*   **`aot-orchestrator`**: The heart of the system. Manages the `AgentRun` CRD lifecycle and talks to the K8s API.
*   **`aot-api`**: The gRPC/REST gateway for the TUI and Web clients.
*   **`aot-rpc-gateway`**: The sidecar that runs inside the Agent Pod, translating standard input/output into gRPC streams.

---

## 2. Workspace Management: The "Sesh" Pattern + K8s Workflows

We will adopt the **Worktree-First** approach inspired by `sesh`. This allows agents to work on the same repository in parallel without clobbering each other's state, and enables seamless human hand-off.

### A. Worktree Management (Local Path Provisioner)
*   **The Hub:** A central directory (e.g., `/mnt/aot/repos`) holds the `bare` clones of the repositories.
*   **The Spoke:** Every `AgentRun` gets its own **Git Worktree** in a unique path (e.g., `/mnt/aot/worktrees/run-123`).
*   **Provisioning:** We use the **K8s Local Path Provisioner**. On a single-node (local) cluster, this maps directly to your host's disk. On a remote cluster, it maps to high-performance local NVMe.
*   **Why Worktrees?**
    1.  **Instant Setup:** Creating a worktree is significantly faster than a full clone.
    2.  **Shared Cache:** The objects are shared in the `bare` repo, but the working directory is isolated.
    3.  **Human Integration:** A user can `cd` into the same worktree the agent is using to inspect progress.

### B. Human Editor Integration
*   **The `aot open` Command:** A CLI utility (Go) that identifies the active worktree of an agent.
*   **The Flow:**
    1.  User: `aot open run-123`
    2.  AOT finds the path: `/mnt/aot/worktrees/run-123`.
    3.  AOT executes `$EDITOR /mnt/aot/worktrees/run-123`.
    4.  Because it's a local path (or synced via Mutagen/Sshfs for remote), your local Neovim/VSCode just works.

---

## 3. UI Layer: Parity & Extensibility

### A. TUI (Bubbletea + OpenTUI)
*   **Implementation:** Go (using `charmbracelet/bubbletea` and `lipgloss`).
*   **Features:** Dashboard view of the "Fleet", real-time streaming of agent logs, and an interactive "Ask Human" prompt.

### B. Web (Next.js + Tailwind + gRPC-Web)
*   **Implementation:** TypeScript/React.
*   **Parity:** Uses the same gRPC service definitions as the TUI.
*   **Visuals:** Richer OTel trace visualizations (using Jaeger-style gantt charts) and project-level health metrics.

---

## 4. The Injection Pattern: Great UX & Security

How do we inject secrets and context without making it a "configuration nightmare"?

### A. The "Hydration" Sidecar
Every Agent Pod has a **Go-based Hydration Sidecar** that runs *before* the agent harness.
1.  **Context Fetching:** It pulls the latest `devbox.json`, `AGENTS.md`, and task instructions.
2.  **Secret Injection:** It pulls secrets from K8s Secrets (or an OIDC provider) and populates them into the environment or a `.env` file *only* for the duration of the run.
3.  **Worktree Setup:** It runs the `git worktree add` command to prepare the workspace.

### B. The "sensible defaults" Injection
To ensure **Readability and Composability**, the system injects a standard "Agent Environment" that includes:
*   **`AOT_TRACE_PARENT`**: For OTel propagation.
*   **`AOT_RPC_URL`**: For the agent to talk back to the gateway.
*   **`AOT_WORKSPACE_ROOT`**: The path to the worktree.

---

## 5. Architectural Diagram: Go Backend & Sync

```text
       [ TUI Client (Go) ]       [ Web Client (Next.js) ]
                │                         │
                └───────────┬─────────────┘
                            │ (gRPC / Protobuf)
                            ▼
                  [ AOT API Server (Go) ]
                            │
            ┌───────────────┴───────────────┐
            ▼                               ▼
    [ Orchestrator ]               [ Shared Brain ]
    (K8s CRD Controller)           (Postgres + pgvector)
            │
            ▼
    [ Agent Pod (K8s) ]
    ┌──────────────────────────────────────────────────────────┐
    │  [ RPC Gateway Sidecar (Go) ] <─▶ [ pi-mono Harness ]    │
    │             │                             │              │
    │             └─────────────┬───────────────┘              │
    │                           ▼                              │
    │  [ Devbox Container ] <──▶ [ Local Path / Git Worktree ] │
    └──────────────────────────────────────────────────────────┘
```
