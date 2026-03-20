## Why

The planning agent is told to use `openspec new change` and `openspec validate` via its system prompt, but it ignores the instructions and writes flat files instead (`proposal.md`, `specs/when_then_scenarios.md`). This means the OpenSpec change has no `.openspec.yaml`, no proper schema, wrong directory structure, and `openspec validate` correctly rejects it. The agent can't be trusted to use OpenSpec CLI on its own — the Temporal activity must scaffold the change BEFORE the agent starts, then tell the agent to fill in the artifacts.

## What Changes

- **PlanRun scaffolds the change** via ExecCommand before starting the agent: `openspec init` + `openspec new change "<run-id>"` + `openspec status --change "<run-id>" --json` to get the artifact list
- **PlanRun tells the agent exactly what files to write** — instead of a vague "create an OpenSpec change", the prompt includes the exact file paths from `openspec status` output (e.g., "Write the proposal to `openspec/changes/ar-xyz/proposal.md`")
- **PlanRun uses `openspec instructions` for each artifact** — gets the template and schema-specific guidance, includes it in the agent prompt so the agent knows the exact format
- **Install OpenSpec extension for pi** — add `@fission-ai/openspec` to the pi agent's installed extensions so it has access to OpenSpec skills
- **Agent prompt includes artifact templates** — the WHEN/THEN format, the proposal structure, the tasks format

## Capabilities

### New Capabilities

None.

### Modified Capabilities
- `run-pipeline`: PlanRun scaffolds the OpenSpec change via CLI before the agent starts, provides structured artifact templates in the prompt.
- `container-images`: Sidecar installs OpenSpec pi extension alongside the CLI.

## Impact

- `internal/temporal/activities_spec_driven.go` — PlanRun rewritten to scaffold change, get instructions, build structured prompt
- `docker/Dockerfile.sidecar` — Install OpenSpec pi extension
- `internal/sidecar/gateway.go` — Plan stage system prompt updated to reference real file paths
