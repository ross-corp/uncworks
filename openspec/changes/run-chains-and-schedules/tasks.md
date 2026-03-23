## 1. RunTemplate CRD + Go Types

- [ ] 1.1 Create `api/v1alpha1/runtemplate_types.go` with RunTemplateSpec (prompt, repos, modelTier, manageModelTier, implementModelTier, orchestrationMode, autoPush, autoPR, prBaseBranch, projectRef, specRef, ttlSeconds, image, envVars, devboxConfig, maxBudget, tags), RunTemplateStatus (phase, conditions), RunTemplate, RunTemplateList — follow project_types.go patterns
- [ ] 1.2 Add kubebuilder markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, printcolumns for Name, Project, Phase, Age
- [ ] 1.3 Add `init()` function to register RunTemplate and RunTemplateList with SchemeBuilder
- [ ] 1.4 Run `make generate` to produce deepcopy functions in `zz_generated.deepcopy.go`
- [ ] 1.5 Run `make manifests` to generate CRD YAML in `deploy/crds/aot.uncworks.io_runtemplates.yaml`
- [ ] 1.6 Add validation webhook or CEL rule: either prompt or specRef must be non-empty
- [ ] 1.7 Add unit test: `TestRunTemplateSpec_Defaults` — verify default values are applied by kubebuilder markers

## 2. Chain CRD + Go Types + DAG Validation

- [ ] 2.1 Create `api/v1alpha1/chain_types.go` with ChainSpec (steps []ChainStep, projectRef), ChainStep (name, runTemplateRef, dependsOn []string, contextFrom []string, branchFrom string), ChainStatus (phase, stepCount, conditions), Chain, ChainList
- [ ] 2.2 Add kubebuilder markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, printcolumns for Name, Steps, Project, Phase, Age
- [ ] 2.3 Add `init()` function to register Chain and ChainList with SchemeBuilder
- [ ] 2.4 Run `make generate` and `make manifests` to produce deepcopy and CRD YAML
- [ ] 2.5 Implement DAG validation function `ValidateChainDAG(steps []ChainStep) error` in `api/v1alpha1/chain_validation.go` — checks for cycles (Kahn's algorithm), undefined step references in dependsOn, and validates contextFrom/branchFrom references exist in dependsOn
- [ ] 2.6 Add validation webhook or CEL rule that calls ValidateChainDAG on create and update
- [ ] 2.7 Add unit test: `TestValidateChainDAG_Linear` — A->B->C passes
- [ ] 2.8 Add unit test: `TestValidateChainDAG_Diamond` — A->B, A->C, B+C->D passes
- [ ] 2.9 Add unit test: `TestValidateChainDAG_Cycle` — A->B->A returns cycle error
- [ ] 2.10 Add unit test: `TestValidateChainDAG_UndefinedDep` — B depends on nonexistent step returns error
- [ ] 2.11 Add unit test: `TestValidateChainDAG_BranchFromNotInDependsOn` — branchFrom references a step not in dependsOn returns error

## 3. Schedule CRD + Go Types

- [ ] 3.1 Create `api/v1alpha1/schedule_types.go` with ScheduleSpec (cronExpression, runTemplateRef, chainRef, concurrencyPolicy, suspend, successfulRunHistoryLimit, failedRunHistoryLimit, projectRef), ScheduleStatus (phase, nextFireTime, lastFireTime, executionCount, activeRunRef, conditions), Schedule, ScheduleList
- [ ] 3.2 Add kubebuilder markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, printcolumns for Name, Cron, Target, Phase, NextFire, Age
- [ ] 3.3 Add `+kubebuilder:validation:Enum=Forbid;Replace;Allow` for ConcurrencyPolicy type with default "Forbid"
- [ ] 3.4 Add `init()` function to register Schedule and ScheduleList with SchemeBuilder
- [ ] 3.5 Run `make generate` and `make manifests`
- [ ] 3.6 Add validation: exactly one of runTemplateRef or chainRef must be set; cron expression must parse via a cron library (e.g., `github.com/robfig/cron/v3`)
- [ ] 3.7 Add unit test: `TestScheduleSpec_CronValidation` — valid and invalid cron expressions
- [ ] 3.8 Add unit test: `TestScheduleSpec_MutualExclusiveRef` — both set, neither set, one set

## 4. ChainRun CRD + Go Types

- [ ] 4.1 Create `api/v1alpha1/chainrun_types.go` with ChainRunSpec (chainRef, labels), ChainRunStepStatus (name, phase, agentRunRef, startedAt, completedAt, message), ChainRunStatus (phase, steps []ChainRunStepStatus, startedAt, completedAt, conditions), ChainRun, ChainRunList
- [ ] 4.2 Add kubebuilder markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, printcolumns for Name, Chain, Phase, Steps, Age
- [ ] 4.3 Add ChainRunPhase enum: Pending, Running, Succeeded, Failed, Cancelled
- [ ] 4.4 Add ChainRunStepPhase enum: Pending, Running, Succeeded, Failed, Skipped, Cancelled
- [ ] 4.5 Add `init()` function to register ChainRun and ChainRunList with SchemeBuilder
- [ ] 4.6 Run `make generate` and `make manifests`

## 5. Schedule Controller (Cron Tick Logic)

- [ ] 5.1 Create `internal/controller/schedule_controller.go` with ScheduleReconciler struct implementing reconcile.Reconciler
- [ ] 5.2 Implement Reconcile: compute nextFireTime from cron expression, check if `now >= nextFireTime`, handle suspend flag
- [ ] 5.3 Implement fire logic: create AgentRun (for runTemplateRef) or ChainRun (for chainRef), resolve RunTemplate fields with Project defaults at trigger time
- [ ] 5.4 Implement concurrency policy: Forbid (skip if active run), Replace (cancel active then create), Allow (create unconditionally) — query active runs by label `aot.uncworks.io/schedule: {name}`
- [ ] 5.5 Implement history limit garbage collection: after creating a new run, list completed runs by schedule label, delete oldest beyond the limit
- [ ] 5.6 Set RequeueAfter to time until nextFireTime (minimum 30s, maximum 60s)
- [ ] 5.7 Register the controller in `internal/controller/setup.go` or main.go
- [ ] 5.8 Add unit test: `TestScheduleReconciler_FiresOnTime` — mock clock, verify AgentRun is created when nextFireTime passes
- [ ] 5.9 Add unit test: `TestScheduleReconciler_Suspended` — verify no run is created when suspend is true
- [ ] 5.10 Add unit test: `TestScheduleReconciler_ConcurrencyForbid` — verify skip when active run exists
- [ ] 5.11 Add unit test: `TestScheduleReconciler_ConcurrencyReplace` — verify cancel + create
- [ ] 5.12 Add unit test: `TestScheduleReconciler_HistoryLimit` — verify old runs are deleted

## 6. Chain Controller (DAG Executor via Temporal)

- [ ] 6.1 Create `internal/controller/chain_controller.go` with ChainRunReconciler that watches ChainRun CRDs and starts Temporal workflows
- [ ] 6.2 Implement Reconcile: on new ChainRun (phase Pending), read the referenced Chain, initialize step statuses, start ChainRunWorkflow on Temporal
- [ ] 6.3 Create `internal/temporal/workflow_chain.go` with ChainRunWorkflow function and ChainRunWorkflowInput struct
- [ ] 6.4 Implement topological sort using Kahn's algorithm: `topoSort(steps []ChainStep) ([][]ChainStep, error)` — returns levels of parallel steps
- [ ] 6.5 Implement level-based execution loop: for each level, launch AgentRunWorkflow child workflows for all steps, wait for all to complete
- [ ] 6.6 Implement context injection: before launching a step, read completed parent AgentRun log outputs and prepend to prompt (contextFrom), read parent branch info and set repos[].branch (branchFrom)
- [ ] 6.7 Implement step status update activity: `UpdateChainRunStepStatus` — patches ChainRun status with step phase, agentRunRef, timing
- [ ] 6.8 Implement cancel handling: listen for cancel signal, cancel all running child workflows, mark remaining steps Skipped
- [ ] 6.9 Implement failure propagation: when a step fails, mark all transitively dependent steps as Skipped, continue executing independent steps in the same level
- [ ] 6.10 Register ChainRunWorkflow and activities on the Temporal worker with task queue `aot-chain-runs`
- [ ] 6.11 Add query handler `get-chain-state` returning per-step status for UI polling
- [ ] 6.12 Add unit test: `TestTopoSort_Linear` — A->B->C returns [[A],[B],[C]]
- [ ] 6.13 Add unit test: `TestTopoSort_Diamond` — returns [[A],[B,C],[D]]
- [ ] 6.14 Add unit test: `TestTopoSort_FanOut` — A->B, A->C returns [[A],[B,C]]
- [ ] 6.15 Add workflow test: `TestChainRunWorkflow_AllSucceed` — mock child workflows succeeding, verify all steps marked Succeeded
- [ ] 6.16 Add workflow test: `TestChainRunWorkflow_StepFails` — mock step A failing, verify B and C marked Skipped
- [ ] 6.17 Add workflow test: `TestChainRunWorkflow_ContextInjection` — verify child workflow input includes parent log output in prompt
- [ ] 6.18 Add workflow test: `TestChainRunWorkflow_BranchPropagation` — verify child workflow input includes parent branch in repos

## 7. REST API Endpoints

- [ ] 7.1 Add RunTemplate CRUD endpoints in `internal/server/`: GET /api/v1/run-templates, GET /api/v1/run-templates/{name}, POST /api/v1/run-templates, PUT /api/v1/run-templates/{name}, DELETE /api/v1/run-templates/{name}
- [ ] 7.2 Add POST /api/v1/run-templates/{name}/trigger — create AgentRun from template, resolve project defaults
- [ ] 7.3 Add Chain CRUD endpoints: GET /api/v1/chains, GET /api/v1/chains/{name}, POST /api/v1/chains, PUT /api/v1/chains/{name}, DELETE /api/v1/chains/{name}
- [ ] 7.4 Add POST /api/v1/chains/{name}/trigger — create ChainRun from chain
- [ ] 7.5 Add DELETE /api/v1/chains/{name} with 409 check: query Schedules referencing this chain
- [ ] 7.6 Add Schedule CRUD endpoints: GET /api/v1/schedules, GET /api/v1/schedules/{name}, POST /api/v1/schedules, PUT /api/v1/schedules/{name}, DELETE /api/v1/schedules/{name}
- [ ] 7.7 Add POST /api/v1/schedules/{name}/suspend, POST /api/v1/schedules/{name}/resume, POST /api/v1/schedules/{name}/trigger
- [ ] 7.8 Add ChainRun endpoints: GET /api/v1/chain-runs, GET /api/v1/chain-runs/{name}, POST /api/v1/chain-runs/{name}/cancel
- [ ] 7.9 Add query parameter support: ?project= filter for run-templates, chains, schedules; ?chain= filter for chain-runs
- [ ] 7.10 Add DELETE /api/v1/run-templates/{name} with 409 check: query Chains referencing this template
- [ ] 7.11 Add API tests for each endpoint (create, list, get, update, delete, trigger)

## 8. UI: ScheduleListView + ChainRunDetailView

- [ ] 8.1 Create `web/src/views/ScheduleListView.tsx` — table of schedules with name, cron (human-readable), target, status, lastFireTime, nextFireTime, executionCount, suspend toggle, trigger button
- [ ] 8.2 Create `web/src/views/ChainListView.tsx` — table of chains with name, step count, project, last triggered, trigger button
- [ ] 8.3 Create `web/src/views/ChainRunDetailView.tsx` — vertical DAG graph rendering steps as nodes with edges for dependsOn, live status per node, cancel button
- [ ] 8.4 Implement DAG layout: use a simple top-down layout algorithm (levels from topoSort, horizontal positioning within each level) — consider using reactflow or dagre for layout
- [ ] 8.5 Add live status updates to ChainRunDetailView via SSE or polling (reuse existing AgentRun SSE patterns)
- [ ] 8.6 Add click handler on DAG nodes to navigate to AgentRun detail page
- [ ] 8.7 Add chain context badge to RunListView rows: show chain name + step name for runs created by ChainRuns
- [ ] 8.8 Add routes to `web/src/AppNew.tsx`: /schedules -> ScheduleListView, /chains -> ChainListView, /chain-runs/:name -> ChainRunDetailView
- [ ] 8.9 Add navigation entries in sidebar for "Schedules" and "Chains"
- [ ] 8.10 Add RunTemplate picker component for the new-run form: dropdown of templates with "Save as Template" action
- [ ] 8.11 Run `npx tsc --noEmit -p web/tsconfig.json` — verify all new views compile without errors

## 9. RBAC + Helm Templates

- [ ] 9.1 Add CRD YAML files to `deploy/crds/`: aot.uncworks.io_runtemplates.yaml, aot.uncworks.io_chains.yaml, aot.uncworks.io_schedules.yaml, aot.uncworks.io_chainruns.yaml
- [ ] 9.2 Add RBAC ClusterRole rules in `deploy/helm/aot/templates/` granting the controller get/list/watch/create/update/patch/delete on all four new CRDs
- [ ] 9.3 Add RBAC rules for the API server ServiceAccount: get/list/watch/create/update/patch/delete on all four new CRDs
- [ ] 9.4 Add Helm values for schedule controller tick interval (default 60s) and chain run task queue name
- [ ] 9.5 Verify `helm template` renders without errors with the new templates
- [ ] 9.6 Verify `helm lint deploy/helm/aot/` passes

## 10. Tests

- [ ] 10.1 Run `go vet ./...` — no errors across all new and modified packages
- [ ] 10.2 Run `go build ./...` — all packages compile
- [ ] 10.3 Run `go test ./api/v1alpha1/...` — CRD type tests and DAG validation tests pass
- [ ] 10.4 Run `go test ./internal/controller/...` — schedule and chain controller tests pass
- [ ] 10.5 Run `go test ./internal/temporal/...` — chain workflow tests pass
- [ ] 10.6 Run `go test ./internal/server/...` — API endpoint tests pass
- [ ] 10.7 Run `npx tsc --noEmit -p web/tsconfig.json` — web UI compiles
- [ ] 10.8 Manual test: create a RunTemplate, trigger it, verify AgentRun is created with template's configuration
- [ ] 10.9 Manual test: create a Chain with 3 steps (A->B->C), trigger it, verify steps execute in order with context passing
- [ ] 10.10 Manual test: create a Schedule with a 1-minute cron, verify it fires and creates runs on schedule
- [ ] 10.11 Manual test: suspend and resume a Schedule, verify it stops and resumes firing
- [ ] 10.12 Manual test: cancel a running ChainRun, verify running steps are cancelled and pending steps are skipped
