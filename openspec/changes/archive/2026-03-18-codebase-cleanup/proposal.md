## Why

A thorough codebase audit revealed dead code, unwired features, hardcoded config that should be configurable, and duplicate logic. Deprecated pod-based Temporal activities remain registered but never called. The spec-driven pipeline env vars are documented in Helm but hardcoded in Go. Duplicate `lookupPodName` and `LogsTab` implementations exist across files. An unused `ThemeToggle` component and `useKeyboardNavigation` hook add noise. Cleaning this up reduces confusion for new contributors and eliminates silent failures.

## What Changes

- **Remove deprecated Temporal activities**: `CreateAgentPod`, `CleanupPod`, `CollectLogs`, `CollectJuniorResults` and their supporting types/constants. These were replaced by the deployment-based architecture but never deleted.
- **Wire pipeline env vars**: Read `AOT_PIPELINE_MAX_RETRIES`, `AOT_PIPELINE_PLAN_TIMEOUT` from environment instead of hardcoded constants in `workflow_spec_driven.go`.
- **Deduplicate pod lookup**: Extract shared `lookupRunningPod` function used by both `FileHandler` and `ExecHandler` instead of duplicated methods.
- **Deduplicate LogsTab**: Extract `LogsTab` into its own component file, imported by both `RunDetail.tsx` and `DetailPane.tsx`.
- **Remove dead frontend code**: Delete `ThemeToggle.tsx` (unused component) and `useKeyboardNavigation.ts` (unused hook, superseded by `useKeyboard`).
- **Verify spec-driven activity registration**: Confirm `PlanRun` and `VerifyRun` methods on `*Activities` are discovered by the Temporal worker, add explicit test.

## Capabilities

### New Capabilities

None — this is a cleanup change.

### Modified Capabilities

None — no requirement-level behavior changes. All changes are implementation-level (removing dead code, deduplicating, wiring config).

## Impact

- `internal/temporal/activities.go` — Remove 4 deprecated functions + 6 supporting types + 3 constants
- `internal/temporal/workflow.go` — Remove deprecated activity constant references
- `internal/temporal/workflow_spec_driven.go` — Read env vars instead of hardcoded defaults
- `internal/server/files.go` — Extract shared `lookupRunningPod` function
- `internal/server/exec.go` — Use shared pod lookup
- `web/src/components/RunDetail.tsx` — Extract LogsTab to shared component
- `web/src/components/DetailPane.tsx` — Import shared LogsTab
- `web/src/components/ThemeToggle.tsx` — Delete
- `web/src/hooks/useKeyboardNavigation.ts` — Delete
