## 1. CRD + Proto + API Fields

- [x] 1.1 Add `project`, `feature`, `tags` fields to AgentRunSpec in `api/v1alpha1/types.go`
- [x] 1.2 Add `project`, `feature`, `tags` fields to proto `AgentRunSpec` in `api.proto` and regenerate
- [x] 1.3 Map new fields in `grpc.go` (specProtoToCRD, crdToProto)
- [x] 1.4 Map new fields in controller `mapping.go` (BuildWorkflowInput)
- [x] 1.5 Add `project_filter`, `feature_filter`, `tag_filter` to ListAgentRuns proto and implement label-selector filtering in `grpc.go`
- [x] 1.6 Add `ClassifyRun` RPC endpoint that takes prompt + repos and returns suggested project/feature/tags via LLM
- [x] 1.7 Add contract tests for new field mappings (proto↔CRD, CRD↔workflow)

## 2. Deterministic Auto-Assignment

- [x] 2.1 In `grpc.go` CreateAgentRun: auto-set `aot.uncworks.io/repo` label from repos[0].url (extract repo name)
- [x] 2.2 In controller: auto-set `aot.uncworks.io/feature` from OpenSpec change name for spec-driven runs
- [x] 2.3 Add unit tests for repo name extraction and label assignment

## 3. LLM Classification

- [x] 3.1 Create `internal/server/classify.go` with ClassifyRun handler that calls LiteLLM with prompt + existing projects/features
- [x] 3.2 Query existing projects/features by listing distinct label values from AgentRun CRDs
- [x] 3.3 Parse LLM JSON response into ClassifyRunResponse
- [x] 3.4 Add integration test for classification with mock LLM

## 4. Frontend — Run List Hierarchy

- [x] 4.1 Update tab order in RunDetailView: 1) Logs, 2) Traces, 3) Files, 4) Shell — remove Verify tab
- [x] 4.2 Add project selector to RunListView header (press `p` to open, filter by project label)
- [x] 4.3 Add feature-grouped view mode to RunListView (press `1` for features, `2` for all runs)
- [x] 4.4 Feature row component: shows name, aggregate status, attempt count, PR link
- [x] 4.5 Unassigned runs section below features

## 5. Frontend — New Run View

- [x] 5.1 Add project, feature, tags fields to NewRunView form
- [x] 5.2 Call ClassifyRun on prompt blur/debounce to get suggestions
- [x] 5.3 Pre-fill project/feature/tags from classification response
- [x] 5.4 Allow user to edit or clear suggestions before submitting

## 6. Frontend — Feature Detail View

- [x] 6.1 Create FeatureDetailView showing all runs for a feature, OpenSpec change link, aggregate status
- [x] 6.2 Add retry action (`r` key) that creates new run with same prompt + feature label + failure context
- [x] 6.3 Add route `/feature/:name` in AppNew.tsx

## 7. Post-Run Tag Enrichment

- [x] 7.1 Add `EnrichRunTags` Temporal activity that analyzes git diff and appends file-type and scope tags
- [x] 7.2 Call EnrichRunTags after successful verify/completion in both single and spec-driven workflows
- [x] 7.3 Add unit test for tag derivation from diff output
