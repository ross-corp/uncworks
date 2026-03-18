## Why

The spec-driven pipeline (Plan → Execute → Verify) is implemented but has never run end-to-end. The first live test revealed two critical problems: (1) the free cloud model is too slow/rate-limited, causing 10+ minute planning stages and execution timeouts, and (2) every pipeline stage uses the same hardcoded model, timeout, and retry config — there's no way to tune stage behavior without changing code. Production customers need to configure each stage independently: fast cheap models for planning, powerful models for execution, different timeouts, retry policies, and failure hooks per stage.

## What Changes

- **Stage configuration system**: Each pipeline stage (plan, execute, verify) gets its own configuration block with model, timeout, retries, and onFailure behavior. Configuration flows from CRD spec → Temporal workflow → sidecar agent invocation.
- **`PipelineConfig` on AgentRunSpec**: New CRD/proto field that lets users (or the UI) specify per-stage settings when creating a run.
- **Sensible defaults**: Plan uses fast model with 5min timeout, Execute uses capable model with 15min timeout and 3 retries, Verify uses fast model with 3min timeout.
- **ExecCommand RPC** on the sidecar for lightweight bash execution (already implemented).
- **Cleanup retry cap** at 5 attempts (already implemented — was infinite, flooded workers).
- **`pollUntilAgentDone` timeout** handling for UNSPECIFIED agent state (already implemented).
- Deploy everything to aot-local and validate end-to-end.

## Capabilities

### New Capabilities
- `sidecar-exec`: Lightweight command execution RPC on the sidecar (already implemented).
- `pipeline-config`: Per-stage configuration for spec-driven runs — model, timeout, retries, onFailure hook per stage.

### Modified Capabilities
- `run-pipeline`: Pipeline stages now read configuration from the run spec instead of hardcoded defaults.

## Impact

- `api/v1alpha1/types.go` — Add `PipelineConfig` struct with `PlanConfig`, `ExecuteConfig`, `VerifyConfig` to AgentRunSpec
- `deploy/crds/agentrun-crd.yaml` — Add `pipelineConfig` to spec schema
- `proto/aot/api/v1/api.proto` — Add `PipelineConfig` message to `AgentRunSpec`
- `internal/temporal/workflow_spec_driven.go` — Read stage config from WorkflowInput, pass to activities
- `internal/temporal/activities_spec_driven.go` — Use per-stage model and timeout from config
- `internal/sidecar/gateway.go` — StartAgent respects model override from env/config
- `web/src/components/AgentRunForm.tsx` — Add pipeline config section for spec-driven mode (expandable advanced settings)
- `web/src/types/agent-run.ts` — Add PipelineConfig types
- All previously listed changes (ExecCommand, deploy, validate)
