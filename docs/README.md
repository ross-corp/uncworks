# UNCWORKS Documentation

UNCWORKS is a Kubernetes-native platform for orchestrating autonomous coding agents. It manages the full lifecycle of agent runs -- from provisioning sandboxed workspaces and cloning repositories to executing LLM-driven coding tasks with human-in-the-loop oversight.

The system uses a control plane architecture (API server, controller, Temporal worker) that schedules agent pods, routes LLM requests through a LiteLLM proxy, and exposes a web dashboard for monitoring and interaction.

## Getting Started

- [Quick Start](getting-started.md) -- Prerequisites, cluster setup, first run

## Guides

- [Creating Runs](guides/creating-runs.md) -- Submitting agent runs via the web UI
- [Spec-Driven Pipeline](guides/spec-driven.md) -- Plan/Execute/Verify with OpenSpec
- [Model Configuration](guides/models.md) -- LiteLLM proxy, Ollama, OpenRouter, model tiers

## Reference

- [ConnectRPC API](reference/api.md) -- Client API: CreateAgentRun, WatchAgentRun, SendHumanInput, etc.
- [AgentRun CRD](reference/crd.md) -- Kubernetes custom resource spec and status fields
- [Determinism Extension](reference/extension.md) -- Tools, policies, and guardrails for agent behavior
- [Helm Values](reference/helm-values.md) -- All configurable Helm chart values

## Contributing

- [Local Development](contributing/development.md) -- k0s cluster, Taskfile commands, building images
- [Testing](contributing/testing.md) -- Unit tests, contract tests, E2E, pre-commit hooks
