## 1. Sidecar ExecCommand RPC

- [x] 1.1 Add `ExecCommand` RPC to `proto/aot/agent/v1/agent.proto`
- [x] 1.2 Regenerate proto code (`task proto:gen` via devbox)
- [x] 1.3 Implement `ExecCommand` handler in `internal/sidecar/gateway.go`
- [ ] 1.4 Write tests for ExecCommand handler (success, failure, timeout)

## 2. Replace execInSidecar with ExecCommand

- [x] 2.1 Update `execInSidecar` to call `ExecCommand` RPC instead of `StartAgent`
- [x] 2.2 Parse stdout from ExecCommand response
- [x] 2.3 Update `VerifyRun` to use actual command output from ExecCommand
- [x] 2.4 Fix `pollUntilAgentDone` to handle UNSPECIFIED state (agent never started)

## 3. Runtime Fixes

- [x] 3.1 Cap cleanup retry policy at 5 attempts (was infinite — caused worker flood)
- [x] 3.2 Fix sidecar Dockerfile npm package name (`@fission-ai/openspec`)
- [x] 3.3 Terminate stuck old workflows flooding the Temporal worker

## 4. Pipeline Stage Configuration

- [x] 4.1 Add `PipelineConfig` and `StageConfig` structs to `api/v1alpha1/types.go`
- [x] 4.2 Add `pipelineConfig` to CRD spec schema in `deploy/crds/agentrun-crd.yaml`
- [x] 4.3 Add `PipelineConfig` message to `proto/aot/api/v1/api.proto` and regenerate
- [x] 4.4 Update `specProtoToCRD` and `crdToProto` mappings in `internal/server/grpc.go`
- [x] 4.5 Add `PipelineConfig` to `WorkflowInput` in `internal/temporal/workflow.go`
- [x] 4.6 Update `runSpecDrivenPipeline` to read per-stage config (model, timeout, retries, onFailure)
- [x] 4.7 Apply default config values when fields are zero/empty (resolveStageConfig function)
- [x] 4.8 Pass stage model to sidecar via `PI_MODEL` env var in `StartAgentRequest.env_vars`
- [x] 4.9 Update `PlanRun` activity to use plan stage config (timeout, model)
- [x] 4.10 Update `VerifyRun` activity to use verify stage config (timeout, model)
- [x] 4.11 Default stage timeouts: plan=5min, execute=15min, verify=3min

## 5. Frontend: Pipeline Config UI

- [x] 5.1 Add `PipelineConfig` type to `web/src/types/agent-run.ts`
- [x] 5.2 Add collapsible "Pipeline Settings" section in `AgentRunForm.tsx` (shown when spec-driven mode selected)
- [x] 5.3 Pipeline settings: model dropdown per stage, timeout input, retries input
- [x] 5.4 Pass pipeline config through to `createAgentRun` API call

## 6. Build, Deploy, and Validate

- [ ] 6.1 Rebuild all Docker images and deploy to aot-local
- [ ] 6.2 Create spec-driven run with default config, verify plan→execute→verify completes
- [ ] 6.3 Create spec-driven run with custom model config, verify model override works
- [ ] 6.4 Verify single-mode run still works (backward compatibility)
- [ ] 6.5 Check web UI shows pipeline config and stage progression
- [ ] 6.6 All existing tests pass (`go test ./...`)
