## Context

The AOT architecture aims to provide a unified environment for both local and remote agent orchestration. This update shifts the UI stack from a Go-based TUI to a **SolidJS-based TUI and WebUI** to ensure high reactive performance and shared component logic. We are also adopting **k0s** as our Kubernetes distribution, introducing **backend-agnostic AgentRuns**, and enforcing a **Test-First mandate**.

## Goals / Non-Goals

**Goals:**
- **SolidJS Everywhere**: Use a single reactive framework (SolidJS) for both the Web Dashboard and the Terminal Dashboard (via OpenTUI).
- **Test-First Reliability**: 100% verification for every change (E2E, integration, unit, smoke, race, contact).
- **Multi-Backend AgentRuns**: Support `Pod` (initial), `KubeVirt`, and `External` (SSH/Lima/VM) execution modes in the `AgentRun` CRD.
- **K8s Isolation via k0s**: Use a lightweight, single-binary Kubernetes distribution (k0s) to manage the agent lifecycle.
- **Universal Devboxes**: Enforce the `bun` runtime in all agent pods via `devbox.json` for consistent tool execution.

**Non-Goals:**
- **Automated k0s installation**: The Go orchestrator will NOT manage k0s installation initially (handled via manual scripts/Helm).
- **Full KubeVirt/External implementation**: Only the `Pod` backend will be fully implemented in 1.0.0; others will be stubbed in the CRD and roadmap.

## Decisions

### 1. UI Stack: SolidJS + OpenTUI + Playwright
- **Rationale**: Building two separate UIs (Go TUI and React/Solid Web) doubles development time and state management complexity. Using SolidJS for both allows us to share business logic, gRPC clients, and state stores. OpenTUI (powered by Zig and Yoga Flexbox) gives us a high-performance terminal renderer that integrates with SolidJS.

### 2. Orchestration: k0s and Backend-Agnostic AgentRuns
- **Rationale**: k0s is a zero-friction, single-binary distribution. The `AgentRun` CRD will use a `Spec.Backend` field to switch between `Pod`, `KubeVirt`, or `External`. This future-proofs the system for deeper isolation (VMs) or external runners (SSH).

### 3. Execution: Bun + Devbox
- **Rationale**: Bun is extremely fast for agent tools (scripts, linting). By enforcing `devbox` in the agent pod, we ensure the agent has exactly the tools it needs without bloating the base container image.

### 4. Testing Taxonomy
- **Unit**: Go (backend), Solid/Jest (frontend).
- **Integration**: Go/Testcontainers for Postgres and K8s interactions.
- **E2E**: Full system flows using a real k0s cluster and Playwright.
- **Contract**: gRPC-based verification between Control Plane and Agent Pods.

## Risks / Trade-offs

- **[Risk] OpenTUI Maturity**: OpenTUI is high-performance but relatively new. → **Mitigation**: Maintain clear abstraction layers in the UI.
- **[Risk] Test Overhead**: A "Test-Everything" mandate increases implementation time. → **Mitigation**: Invest in fast CI (using Bun and k0s) to keep iteration speed high.
- **[Risk] KubeVirt Complexity**: Supporting VMs in K8s is complex. → **Mitigation**: Implement as a "Stub" first, ensuring the CRD schema supports it before the full controller logic is built.
