# Design: Agent Orchestration Tool (AOT) Control Plane & Execution

## Architecture Overview

The system is designed as a **Cloud-Native Service-Oriented Architecture (SOA)** built on Kubernetes. It abstracts the "Agent Harness" (initially `pi-mono`) from the Orchestration and State layers, ensuring composability and observability from day one.

### Core Philosophy
- **Loose Coupling:** The Orchestrator does not care *how* the agent works, only that it complies with an RPC/Event interface.
- **Observability First:** OpenTelemetry (OTel) tracing is baked into the control plane and injected into the agent execution harness.
- **Kube-Native Scheduling:** Leverage Kubernetes Custom Resource Definitions (CRDs) for stateful agent definitions, enabling Knative for scale-to-zero or event-driven execution in the future.

---

## 1. The Control Plane (Kubernetes Services)

### A. The API Server & Orchestrator (Go / Rust)
The central nervous system of AOT.
*   **Role:** Exposes gRPC/REST endpoints for the Web and TUI clients. Manages the lifecycle of Agent workloads.
*   **CRD Management:** Translates a user request (e.g., "Fix Issue #42") into a K8s `AgentRun` Custom Resource.
*   **Routing:** Handles human-in-the-loop (HITL) communication by routing WebSocket streams from the Client to the specific `AgentRun` Pod.

### B. Shared Brain (PostgreSQL / Vector DB)
*   **Role:** The persistent state for multi-agent collaboration.
*   **Implementation:** A StatefulSet running PostgreSQL (with pgvector for RAG/Memory) deployed in-cluster by default, with configuration options to point to external databases (e.g., Supabase, RDS).
*   **Function:** Stores project context, previous run history, and long-term agent memory.

### C. Observability Stack (OpenTelemetry Collector)
*   **Role:** Central aggregation for distributed traces.
*   **Function:** The API Server, the LiteLLM Proxy, and the Agent Harness all push traces to this collector (which can then forward to Jaeger, Datadog, etc.).

---

## 2. The Execution Plane (The Agent Pod)

When an `AgentRun` CRD is created, K8s schedules a Pod (or a KubeVirt VM).

### The Container Layout
Instead of a monolithic container, we use a **Sidecar Pattern** within the Pod to enforce service boundaries.

#### Container 1: The Devbox Environment (User Code)
*   **Image:** A base Ubuntu image dynamically provisioned via `devbox.json` (or `devcontainer.json`).
*   **Role:** The sandbox where code is checked out, built, and tested.

#### Container 2: The Agent Harness (`pi-mono`)
*   **Role:** The LLM execution loop. It interacts with the Devbox container via shared volumes and localized IPC.
*   **Extension API:** We utilize `pi-mono`'s Extension SDK to build an "AOT Extension". This extension:
    1.  Hooks into `pi.on("tool_call")` to emit OTel Spans for every tool execution.
    2.  Provides a custom `/ask_human` tool that pauses the session and sends an RPC event back to the Control Plane for HITL.

#### Container 3: The RPC Gateway (Sidecar)
*   **Role:** Translates `pi-mono`'s internal JSONL/RPC over stdin/stdout into standard gRPC streams that the Control Plane can route to the frontend clients.
*   **Why?** This ensures we can swap out `pi-mono` in the future for another harness (like AutoGPT or a custom Python loop) simply by writing a new Gateway sidecar that implements our standard `Agent.proto` interface.

---

## 3. Interaction Models

### Human-in-the-Loop (HITL)
1.  **Agent Pauses:** The `pi-mono` harness (via our AOT extension) hits a blocked state or uses the `/ask_human` tool.
2.  **Event Emitted:** The RPC sidecar catches this and emits a `WaitingForInput` event to the Control Plane via gRPC.
3.  **Client Notification:** The Control Plane pushes this to the TUI/Web client via WebSocket.
4.  **Human Responds:** The user types a response, which flows back down to the specific Pod's stdin.

### Multi-Agent Communication
*   **The Orchestrator Approach:** A "Senior Agent" pod emits an event to the Control Plane requesting a sub-task. The Control Plane creates a *new* `AgentRun` CRD for a "Junior Agent".
*   **State Sharing:** They do not communicate directly. They share state by committing code to a Git branch or writing records to the Shared PostgreSQL Brain.

---

## 4. Why CRDs and Knative?
*   **CRDs (`AgentTemplate`, `AgentRun`):** Allow us to use standard `kubectl` to monitor agents. You could type `kubectl get agentruns -w` to see the status of your AI fleet.
*   **Knative Eventing (Future-Proofing):** By building on K8s, we can eventually bind agents to Knative event sources. Example: A GitHub Webhook fires (Push event) -> Knative creates an `AgentRun` to review the code -> Agent spins up, reviews, posts a comment, and spins down to zero.
