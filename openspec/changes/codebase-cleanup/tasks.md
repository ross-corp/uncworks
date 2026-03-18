## 1. Remove Deprecated Temporal Activities

- [x] 1.1 Delete `CreateAgentPod` function and `CreateAgentPodInput`/`CreateAgentPodOutput` types from `internal/temporal/activities.go`
- [x] 1.2 Delete `CleanupPod` function and `CleanupPodInput` type from `internal/temporal/activities.go`
- [x] 1.3 Delete `CollectLogs` function and `CollectLogsInput`/`CollectLogsOutput` types from `internal/temporal/activities.go`
- [x] 1.4 Delete `CollectJuniorResults` function and `CollectJuniorResultsInput`/`CollectJuniorResultsOutput` types from `internal/temporal/activities.go` and constant from `workflow.go`
- [x] 1.5 Remove deprecated activity constants (`ActivityCreateAgentPod`, `ActivityCleanupPod`, `ActivityCollectLogs`) from `internal/temporal/workflow.go`
- [x] 1.6 Verify build passes and no references remain (`go build ./...`, `grep` for removed names)

## 2. Wire Pipeline Environment Variables

- [x] 2.1 Add `envOrDefaultInt` and `envOrDefaultDuration` helpers to `internal/temporal/workflow_spec_driven.go`
- [x] 2.2 Replace hardcoded `defaultMaxRetries = 3` with `envOrDefaultInt("AOT_PIPELINE_MAX_RETRIES", 3)`
- [x] 2.3 Replace hardcoded `defaultPlanTimeout = 2 * time.Minute` with `envOrDefaultDuration("AOT_PIPELINE_PLAN_TIMEOUT", "2m")`
- [x] 2.4 Write test verifying env var override works

## 3. Deduplicate Pod Lookup

- [x] 3.1 Extract `lookupRunningPod(ctx, k8sClient, namespace, runID)` as a package-level function in `internal/server/`
- [x] 3.2 Update `FileHandler.lookupPodName` to call `lookupRunningPod`
- [x] 3.3 Update `ExecHandler.lookupPodName` to call `lookupRunningPod`
- [x] 3.4 Verify both handlers still work (run existing tests)

## 4. Deduplicate LogsTab Component

- [x] 4.1 Extract `LogsTab` from `RunDetail.tsx` into `web/src/components/LogsTab.tsx`
- [x] 4.2 Update `RunDetail.tsx` to import `LogsTab` from the new file
- [x] 4.3 Update `DetailPane.tsx` to import `LogsTab` from the new file, removing its inline copy
- [x] 4.4 Verify TypeScript compiles and both views render correctly

## 5. Remove Dead Frontend Code

- [x] 5.1 Delete `web/src/components/ThemeToggle.tsx`
- [x] 5.2 Delete `web/src/hooks/useKeyboardNavigation.ts`
- [x] 5.3 Verify no imports reference the deleted files (`grep`)
- [x] 5.4 Verify TypeScript compiles

## 6. Verify Spec-Driven Activity Registration

- [x] 6.1 Add test that verifies `PlanRun` and `VerifyRun` are discoverable as methods on `*Activities` via reflection
- [x] 6.2 Verify `go test ./internal/temporal/` passes

## 7. Formalize Roadmap Process

- [x] 7.1 Create `ROADMAP.md` in project root with sections: Current (in-progress), Next (planned), Future (backlog)
- [x] 7.2 Add knowledge system (brain, embeddings, SearchPastWork) to roadmap as "Next" item
- [x] 7.3 Add spec-driven pipeline improvements (direct exec in sidecar, verdict file streaming) to roadmap as "Future" item
- [x] 7.4 Document the roadmap process in ROADMAP.md: how items are added, prioritized, and graduated to OpenSpec changes
