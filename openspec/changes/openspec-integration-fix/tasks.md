## 1. OpenSpec Init in Workspace

- [ ] 1.1 Add `initOpenSpec` helper that runs `openspec init --tools pi --force` via ExecCommand if `openspec/config.yaml` doesn't exist in workspace
- [ ] 1.2 Call `initOpenSpec` at the start of PlanRun activity, before starting the planning agent
- [ ] 1.3 Verify init creates `openspec/config.yaml` in the workspace

## 2. Fix PlanRun Validation

- [ ] 2.1 After `pollUntilAgentDone` in PlanRun, run `openspec validate <change-name> --json` via ExecCommand
- [ ] 2.2 Parse the validation JSON response in Go (not python3) — extract `items[0].valid` and `items[0].issues`
- [ ] 2.3 Run `openspec status --change <change-name> --json` via ExecCommand
- [ ] 2.4 Parse status JSON — check all `applyRequires` artifacts have `status: "done"`
- [ ] 2.5 Only return `SpecsValid: true` when both validate and status pass
- [ ] 2.6 Return validation errors in PlanRunOutput so workflow can include them in retry context
- [ ] 2.7 Add `ValidationErrors` field to PlanRunOutput

## 3. Pre-Execute Artifact Check

- [ ] 3.1 After PlanRun returns, verify OpenSpec change directory exists via ExecCommand: `test -d openspec/changes/<id>`
- [ ] 3.2 Verify `proposal.md` exists: `test -f openspec/changes/<id>/proposal.md`
- [ ] 3.3 Verify at least one spec file exists: `ls openspec/changes/<id>/specs/*/spec.md`
- [ ] 3.4 Verify `tasks.md` exists: `test -f openspec/changes/<id>/tasks.md`
- [ ] 3.5 If any check fails, retry planning or fail with clear error

## 4. Fix Verify Gate Error Handling

- [ ] 4.1 Remove all `2>/dev/null` redirects from ExecCommand calls in VerifyRun
- [ ] 4.2 Replace python3 JSON piping with direct Go JSON parsing — write `parseOpenSpecListJSON` helper that strips text prefix and unmarshals
- [ ] 4.3 Replace python3 validate piping with direct Go JSON parsing — write `parseOpenSpecValidateJSON` helper
- [ ] 4.4 Fix validation gate: actually check `valid: true` from parsed response, don't hardcode
- [ ] 4.5 Fix archive gate: remove `|| true`, capture and report archive errors
- [ ] 4.6 Include stderr in failure reports when ExecCommand returns non-zero exit code
- [ ] 4.7 Log all OpenSpec command outputs (stdout + stderr) for debugging

## 5. Use openspec instructions for Task Context

- [ ] 5.1 Before execute stage, run `openspec instructions apply --change <id> --json` via ExecCommand
- [ ] 5.2 Parse the instructions JSON to get the task list and context file paths
- [ ] 5.3 Include the instructions context in the execute agent's prompt (so the agent has the full spec context)

## 6. Testing

- [ ] 6.1 Unit test: `parseOpenSpecListJSON` handles valid JSON, text-prefixed JSON, empty response, error response
- [ ] 6.2 Unit test: `parseOpenSpecValidateJSON` handles valid response, invalid change, malformed JSON
- [ ] 6.3 Unit test: `initOpenSpec` succeeds when workspace has no config, skips when config exists
- [ ] 6.4 Integration test: PlanRun with mock sidecar returns real validation errors on invalid change
- [ ] 6.5 Integration test: VerifyRun fails correctly when openspec list reports incomplete tasks
