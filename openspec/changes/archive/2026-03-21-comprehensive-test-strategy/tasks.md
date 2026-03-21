## 1. Contract Tests (test/contract/)

- [x] 1.1 Create `boundary_proto_crd_test.go`: Test every AgentRunSpec proto field maps to CRD and back via specProtoToCRD/crdToProto (including autoPush, autoPR, pipelineConfig, prBaseBranch, displayName)
- [x] 1.2 Create `boundary_crd_workflow_test.go`: Test controller maps all CRD fields to WorkflowInput (including PipelineConfig, AutoPush, AutoPR, PRBaseBranch, OrchestrationMode, SpecContent)
- [x] 1.3 Create `boundary_workflow_activity_test.go`: Test PlanRunInput/VerifyRunInput/StartAgentInput/PushChangesInput/CreatePRInput contain all required fields when constructed by the workflow
- [x] 1.4 Create `boundary_span_types_test.go`: Define the canonical set of valid span types and test that every span creation path in gateway.go uses one of them
- [x] 1.5 Create `boundary_rest_types_test.go`: JSON-serialize server types (AgentLogEntry, TraceSpan, FileEntry, ThinkingResponse, VerificationResult) and verify field names match frontend expectations
- [x] 1.6 Create `boundary_nginx_routes_test.go`: Parse nginx config template from Helm chart and verify every mux.HandleFunc route in the API server is covered by a proxy rule

## 2. Integration Tests (test/integration/)

- [x] 2.1 Create `sidecar_spans_test.go`: Feed real pi JSONL through maybeCaptureStreamEvent and parseAgentJSONL, verify span count matches log entry count for tool calls
- [x] 2.2 Create `hydrator_workspace_test.go`: Run hydrator with single-repo and multi-repo configs against temp git repos, verify directory layout and resolveWorkDir returns correct paths
- [x] 2.3 Create `jsonl_parser_test.go`: Parse the same agent.jsonl with parseAgentJSONL and parseThinkingFromLines, verify no contradictions (thinking=true for completed messages)
- [x] 2.4 Create `loop_detection_test.go`: Feed N identical tool_use message_end events through extractToolCallSignature, verify the loop detection counter triggers at threshold
- [x] 2.5 Create `workspace_resolution_test.go`: Create temp directory structures for each layout (root .git, subdir .git, legacy /src/.git, explicit path) and verify resolveWorkDir returns correct path

## 3. Smoke Tests (e2e/)

- [x] 3.1 Create `smoke_pipeline_test.go`: Create spec-driven run, poll until SUCCEEDED or FAILED, assert SUCCEEDED within 15 min
- [x] 3.2 Create `smoke_files_test.go`: Create run, poll until RUNNING, verify GET /files returns entries, verify .aot and .bare are filtered
- [x] 3.3 Create `smoke_shell_test.go`: Create run, poll until RUNNING, send WebSocket upgrade to /exec, assert 101 response
- [x] 3.4 Create `smoke_traces_test.go`: Wait for run completion, compare tool_call count from /logs/structured with tool span count from /traces
- [x] 3.5 Create `smoke_git_test.go`: Create run with autoPush=true, verify branch exists after completion (requires GITHUB_TOKEN and test repo)
- [x] 3.6 Create `smoke_validation_test.go`: Verify that a completed spec-driven run has valid OpenSpec specs (openspec validate passes)

## 4. Fix Stale Tests

- [x] 4.1 Update hydrator_test.go: Fix workspace path assertions from `/workspace/src/<repo>` to `/workspace/<repo>`
- [x] 4.2 Update any test referencing `/workspace/src/` paths to use new layout
- [x] 4.3 Verify all existing tests pass with `go test ./...` after changes

## 5. CI Integration

- [x] 5.1 Add contract + integration test targets to Taskfile.yml (task test:contract, task test:integration)
- [x] 5.2 Document test layers and how to run each in test/README.md
