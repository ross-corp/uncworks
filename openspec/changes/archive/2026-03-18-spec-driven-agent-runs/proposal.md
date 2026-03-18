## Why

Agent runs currently report "Succeeded" when the agent process exits cleanly (exit code 0), regardless of whether the agent actually accomplished the task. A model that gives up after one failed tool call, hallucinates completion, or produces garbage output is indistinguishable from one that did real work. There is no programmatic way to evaluate whether a run accomplished its goal. This makes the platform unusable for production customers who need reliable outcomes, not just process lifecycle tracking.

## What Changes

- Introduce a **multi-stage run pipeline**: Plan → Execute → Verify, replacing the current single-stage "run agent until it stops" model.
- **Stage 1 (Plan)**: An agent generates a structured OpenSpec change from the user's input (prompt or spec), producing `proposal.md`, `specs/*.md` with WHEN/THEN acceptance criteria, and `tasks.md`. The OpenSpec CLI (`openspec validate --json`) ensures the spec is well-formed before proceeding.
- **Stage 2 (Execute)**: An agent implements the change using the spec and tasks as its guide. Task completion is tracked via `openspec list --json` (checkbox completion percentage).
- **Stage 3 (Verify)**: A hybrid evaluation stage. Automated checks run commands derived from spec scenarios (test suites, builds, file existence). An LLM judge evaluates semantic criteria against the git diff and agent log. If verification fails, Stage 2 retries with failure context (max 3 retries). On final failure, the run surfaces to the user as Failed with the verification report.
- Each run creates an OpenSpec change at `openspec/changes/<run-id>/` on the workspace PVC. On success, the change is archived. On failure, the change artifacts serve as the failure report.
- The "Succeeded" phase now means **verified against spec**, not just "process exited 0".
- Add a new Temporal activity for each stage (PlanRun, ExecuteRun, VerifyRun) orchestrated by the existing AgentRunWorkflow.
- Expose the eval spec and verification results through the API and structured log viewer.

## Capabilities

### New Capabilities
- `run-pipeline`: Multi-stage Temporal workflow (Plan → Execute → Verify) with retry loop, replacing the single-stage agent execution.
- `spec-generation`: Stage 1 agent that converts user input (prompt or detailed spec) into a well-formed OpenSpec change with machine-evaluatable acceptance criteria.
- `run-verification`: Stage 3 hybrid evaluation — automated checks from spec scenarios + LLM judge for semantic criteria. Returns structured pass/fail with explanation.

### Modified Capabilities
- `helm-chart`: Worker deployment needs the `openspec` CLI available in the sidecar image for `openspec validate --json` and `openspec list --json` calls.
- `container-images`: Sidecar image must include the `openspec` npm package for CLI access within agent pods.

## Impact

- `internal/temporal/workflow.go` — AgentRunWorkflow refactored into multi-stage pipeline with plan/execute/verify activities and retry loop.
- `internal/temporal/activities.go` — New PlanRun, ExecuteRun, VerifyRun activities that exec openspec CLI commands and manage agent invocations.
- `internal/sidecar/gateway.go` — Sidecar must support running the planning agent and verification agent in addition to the execution agent (sequential, not concurrent).
- `api/v1alpha1/types.go` — AgentRunStatus gains fields for current stage, verification result, retry count, and eval spec reference.
- `deploy/crds/agentrun-crd.yaml` — CRD schema updated with new status fields.
- `proto/aot/api/v1/api.proto` — Proto schema updated to expose stage, verification results, and eval spec through the API.
- `docker/Dockerfile.sidecar` — Install `openspec` CLI alongside pi-coding-agent.
- `internal/server/grpc.go` — GetAgentRun enriched with stage and verification data.
- `web/src/components/RunDetail.tsx` — UI shows current stage, verification results, retry history.
- `packages/pi-aot-extension/` — Agent harness may need OpenSpec skill awareness for /opsx:propose and /opsx:apply invocations.
