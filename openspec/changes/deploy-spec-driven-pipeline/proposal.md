## Why

The spec-driven pipeline (Plan → Execute → Verify) is fully implemented in code but has never run end-to-end in the cluster. The sidecar image doesn't have `openspec` installed yet (Dockerfile was updated but never rebuilt/deployed). The `PlanRun` and `VerifyRun` activities haven't been exercised against real agents. Until this is deployed and validated, the entire spec-driven feature is theoretical. We need to build, deploy, run the first spec-driven agent, and fix whatever breaks.

## What Changes

- Rebuild sidecar Docker image with `openspec` CLI installed
- Deploy all updated images to `aot-local` (k0s cluster)
- Apply updated CRD with spec-driven orchestration mode and new status fields
- Create a test spec-driven run and observe it through plan → execute → verify stages
- Fix the `execInSidecar` function which currently spawns a full pi-agent just to run bash commands — replace with direct command execution via the sidecar's existing exec capability
- Fix any runtime issues discovered during the first real spec-driven run (activity registration, stage transitions, verification gate failures)
- Add a lightweight sidecar RPC for running arbitrary commands in the workspace (needed by verification gates)
- Verify the web UI correctly displays stage badges, verification results, and retry history

## Capabilities

### New Capabilities
- `sidecar-exec`: A lightweight command execution RPC on the sidecar that runs bash commands directly in the workspace without spawning a full pi-agent. Used by verification gates to run `openspec` CLI commands, test suites, and file checks.

### Modified Capabilities

None.

## Impact

- `docker/Dockerfile.sidecar` — Already updated (openspec install), needs rebuild
- `internal/sidecar/gateway.go` — Add `ExecCommand` RPC handler for lightweight bash execution
- `proto/aot/agent/v1/agent.proto` — Add `ExecCommand` RPC definition
- `internal/temporal/activities_spec_driven.go` — Replace `execInSidecar` (agent-based) with direct `ExecCommand` RPC calls
- `deploy/crds/agentrun-crd.yaml` — Already updated, needs `kubectl apply`
- All control plane deployments — Need image update and rollout restart
- `web/` — Already updated with stage badges and verification panel, needs rebuild
