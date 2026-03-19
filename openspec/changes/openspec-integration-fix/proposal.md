## Why

The spec-driven pipeline claims to use OpenSpec for plan validation, task tracking, and verification — but an audit reveals the integration is mostly fake. PlanRun returns `SpecsValid: true` without calling any OpenSpec CLI commands. The verify gates run `openspec list` and `openspec validate` but swallow all errors with `2>/dev/null` and hardcode validation to pass on failure. The archive gate uses `|| true` so archive errors are invisible. The system works by accident (the agent happens to do the right thing) rather than by design (Temporal verifies each step programmatically). This needs to be fixed before production customers rely on it.

## What Changes

- **PlanRun**: After the planning agent completes, run `openspec validate --json` and `openspec status --change <id> --json` via ExecCommand. Only return `SpecsValid: true` if both pass. If validation fails, return the errors so the workflow can retry or fail.
- **Pre-execute check**: Before starting the execute agent, verify the OpenSpec change directory exists and contains the expected artifacts (proposal.md, at least one spec file, tasks.md).
- **Verify error handling**: Remove all `2>/dev/null` redirects. Parse JSON responses with proper error handling. If `openspec list` or `openspec validate` returns invalid JSON or errors, fail the gate with a clear error message.
- **Verify validation gate**: Actually check `valid: true` in the response. Don't hardcode `result.ValidationValid = true` after the check.
- **Archive gate**: Remove `|| true`. If `openspec archive` fails, include the error in the verification result instead of silently ignoring it.
- **OpenSpec init in workspace**: Before the planning agent runs, execute `openspec init` in the workspace if `.openspec.yaml` doesn't exist, ensuring the OpenSpec CLI has a valid project context.
- **Structured logging**: Log the actual openspec command output (stdout/stderr) so verification results are debuggable.

## Capabilities

### New Capabilities

None — this fixes existing capabilities that were implemented incorrectly.

### Modified Capabilities
- `run-verification`: Fix error handling, remove hardcoded passes, fail on real errors.
- `run-pipeline`: Add real plan validation and pre-execute artifact checks.

## Impact

- `internal/temporal/activities_spec_driven.go` — Fix PlanRun (add validation), fix VerifyRun (remove error swallowing, fix validation gate, fix archive gate)
- `internal/temporal/workflow_spec_driven.go` — Add pre-execute artifact check between plan and execute stages
- Tests updated to verify real error paths
