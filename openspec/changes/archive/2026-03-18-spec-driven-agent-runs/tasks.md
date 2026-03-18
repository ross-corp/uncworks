## 1. Data Model & CRD

- [x] 1.1 Add `orchestrationMode: "spec-driven"` to CRD enum and Go types (`api/v1alpha1/types.go`, `deploy/crds/agentrun-crd.yaml`)
- [x] 1.2 Add `Stage` field to `AgentRunStatus` (enum: `planning`, `executing`, `verifying`, empty for non-spec-driven)
- [x] 1.3 Add `RetryCount` int32 field to `AgentRunStatus`
- [x] 1.4 Add `VerificationResult` string field to `AgentRunStatus` (JSON-encoded verdict)
- [x] 1.5 Update proto schema (`proto/aot/api/v1/api.proto`) with stage, retry_count, verification_result fields
- [x] 1.6 Regenerate Go and TypeScript proto code (`task proto:gen`)

## 2. Sidecar: Stage-Aware Agent Invocation

- [x] 2.1 Add `stage` field to `StartAgentRequest` proto in `proto/aot/agent/v1/agent.proto`
- [x] 2.2 Update sidecar `StartAgent` handler to configure pi-coding-agent differently per stage (system prompt, tool restrictions)
- [x] 2.3 Install `openspec` CLI in sidecar Docker image (`docker/Dockerfile.sidecar`: `npm install -g openspec`)
- [x] 2.4 Plan stage: system prompt instructs OpenSpec change creation (`openspec new change`, then generate proposal/specs/tasks)
- [x] 2.5 Execute stage: system prompt instructs `/opsx:apply` implementation workflow, full tool set
- [x] 2.6 Verify stage: system prompt instructs evaluation against spec, read-only tools + exec for test commands
- [x] 2.7 Write tests for stage-specific agent configuration

## 3. Temporal Workflow: Multi-Stage Pipeline

- [x] 3.1 Add `PlanRun` activity: invoke sidecar StartAgent with stage=plan, wait for completion, run `openspec validate --json` and `openspec status --change <id> --json` to confirm artifacts complete
- [x] 3.2 Add `VerifyRun` activity: run verification pipeline (openspec list → openspec validate → automated checks → LLM judge → openspec archive)
- [x] 3.3 Refactor `AgentRunWorkflow` to detect `spec-driven` orchestration mode and route to `runSpecDrivenPipeline`
- [x] 3.4 Implement `runSpecDrivenPipeline`: Plan → Execute → Verify loop with retry (max 3, configurable via env var)
- [x] 3.5 Implement retry context injection: on verify failure, prepend structured failure report to execute agent prompt
- [x] 3.6 Update workflow state query to include stage and retry count
- [x] 3.7 Write Temporal workflow unit tests for the spec-driven pipeline (mock activities)

## 4. Verification Activity: OpenSpec CLI Integration

- [x] 4.1 Implement task completion gate: exec `openspec list --json` in pod, parse JSON, check `completedTasks == totalTasks`
- [x] 4.2 Implement structural validation gate: exec `openspec validate --json` in pod, check `valid: true`
- [x] 4.3 Implement automated scenario checks: parse spec WHEN/THEN for command references, exec in workspace, check exit codes and output
- [x] 4.4 Implement file existence checks: parse spec WHEN/THEN for file path references, check `os.Stat` in workspace
- [x] 4.5 Implement LLM judge: build prompt from spec WHEN/THEN + git diff + agent log, invoke via LiteLLM, parse structured per-scenario verdict
- [x] 4.6 Implement archive gate: on all-pass, exec `openspec archive --yes` to seal the change
- [x] 4.7 Implement structured verdict output: write `verification-result.json` to change directory
- [x] 4.8 Write tests for each verification gate (task completion, validation, automated checks, LLM judge, archive)

## 5. API & Proto Updates

- [x] 5.1 Update `GetAgentRun` to include stage, retry_count, and verification_result in response
- [x] 5.2 Update `ListAgentRuns` to allow filtering by stage
- [x] 5.3 Add `GetVerificationResult` REST endpoint: `GET /api/v1/runs/{id}/verification` returns the structured verdict JSON from workspace
- [x] 5.4 Update `crdToProto` and `specProtoToCRD` mappings for new status fields and orchestration mode
- [x] 5.5 Write contract tests for new API fields

## 6. Web UI

- [x] 6.1 Update run list to show current stage badge (Planning / Executing / Verifying) alongside phase
- [x] 6.2 Update run detail info tab to display stage, retry count, and verification summary
- [x] 6.3 Add verification result panel in detail view: shows per-gate pass/fail with expandable details (task completion, validation, automated checks, LLM verdict)
- [x] 6.4 Update structured log viewer to show stage transitions as system events
- [x] 6.5 Show retry history: which attempt, what failed, what was retried
- [x] 6.6 Add `spec-driven` option to orchestration mode selector in create form

## 7. Configuration & Deployment

- [x] 7.1 Add pipeline configuration to Helm values: `pipeline.maxRetries`, `pipeline.planTimeout`, `pipeline.verifyModel`
- [x] 7.2 Update worker Helm template to pass pipeline env vars (`AOT_PIPELINE_MAX_RETRIES`, `AOT_PIPELINE_PLAN_TIMEOUT`)
- [x] 7.3 Update `.env.example` with pipeline configuration documentation
- [x] 7.4 Update `deploy/crds/agentrun-crd.yaml` with new status fields and `spec-driven` orchestration mode enum

## 8. Testing & Validation

- [x] 8.1 E2E test: create spec-driven run with prompt, verify it plans (openspec status shows artifacts complete), executes, and verifies (openspec archive succeeds)
- [x] 8.2 E2E test: create spec-driven run that should fail verification (incomplete tasks), verify retry and eventual failure
- [x] 8.3 E2E test: create run with specContent, verify auto-upgrade to spec-driven mode
- [x] 8.4 E2E test: single-mode run still works unchanged (backward compat)
- [x] 8.5 Playwright test: verify stage badges and verification results in UI
- [x] 8.6 Integration test: verify `openspec validate --json`, `openspec list --json`, and `openspec archive` work correctly inside sidecar container
- [x] 8.7 Unit test: verification gate pipeline (mock each gate, test short-circuit behavior)
