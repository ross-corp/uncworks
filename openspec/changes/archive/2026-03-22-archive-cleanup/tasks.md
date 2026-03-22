## 1. Add RetentionDays field and env var parsing

- [x] 1.1 Add `RetentionDays` field to `AgentRunReconciler` struct
- [x] 1.2 Read `AOT_RETENTION_DAYS` env var in `cmd/controller/main.go`, default to 7
- [x] 1.3 Pass retention days to reconciler on construction

## 2. Implement cleanupExpiredRuns method

- [x] 2.1 Add `cleanupExpiredRuns(ctx)` method that lists all AgentRuns, filters terminal+expired, deletes Deployment+PVC, and annotates as archived
- [x] 2.2 Skip runs already annotated with `aot.uncworks.io/archived: "true"`
- [x] 2.3 Tolerate NotFound errors when Deployment or PVC is already deleted

## 3. Wire cleanup loop into controller manager

- [x] 3.1 Implement `Start(ctx)` method on a cleanup runnable
- [x] 3.2 Register the runnable with the manager via `mgr.Add()`

## 4. Update Helm chart

- [x] 4.1 Add `AOT_RETENTION_DAYS` env var to controller deployment template
- [x] 4.2 Add `controller.retentionDays` to values.yaml with default 7

## 5. Add unit tests

- [x] 5.1 Test that expired runs get Deployment+PVC deleted and archived annotation set
- [x] 5.2 Test that non-expired runs are not cleaned up
- [x] 5.3 Test that already-archived runs are skipped

## 6. Build and test verification

- [x] 6.1 Run `go build ./cmd/... ./internal/...` — compiles cleanly
- [x] 6.2 Run `go test ./internal/controller/... -count=1` — all tests pass
