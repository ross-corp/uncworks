## Architecture

Three test layers, each runnable independently:

```
Layer 1: Contract Tests (go test ./test/contract/...)
  ├── boundary_proto_crd_test.go      # Proto ↔ CRD field mapping
  ├── boundary_crd_workflow_test.go   # CRD → WorkflowInput completeness
  ├── boundary_workflow_activity_test.go  # Workflow → Activity input fields
  ├── boundary_span_types_test.go     # Sidecar span types match frontend enum
  ├── boundary_rest_types_test.go     # Server REST JSON ↔ Frontend TS types
  └── boundary_nginx_routes_test.go   # Nginx proxy covers all registered routes

Layer 2: Integration Tests (go test ./test/integration/...)
  ├── sidecar_spans_test.go          # Pi JSONL → spans match activity entries
  ├── hydrator_workspace_test.go     # Workspace layout (single/multi repo)
  ├── jsonl_parser_test.go           # Structured log + thinking parsers agree
  ├── loop_detection_test.go         # Loop detector kills after N repeats
  └── workspace_resolution_test.go   # resolveWorkDir covers all layouts

Layer 3: Smoke Tests (go test -tags e2e ./e2e/...)
  ├── smoke_pipeline_test.go         # Full Plan→Execute→Verify
  ├── smoke_files_test.go            # File explorer during hydration + run
  ├── smoke_shell_test.go            # WebSocket 101 upgrade + TTY
  ├── smoke_traces_test.go           # Trace span count matches activity
  ├── smoke_git_test.go              # autoPush branch + autoPR creation
  └── smoke_validation_test.go       # OpenSpec validate catches bad specs
```

## Workflow

### Contract Tests — No Infrastructure

Each test instantiates structs on both sides of a boundary and verifies all fields map:

```go
// Example: Proto ↔ CRD mapping
func TestBoundary_ProtoToCRD_AllFieldsMapped(t *testing.T) {
    // Create proto with ALL fields set
    proto := &apiv1.AgentRunSpec{
        Prompt: "test", AutoPush: true, AutoPr: true,
        PipelineConfig: &apiv1.PipelineConfig{...},
    }
    // Convert
    crd := specProtoToCRD(proto)
    // Assert every proto field appears in CRD
    assert.Equal(t, proto.AutoPush, crd.AutoPush)
    assert.Equal(t, proto.AutoPr, crd.AutoPR)
    assert.NotNil(t, crd.PipelineConfig)
}
```

### Integration Tests — Local Process

Tests that start real processes but don't need k8s:

```go
// Example: Hydrator workspace layout
func TestIntegration_Hydrator_SingleRepo_Layout(t *testing.T) {
    h := NewHydrator(Config{
        WorkspaceDir: t.TempDir(),
        Repos: []Repository{{URL: "https://github.com/test/repo"}},
    })
    h.Run(ctx)
    // Verify repo is at workspace/<reponame>/
    assert.DirExists(t, filepath.Join(h.WorktreePath(), ".git"))
}
```

### Smoke Tests — k8s Cluster

Build on existing e2e harness. Each test creates a run and polls for specific behavior:

```go
// Example: Trace completeness
func TestSmoke_Traces_MatchActivity(t *testing.T) {
    run := createAndWaitForRun(t, "simple prompt")
    logs := getStructuredLogs(t, run.ID)
    traces := getTraces(t, run.ID)
    toolCalls := countByType(logs, "tool_call")
    toolSpans := countByType(traces, "tool")
    assert.Equal(t, toolCalls, toolSpans, "every tool call should have a trace span")
}
```

## Integration

- Contract tests: `go test ./test/contract/...` — runs in CI on every commit
- Integration tests: `go test ./test/integration/...` — runs in CI, may need git repos
- Smoke tests: `go test -tags e2e ./e2e/...` — runs post-deploy against live cluster
- All use standard Go testing — no new frameworks
