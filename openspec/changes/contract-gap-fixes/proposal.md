## Why

The platform audit revealed four data contract gaps where fields are defined in the CRD but silently dropped during mapping, or where type mismatches between components cause subtle bugs.

1. **Backend field not mapped to WorkflowInput.** The CRD has `spec.backend` (Pod/KubeVirt/External) but `BuildWorkflowInput` in `internal/controller/mapping.go` never copies it. The Temporal workflow has no way to know what backend was requested. Today this is invisible because the controller uses the field directly, but any workflow-side logic that needs to vary by backend (logging, timeout defaults, resource labels) cannot work.

2. **SpecSource not mapped to WorkflowInput.** The CRD has `spec.specSource` (e.g., "editor", "github:owner/repo/path") and `crdToProto` already maps it to the proto response, but `BuildWorkflowInput` skips it. The workflow cannot log or record where the spec originated. Traceability is lost for audit purposes.

3. **TraceSpan time type mismatch.** The sidecar defines `StartTime`/`EndTime` as `time.Time` (`internal/sidecar/gateway.go`), while the server defines them as `string` (`internal/server/traces.go`). Both serialize to JSON correctly, but any Go code that passes spans between packages requires manual conversion. The server type should use `time.Time` to match the sidecar and eliminate implicit string formatting.

4. **AgentLogEntry missing spanId field.** The frontend's `ActivityFeed.tsx` defines `spanId?: string` on `LogEntry` and renders a `SpanBadge` when present. The backend struct `AgentLogEntry` in `internal/server/files.go` has no `SpanId` field, so the JSON response never includes it. The frontend badge never renders.

## What Changes

- Add `Backend` and `SpecSource` fields to `WorkflowInput` struct and map them in `BuildWorkflowInput`.
- Change `server.TraceSpan.StartTime`/`EndTime` from `string` to `time.Time` and update all callers.
- Add `SpanId string` field to `server.AgentLogEntry` with `json:"spanId,omitempty"` tag.
- Update existing contract tests to cover the new mappings and type alignment.

## Capabilities

### Modified Capabilities
- `contract-tests`: Existing boundary tests updated to verify Backend/SpecSource mapping, time type consistency, and spanId presence.

## Impact

- `internal/temporal/workflow.go` — add Backend, SpecSource fields to WorkflowInput
- `internal/controller/mapping.go` — map Backend, SpecSource in BuildWorkflowInput
- `internal/server/traces.go` — change StartTime/EndTime to time.Time
- `internal/server/files.go` — add SpanId to AgentLogEntry
- `test/contract/boundary_crd_workflow_test.go` — add assertions for Backend, SpecSource
- `test/contract/boundary_rest_types_test.go` — update TraceSpan tests for time.Time, add spanId test
- `test/contract/boundary_span_types_test.go` — update sidecar span time type tests if needed
