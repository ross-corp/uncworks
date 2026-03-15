## Why

Complex tasks hit the limits of a single-agent run. A prompt like "refactor the auth module, update all tests, and fix the CI pipeline" forces one agent to context-switch across unrelated concerns, leading to shallow work and missed details. The system already has `SpawnJuniorWorkflow` in `internal/temporal/workflow.go` that can create child workflows, but there is no structured way to express "this spec should decompose into multiple agents" — the primitive exists without the orchestration layer.

Decomposition is how humans handle complexity: break the problem down, assign pieces, integrate results. The same principle applies to agent runs. A senior agent that reads a spec and decomposes it into focused junior tasks will produce better results than a single agent trying to do everything at once. Each junior does one thing well (Unix philosophy), and the senior aggregates the outputs.

## What Changes

- **SpecRun concept**: A new `SpecRun` abstraction that wraps one or more `AgentRun` resources. A SpecRun represents the full execution of a spec — either a single AgentRun (simple mode) or a tree of AgentRuns (orchestrated mode). Implemented as labels and annotations on existing AgentRun CRDs rather than a new CRD, keeping the resource model flat.
- **Senior/junior auto-decomposition**: When a spec is complex (multiple requirements, cross-cutting concerns), a senior agent reads the spec, produces a structured decomposition plan (JSON), and spawns junior AgentRuns for each sub-task. The senior then waits for juniors to complete and reviews/integrates their outputs.
- **Explicit orchestration mode**: Users can define the run tree directly in the spec YAML via an `orchestration` field, specifying sub-tasks and their dependencies. This is opt-out from auto-decomposition — the user takes control of the decomposition.
- **Parent-child tracking**: A `parentRunID` field on `AgentRunSpec` links junior runs to their senior. Combined with a `specRunID` label that groups all runs from the same spec execution, this forms a queryable run graph.
- **Orchestration modes**: Three modes — `auto` (senior decides decomposition), `manual` (user defines tree), `single` (no decomposition, current behavior). Defaults to `single` for backward compatibility.

## Capabilities

### New Capabilities
- `spec-run-orchestration`: SpecRun label/annotation system grouping related AgentRuns under a single spec execution, with orchestration mode selection
- `auto-decomposition`: Senior agent reads spec, produces structured decomposition plan, spawns junior AgentRuns, waits for completion, integrates results
- `manual-orchestration`: User defines the orchestration tree in spec YAML; controller spawns the defined sub-tasks without senior agent involvement
- `run-graph-tracking`: Parent-child relationships via `parentRunID` on AgentRunSpec, queryable run graph via labels, tree visualization data model

### Modified Capabilities
- Existing `AgentRunWorkflow` gains an orchestration preamble: before executing, check orchestration mode and potentially spawn children
- Existing `AgentRunSpec` (proto and CRD) gains `parent_run_id`, `orchestration_mode`, and `orchestration` fields

## Impact

- `proto/aot/api/v1/api.proto` — new fields on `AgentRunSpec`: `parent_run_id`, `orchestration_mode`, `orchestration` message
- `api/v1alpha1/types.go` — matching CRD type changes for orchestration fields
- `deploy/crds/agentrun-crd.yaml` — CRD schema update for new fields
- `internal/temporal/workflow.go` — orchestration preamble in `AgentRunWorkflow`, new `OrchestrateSpecWorkflow` or extension of existing workflow
- `internal/controller/agentrun_controller.go` — propagate `specRunID` label, handle parent-child lifecycle
- `web/` — run graph visualization showing parent-child relationships, progress through the tree
- `test/` — E2E tests for auto and manual orchestration flows
