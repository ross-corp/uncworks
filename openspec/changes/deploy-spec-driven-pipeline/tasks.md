## 1. Sidecar ExecCommand RPC

- [ ] 1.1 Add `ExecCommand` RPC to `proto/aot/agent/v1/agent.proto` with `ExecCommandRequest` (command, working_dir, timeout_seconds) and `ExecCommandResponse` (stdout, stderr, exit_code)
- [ ] 1.2 Regenerate proto code (`task proto:gen` via devbox)
- [ ] 1.3 Implement `ExecCommand` handler in `internal/sidecar/gateway.go`: run bash command via `exec.CommandContext`, capture stdout/stderr, enforce timeout
- [ ] 1.4 Write tests for ExecCommand handler (success, failure, timeout)

## 2. Replace execInSidecar with ExecCommand

- [ ] 2.1 Update `execInSidecar` in `internal/temporal/activities_spec_driven.go` to call `ExecCommand` RPC instead of `StartAgent`
- [ ] 2.2 Parse stdout from ExecCommand response (currently returns empty string)
- [ ] 2.3 Update `VerifyRun` to use actual command output from ExecCommand for verification gates
- [ ] 2.4 Remove the `pollUntilAgentDone` calls from verification code paths that now use ExecCommand

## 3. Build and Deploy

- [ ] 3.1 Rebuild all Docker images (`task docker:build`)
- [ ] 3.2 Import images into k0s (`sudo task k0s:images`)
- [ ] 3.3 Apply updated CRD (`kubectl apply -f deploy/crds/agentrun-crd.yaml`)
- [ ] 3.4 Update all deployments to use local images and restart
- [ ] 3.5 Verify all pods are running and healthy (`/healthz`, `/readyz`)

## 4. First Spec-Driven Run

- [ ] 4.1 Create a spec-driven run via API with a simple prompt (e.g., "Create a file called HELLO.txt with 'Hello World'")
- [ ] 4.2 Observe plan stage: verify OpenSpec change is created in workspace
- [ ] 4.3 Observe execute stage: verify agent implements the plan
- [ ] 4.4 Observe verify stage: verify gates run (task completion, validation, archive)
- [ ] 4.5 Check structured logs show all three stages
- [ ] 4.6 Check web UI shows stage badge and verification panel
- [ ] 4.7 Fix any runtime issues discovered (iterate: fix → rebuild → redeploy → retest)

## 5. Validation

- [ ] 5.1 Single-mode run still works (backward compatibility)
- [ ] 5.2 Spec-driven run with specContent auto-upgrades correctly
- [ ] 5.3 Verification failure triggers retry (test with a deliberately bad prompt)
- [ ] 5.4 All existing tests still pass (`go test ./...`)
