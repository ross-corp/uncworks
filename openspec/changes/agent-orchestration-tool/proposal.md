# Proposal: Agent Orchestration Tool (AOT)

## Goal
Create an agent orchestration tool for local and remote use, inspired by Stripe's "Minions" system. It provides a "Cloud Native OS for AI Engineers"—treating agents as containerized, orchestratable workloads with guaranteed environments and standard I/O.

## Architectural Pillars
1. **Kubernetes-First Orchestration**: Utilize K8s for both local (`kind`, `k3d`) and remote (EKS, GKE) execution.
2. **Ephemeral "Devboxes"**: Agents run in isolated, disposable pods (or KubeVirt VMs) pre-configured via `devbox.json` or `devcontainer.json`.
3. **Agent Harness**: Use `pi-mono` as the core agent execution loop within the pods.
4. **Multi-Agent Collaboration**: Support long-lived processes with agents playing specific roles (Senior, Junior, Reviewer).
5. **Unified Client**: A TUI and Web client for 1.0.0, providing both RPC and UI interfaces.
6. **Composability**: Agents leverage existing patterns like MCP (Model Context Protocol) and skills.

## Core Components
- **Control Plane**: K8s-based API server for job dispatching, model routing (via LiteLLM Proxy), and state management (Redis/PG).
- **Execution Plane**: Ephemeral pods/VMs with `pi-mono` harness, workspace mounts (PVC/Sync), and SSH access.
- **Client Interface**: TUI and Web clients for monitoring and interacting with agents.

## Design Decisions
- **Enforced Environment**: Defaulting to `devbox.json` for environment reproducibility.
- **Agent Lifecycle**: Unattended one-shot tasks with automated feedback (lint, test) as the primary mode.
- **Collaborative Flow**: Agents communicate via internal event buses or shared workspaces to parallelize tasks.
- **Editor Integration**: First-class support for Neovim and other standard IDEs.

## Success Criteria (1.0.0)
- Functional K8s-based orchestrator.
- TUI and Web client interfaces.
- Support for `pi-mono` harness.
- Ephemeral, reproducible execution environments.
- Multi-agent collaboration patterns.
