## 1. Add Backend and SpecSource to WorkflowInput

- [x] 1.1 Add `Backend string` and `SpecSource string` fields to `WorkflowInput` in `internal/temporal/workflow.go`
- [x] 1.2 Map `Backend: string(agentRun.Spec.Backend)` and `SpecSource: agentRun.Spec.SpecSource` in `BuildWorkflowInput` in `internal/controller/mapping.go`

## 2. Align server TraceSpan time types

- [x] 2.1 Add RFC3339 documentation comments to `StartTime` and `EndTime` in `server.TraceSpan` in `internal/server/traces.go` (kept as string — sidecar writes time.Time which JSON-marshals to RFC3339 strings, server reads/passes as strings)
- [x] 2.2 No code changes needed — documentation comment added above

## 3. Add SpanId to AgentLogEntry

- [x] 3.1 Add `SpanId string \`json:"spanId,omitempty"\`` to `AgentLogEntry` in `internal/server/files.go`

## 4. Update contract tests

- [x] 4.1 Add `TestBoundary_CRDWorkflow_BackendMapped` to `test/contract/boundary_crd_workflow_test.go` — verify BuildWorkflowInput copies Backend for each BackendType (Pod, KubeVirt, External)
- [x] 4.2 Add `TestBoundary_CRDWorkflow_SpecSourceMapped` to `test/contract/boundary_crd_workflow_test.go` — verify BuildWorkflowInput copies SpecSource when present and empty string when absent
- [x] 4.3 TraceSpan test unchanged — types remain string (correct for server's role as pass-through), RFC3339 comment added to struct
- [x] 4.4 Add `TestBoundary_RESTTypes_AgentLogEntrySpanId` to `test/contract/boundary_rest_types_test.go` — verify spanId appears in JSON when set and is omitted when empty

## 5. Verification

- [x] 5.1 Run `go build ./...` — no compilation errors
- [x] 5.2 Run `go test ./test/contract/...` — all contract tests pass
- [x] 5.3 Run `go test ./internal/...` — no regressions
