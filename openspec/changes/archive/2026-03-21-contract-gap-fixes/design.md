## Context

The AOT platform maps data through three layers: CRD (K8s) -> WorkflowInput (Temporal) -> Proto/REST (API/frontend). An audit found four places where fields are defined in one layer but silently dropped or mistyped during mapping to the next. These are small, additive fixes with no breaking changes.

## Goals / Non-Goals

**Goals:**
- WorkflowInput includes Backend and SpecSource from the CRD so workflow logic can use them
- TraceSpan time fields use consistent types across sidecar and server packages
- AgentLogEntry includes optional spanId so the frontend can link log entries to trace spans
- Contract tests cover every gap to prevent regressions

**Non-Goals:**
- Adding new CRD fields (all fields already exist in the CRD schema)
- Changing proto definitions (SpecSource and Backend are already in the proto)
- Modifying frontend code (it already handles spanId when present)
- Changing the sidecar TraceSpan type (it already uses time.Time correctly)

## Decisions

### 1. Add Backend and SpecSource to WorkflowInput

Add two fields to the `WorkflowInput` struct in `internal/temporal/workflow.go`:

```go
Backend    string // "Pod", "KubeVirt", "External"
SpecSource string // "editor", "github:owner/repo/path", etc.
```

Map them in `BuildWorkflowInput` in `internal/controller/mapping.go`:

```go
Backend:    string(agentRun.Spec.Backend),
SpecSource: agentRun.Spec.SpecSource,
```

Backend is stored as a string (not `BackendType`) because the temporal package should not import the CRD types package. The string conversion is explicit at the boundary.

### 2. Align server TraceSpan time fields to time.Time

Change `internal/server/traces.go`:

```go
// Before
StartTime string `json:"startTime"`
EndTime   string `json:"endTime"`

// After
StartTime time.Time `json:"startTime"`
EndTime   time.Time `json:"endTime"`
```

`time.Time` marshals to RFC 3339 JSON by default, which is the same format the frontend expects. This eliminates the type mismatch with `sidecar.TraceSpan` and removes the need for string-to-time conversion when spans cross package boundaries.

**Callers affected:** The `handleGetSpans` handler in `traces.go` constructs `server.TraceSpan` from `sidecar.TraceSpan`. Currently it copies `StartTime` as a string; after this change it copies `time.Time` directly. The contract test in `boundary_rest_types_test.go` that constructs spans with string times needs updating.

### 3. Add SpanId to AgentLogEntry

Add to `internal/server/files.go`:

```go
type AgentLogEntry struct {
    // existing fields...
    SpanId string `json:"spanId,omitempty"`
}
```

The field is populated from the agent's JSONL log output. The agent already writes `spanId` to its log entries when available. The backend struct just needs the field to stop dropping it during deserialization.

### 4. Contract test strategy

Each gap gets a WHEN/THEN scenario verified by a Go test:

- **Backend mapping:** `TestBoundary_CRDWorkflow_BackendMapped` — build WorkflowInput from CRD with Backend=KubeVirt, assert input.Backend == "KubeVirt"
- **SpecSource mapping:** `TestBoundary_CRDWorkflow_SpecSourceMapped` — build WorkflowInput from CRD with SpecSource set, assert it carries through
- **Time type consistency:** `TestBoundary_RESTTypes_TraceSpanTimeType` — verify server.TraceSpan.StartTime is time.Time and serializes to RFC 3339
- **SpanId presence:** `TestBoundary_RESTTypes_AgentLogEntrySpanId` — marshal AgentLogEntry with SpanId set, verify JSON contains `"spanId"`

## Risks / Trade-offs

- **[Risk] Changing StartTime/EndTime type breaks callers** — Low risk. Only `traces.go` constructs `server.TraceSpan` and the test file uses it. Both are updated in this change. The JSON wire format is unchanged (RFC 3339 string).
- **[Risk] Backend as string instead of typed enum in WorkflowInput** — Acceptable trade-off. Keeps the temporal package decoupled from the CRD types package. The string is validated at the CRD level before it reaches the workflow.
