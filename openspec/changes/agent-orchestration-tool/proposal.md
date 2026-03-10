## Why

Existing AI coding agents are either too coupled to a specific IDE (Cursor) or lack the robust, isolated infrastructure (Stripe Minions) needed for production-grade, multi-agent workflows. AOT (Agent Orchestration Tool) provides a "Cloud Native OS for AI Engineers"—treating agents as containerized, orchestratable workloads with guaranteed environments and deep observability.

## What Changes

- **Kubernetes-First Orchestration**: Utilize `k0s` for a lightweight, single-binary Kubernetes distribution, initially deployed via manual scripts/Helm.
- **Backend-Agnostic AgentRuns**: The `AgentRun` CRD SHALL support multiple execution backends: `Pod`, `KubeVirt`, or `External` (SSH/Lima). Initial support is for `Pod`.
- **SolidJS Everywhere**: Use SolidJS for both the Web UI and the TUI (via the OpenTUI framework) for a reactive, shared-logic frontend experience.
- **Test-First Mandate**: Every feature, bug fix, or requirement change MUST include corresponding E2E, integration, unit, and regression tests.
- **Ephemeral Devboxes**: Enforce environment reproducibility using `devbox.json` with the `bun` runtime pre-installed.
- **Multi-Agent Collaboration**: Support long-lived processes with agents playing specific roles (Senior, Junior, Reviewer).
- **Deep Observability**: Integrate OpenTelemetry (OTel) for tracing every action taken by the agent harness.

## Capabilities

### New Capabilities
- `k8s-orchestrator`: Manages the lifecycle of AgentRun CRDs on a k0s cluster with support for Pod, KubeVirt, and External backends.
- `agent-harness`: RPC Gateway sidecar and `/ask_human` HITL workflow.
- `workspace-management`: Git Worktree provisioning and devbox shell environment isolation.
- `client-interfaces`: High-performance SolidJS/OpenTUI terminal dashboard and SolidJS/Playwright web dashboard.
- `observability`: OTel trace propagation from the frontend through the orchestrator to the agent.
- `testing-infra`: Automated E2E and integration suites to ensure the "Zero-Regression" mandate.

### Modified Capabilities
- None

## Impact

- **Infrastructure**: Requires a k0s cluster (local or remote).
- **Dependencies**: Go (backend), Bun/TypeScript (frontend/agent), PostgreSQL (shared brain), OTel Collector.
- **Workflow**: Shifting from manual testing to a "Test-as-Source-of-Truth" model for all development.
