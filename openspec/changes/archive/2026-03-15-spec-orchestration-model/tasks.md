## 1. Proto and CRD Schema Changes

- [x] 1.1 Add `OrchestrationMode` enum to `proto/aot/api/v1/api.proto` with values `ORCHESTRATION_MODE_UNSPECIFIED`, `ORCHESTRATION_MODE_SINGLE`, `ORCHESTRATION_MODE_AUTO`, `ORCHESTRATION_MODE_MANUAL`
- [x] 1.2 Add `OrchestrationTask` message to `api.proto` with fields: `string name`, `string prompt`, `repeated string repo_urls`
- [x] 1.3 Add `Orchestration` message to `api.proto` with field: `repeated OrchestrationTask tasks`
- [x] 1.4 Add fields to `AgentRunSpec` in `api.proto`: `string parent_run_id = 15`, `OrchestrationMode orchestration_mode = 16`, `Orchestration orchestration = 17`
- [x] 1.5 Add `repeated string children = 7` to `AgentRun` message in `api.proto` for child run names
- [x] 1.6 Add `string spec_run_id = 4` and `string parent_run_id = 5` filters to `ListAgentRunsRequest`
- [x] 1.7 Add `GetRunGraph` RPC to `AOTService` with `GetRunGraphRequest` (run ID) and `RunGraph` response (nodes + edges)
- [x] 1.8 Run `buf generate` to regenerate Go and TypeScript protobuf code
- [x] 1.9 Add matching Go types to `api/v1alpha1/types.go`: `OrchestrationMode` string type with constants, `OrchestrationTask` struct, `Orchestration` struct, and new fields on `AgentRunSpec` (`ParentRunID`, `OrchestrationMode`, `Orchestration`)
- [x] 1.10 Update `deploy/crds/agentrun-crd.yaml` and `deploy/helm/aot/crds/agentrun-crd.yaml` with new spec fields and validation (orchestrationMode enum, orchestration tasks max 7, task name regex `^[a-z0-9-]+$`)
- [x] 1.11 Run `controller-gen` to regenerate deepcopy functions for new types
- [x] 1.12 Add validation webhook or CEL rules: manual mode requires non-empty tasks, task names unique, max 7 tasks

## 2. Workflow Orchestration Logic

- [x] 2.1 Add `OrchestrationMode` and `Orchestration` fields to `WorkflowInput` in `internal/temporal/workflow.go`
- [x] 2.2 Add `ParentRunID` and `SpecRunID` fields to `WorkflowInput`
- [x] 2.3 Create orchestration preamble at the start of `AgentRunWorkflow`: check `OrchestrationMode` before step 1
- [x] 2.4 Implement auto-decomposition path: compose decomposition prompt, start senior agent, collect structured JSON output via `GetAgentStatus` extended to return output or new `CollectAgentOutput` activity
- [x] 2.5 Add JSON parsing for decomposition plan: `DecompositionPlan` struct with `Tasks []DecompositionTask` and `IntegrationPrompt string`; parse with fallback to single-run on error
- [x] 2.6 Implement junior fan-out: for each task in decomposition plan, call `SpawnJuniorWorkflow` with `Blocking=true` inside `workflow.Go` goroutines; collect all futures
- [x] 2.7 Update `SpawnJuniorWorkflow` to set `ParentRunID` and `SpecRunID` on child `WorkflowInput`
- [x] 2.8 Implement manual orchestration path: read `Orchestration.Tasks` from input, spawn juniors directly (no senior agent)
- [x] 2.9 Add `CollectJuniorResults` activity: queries each junior's workspace for `git diff HEAD~1` output, returns map of task name to diff string
- [x] 2.10 Implement senior integration step (auto mode only): compose review prompt with junior diffs, start senior agent, wait for completion
- [x] 2.11 Handle junior cancellation cascade: when senior receives cancel signal, cancel all running child workflows
- [x] 2.12 Add max-tasks enforcement: truncate decomposition plan to 7 tasks with warning log

## 3. Controller Changes

- [x] 3.1 In `startWorkflow`, propagate `ParentRunID`, `OrchestrationMode`, and `Orchestration` from CRD spec to `WorkflowInput`
- [x] 3.2 When `OrchestrationMode` is `auto` or `manual`, set label `aot.uncworks.io/spec-run-id` to the AgentRun's name and label `aot.uncworks.io/run-role` to `senior`
- [x] 3.3 When a junior AgentRun is created (detected by `ParentRunID` being non-empty), set label `aot.uncworks.io/spec-run-id` to `ParentRunID`, label `aot.uncworks.io/run-role` to `junior`, annotation `aot.uncworks.io/parent-run` to `ParentRunID`
- [x] 3.4 In `GetAgentRun` gRPC handler, query for AgentRuns with `parentRunID` matching the requested run's name; populate `children` field in response
- [x] 3.5 In `ListAgentRuns` gRPC handler, support `spec_run_id` filter via label selector and `parent_run_id` filter via field selector
- [x] 3.6 Implement `GetRunGraph` gRPC handler: query all runs by `spec-run-id` label, build tree from `parent_run_id` references, return nodes and edges

## 4. UI Changes

- [x] 4.1 Create `web/src/components/RunGraph.tsx` — tree visualization component accepting nodes (name, phase, role) and edges (parent-child)
- [x] 4.2 Style RunGraph nodes with phase-colored badges: green (Succeeded), red (Failed), blue (Running), gray (Pending), yellow (WaitingForInput), orange (Cancelled)
- [x] 4.3 Add RunGraph to `RunDetailPage`: fetch run graph when `children` is non-empty or `parentRunID` is set; display above event log
- [x] 4.4 Add progress summary line: "N/M tasks complete" calculated from junior phases
- [x] 4.5 Add breadcrumb navigation on junior detail pages: "Senior Run Name > Junior Run Name" with link back to senior
- [x] 4.6 Add click-to-navigate on RunGraph nodes: clicking a node navigates to `/runs/:id`
- [x] 4.7 Update `RunListPage` to visually group runs by `spec-run-id`: senior as primary row, juniors indented beneath with smaller font
- [x] 4.8 Add orchestration mode selector to `CreateRunForm`: radio buttons for Single / Auto / Manual
- [x] 4.9 When Manual is selected in CreateRunForm, show dynamic task list: "Add Task" button, each task has name + prompt fields
- [x] 4.10 Wire CreateRunForm orchestration fields to `CreateAgentRun` request

## 5. E2E Tests

- [x] 5.1 Add workflow unit test: `TestWorkflow_AutoDecomposition` — mock senior agent to return valid decomposition JSON, verify junior workflows are spawned with correct prompts
- [x] 5.2 Add workflow unit test: `TestWorkflow_AutoDecomposition_Fallback` — mock senior agent to return malformed JSON, verify single-run fallback
- [x] 5.3 Add workflow unit test: `TestWorkflow_ManualOrchestration` — provide orchestration tasks in input, verify junior workflows spawned without senior agent
- [x] 5.4 Add workflow unit test: `TestWorkflow_OrchestrationCancel` — cancel senior during junior execution, verify all juniors cancelled
- [x] 5.5 Add workflow unit test: `TestWorkflow_SingleMode` — verify `orchestrationMode=single` runs identically to current behavior
- [x] 5.6 Add controller unit test: verify labels and annotations are set correctly for senior and junior runs
- [x] 5.7 Add controller unit test: verify `ListAgentRuns` with `spec_run_id` filter returns correct runs
- [x] 5.8 Add E2E test: `TestE2E_AutoOrchestration` — create an AgentRun with `orchestrationMode=auto` and a multi-concern spec, verify junior runs are created, complete, and senior integrates
- [x] 5.9 Add E2E test: `TestE2E_ManualOrchestration` — create an AgentRun with `orchestrationMode=manual` and defined tasks, verify junior runs match the task list
- [x] 5.10 Add E2E test: `TestE2E_RunGraph` — verify `GetRunGraph` returns correct tree structure after orchestrated run completes

## 6. Verification

- [x] 6.1 Run `buf lint` and `buf breaking` — proto changes are valid and backward compatible
- [x] 6.2 Run `go test ./internal/temporal/...` — workflow tests pass
- [x] 6.3 Run `go test ./internal/controller/...` — controller tests pass
- [x] 6.4 Run `go test ./e2e/...` — E2E tests pass
- [x] 6.5 Run `npx tsc --noEmit -p web/tsconfig.json` — web UI compiles
- [x] 6.6 Manual smoke test: create an auto-orchestrated run, verify juniors spawn, complete, and graph displays in UI
