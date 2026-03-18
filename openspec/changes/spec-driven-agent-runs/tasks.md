## 1. Data Model & CRD

- [ ] 1.1 Add `orchestrationMode: "spec-driven"` to CRD enum and Go types (`api/v1alpha1/types.go`, `deploy/crds/agentrun-crd.yaml`)
- [ ] 1.2 Add `Stage` field to `AgentRunStatus` (enum: `planning`, `executing`, `verifying`, empty for non-spec-driven)
- [ ] 1.3 Add `RetryCount` int32 field to `AgentRunStatus`
- [ ] 1.4 Add `VerificationResult` string field to `AgentRunStatus` (JSON-encoded verdict)
- [ ] 1.5 Update proto schema (`proto/aot/api/v1/api.proto`) with stage, retry_count, verification_result fields
- [ ] 1.6 Regenerate Go and TypeScript proto code (`task proto:gen`)

## 2. Sidecar: Stage-Aware Agent Invocation

- [ ] 2.1 Add `stage` field to `StartAgentRequest` proto in `proto/aot/agent/v1/agent.proto`
- [ ] 2.2 Update sidecar `StartAgent` handler to configure pi-coding-agent differently per stage (system prompt, tool restrictions)
- [ ] 2.3 Install `openspec` CLI in sidecar Docker image (`docker/Dockerfile.sidecar`: `npm install -g openspec`)
- [ ] 2.4 Plan stage: configure agent with `/opsx:propose` system prompt, read-only + openspec tools
- [ ] 2.5 Execute stage: configure agent with `/opsx:apply` system prompt, full tool set
- [ ] 2.6 Verify stage: configure agent with evaluation system prompt, read-only tools + exec for test commands
- [ ] 2.7 Write tests for stage-specific agent configuration

## 3. Temporal Workflow: Multi-Stage Pipeline

- [ ] 3.1 Add `PlanRun` activity: invoke sidecar StartAgent with stage=plan, wait for completion, run `openspec validate --json`, return change path
- [ ] 3.2 Add `VerifyRun` activity: run automated checks (openspec list --json for task completion, exec spec commands), invoke LLM judge if automated pass, return structured verdict
- [ ] 3.3 Refactor `AgentRunWorkflow` to detect `spec-driven` orchestration mode and route to `runSpecDrivenPipeline`
- [ ] 3.4 Implement `runSpecDrivenPipeline`: Plan → Execute → Verify loop with retry (max 3, configurable)
- [ ] 3.5 Implement retry context injection: on verify failure, prepend failure report to execute agent prompt
- [ ] 3.6 Update workflow state query to include stage and retry count
- [ ] 3.7 On success, run `openspec archive` via exec in pod
- [ ] 3.8 Write Temporal workflow unit tests for the spec-driven pipeline (mock activities)

## 4. Verification Activity Implementation

- [ ] 4.1 Implement automated file-existence checks: parse spec scenarios for file path references, check `os.Stat`
- [ ] 4.2 Implement automated command checks: parse spec scenarios for command references, exec in workspace, check exit code
- [ ] 4.3 Implement OpenSpec task completion check: `openspec list --json`, compare completed vs total
- [ ] 4.4 Implement LLM judge: build prompt from spec WHEN/THEN + git diff + agent log, invoke via LiteLLM, parse structured verdict
- [ ] 4.5 Implement structured verdict output: write `verification-result.json` to workspace
- [ ] 4.6 Write tests for each verification check type (file, command, task completion, LLM judge)

## 5. API & Proto Updates

- [ ] 5.1 Update `GetAgentRun` to include stage, retry_count, and verification_result in response
- [ ] 5.2 Update `ListAgentRuns` to allow filtering by stage
- [ ] 5.3 Add `GetVerificationResult` REST endpoint: `GET /api/v1/runs/{id}/verification` returns the structured verdict JSON
- [ ] 5.4 Update `crdToProto` mapping for new status fields
- [ ] 5.5 Write contract tests for new API fields

## 6. Web UI

- [ ] 6.1 Update run list to show current stage badge (Planning / Executing / Verifying) alongside phase
- [ ] 6.2 Update run detail info tab to display stage, retry count, and verification summary
- [ ] 6.3 Add verification result panel in detail view: shows per-criterion pass/fail with expandable details
- [ ] 6.4 Update structured log viewer to show stage transitions as system events
- [ ] 6.5 Show retry history: which attempt, what failed, what was retried
- [ ] 6.6 Update `MODEL_TIER_OPTIONS` to note which models support spec-driven mode

## 7. Configuration & Deployment

- [ ] 7.1 Add pipeline configuration to Helm values: `pipeline.maxRetries`, `pipeline.planTimeout`, `pipeline.verifyModel`
- [ ] 7.2 Update worker Helm template to pass pipeline env vars
- [ ] 7.3 Update `.env.example` with pipeline configuration documentation
- [ ] 7.4 Update `deploy/crds/agentrun-crd.yaml` with new status fields and orchestration mode enum value

## 8. Testing & Validation

- [ ] 8.1 E2E test: create spec-driven run with prompt, verify it plans, executes, and verifies
- [ ] 8.2 E2E test: create spec-driven run that should fail verification, verify retry and eventual failure
- [ ] 8.3 E2E test: create run with specContent, verify auto-upgrade to spec-driven mode
- [ ] 8.4 E2E test: single-mode run still works unchanged
- [ ] 8.5 Playwright test: verify stage badges and verification results in UI
- [ ] 8.6 Integration test: verify `openspec validate --json` and `openspec list --json` work inside sidecar container
