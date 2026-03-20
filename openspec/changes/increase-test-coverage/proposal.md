## Why

Three critical packages have very low test coverage: `internal/server` (21.9%), `internal/sidecar` (10.2%), `internal/temporal` (9.1%). These are the core execution components — the server serves run data to the UI, the sidecar manages agent execution, and the temporal package orchestrates the entire run pipeline. Bugs in these packages cause silent failures that are hard to diagnose. The most important code paths have zero tests.

## What Changes

- **server tests** — cover `parseAgentJSONL` dedup logic, `isHiddenDir`, `parseLsOutput` for file listing
- **sidecar tests** — cover `ExecCommand` workdir resolution, `extractToolCallSignature` loop detection
- **temporal tests** — cover `PlanRun` openspec scaffolding/validation, `VerifyRun` 5-gate logic

## Capabilities

### New Capabilities
- `test-coverage`: Unit tests for critical server, sidecar, and temporal code paths.

### Modified Capabilities

None.

## Impact

- `internal/server/files_test.go` — new or expanded test file
- `internal/sidecar/gateway_test.go` — new or expanded test file
- `internal/temporal/activities_spec_driven_test.go` — new or expanded test file
