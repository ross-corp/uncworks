## 1. Go-Native OpenSpec JSON Parser

- [ ] 1.1 Create `parseOpenSpecJSON(raw string) (json.RawMessage, error)` — strips text prefix, finds first `{`, returns JSON bytes
- [ ] 1.2 Create `parseOpenSpecListResponse(raw string) (*OpenSpecListResponse, error)` — parses list --json into typed struct
- [ ] 1.3 Create `parseOpenSpecValidateResponse(raw string) (*OpenSpecValidateResponse, error)` — parses validate --json into typed struct
- [ ] 1.4 Create `parseOpenSpecStatusResponse(raw string) (*OpenSpecStatusResponse, error)` — parses status --json into typed struct
- [ ] 1.5 Unit tests for all parsers: valid JSON, text-prefixed JSON, empty input, malformed JSON, no JSON found

## 2. Fix 1: PlanRun Validation (was hardcoded true)

- [ ] 2.1 After pollUntilAgentDone, call `openspec validate <change> --json` via ExecCommand (no 2>/dev/null)
- [ ] 2.2 Parse validate response with Go parser — check `items[0].valid == true`
- [ ] 2.3 Call `openspec status --change <change> --json` via ExecCommand
- [ ] 2.4 Parse status response — check all `applyRequires` artifacts have `status: "done"`
- [ ] 2.5 Only return `SpecsValid: true` when BOTH validate and status pass
- [ ] 2.6 Add `ValidationErrors []string` field to PlanRunOutput for retry context
- [ ] 2.7 If validation fails, return errors so workflow can include them in retry prompt

## 3. Fix 2: OpenSpec Init in Workspace

- [ ] 3.1 At start of PlanRun, check if `openspec/config.yaml` exists via ExecCommand (`test -f openspec/config.yaml`)
- [ ] 3.2 If not, run `openspec init --tools pi --force` via ExecCommand
- [ ] 3.3 Verify init succeeded (check exit code)

## 4. Fix 3: Pre-Execute Artifact Check

- [ ] 4.1 In runSpecDrivenPipeline, after PlanRun returns, verify via ExecCommand:
  - `test -d openspec/changes/<id>` (change directory exists)
  - `test -f openspec/changes/<id>/proposal.md` (proposal exists)
  - `ls openspec/changes/<id>/specs/*/spec.md` (at least one spec exists)
  - `test -f openspec/changes/<id>/tasks.md` (tasks exist)
- [ ] 4.2 If any check fails, include missing artifacts in error and retry planning or fail

## 5. Fix 4: Gate 1 Error Handling (task completion)

- [ ] 5.1 Remove `2>/dev/null` from openspec list command
- [ ] 5.2 Replace python3 inline piping with `openspec list --json` → Go parseOpenSpecListResponse
- [ ] 5.3 If command fails or returns no data for this change, FAIL the gate (not pass)
- [ ] 5.4 If TotalTasks == 0 and command succeeded, FAIL with "no tasks found" (not pass)

## 6. Fix 5: Gate 2 Validation Bug (hardcoded true)

- [ ] 6.1 Remove line 165: `result.ValidationValid = true` (the unconditional overwrite)
- [ ] 6.2 Remove `2>/dev/null` and `| tail -1` from validate command
- [ ] 6.3 Replace with `openspec validate "<change>" --json` → Go parseOpenSpecValidateResponse
- [ ] 6.4 If ExecCommand fails, FAIL the gate with stderr (not skip)
- [ ] 6.5 If validate reports invalid, FAIL with specific issues listed

## 7. Fix 6: Gate 3 Test Command Extraction (was stub)

- [ ] 7.1 Implement `detectTestCommands` to parse spec WHEN/THEN scenarios
- [ ] 7.2 Extract backtick-wrapped commands from lines containing "run", "execute", "pass", "exit", "build", "test"
- [ ] 7.3 Execute each found command via ExecCommand
- [ ] 7.4 Gate fails if any command returns non-zero exit code
- [ ] 7.5 Include command output in AutomatedCheck result

## 8. Fix 7: Gate 4 LLM Verdict Parsing (was discarded)

- [ ] 8.1 After verify agent completes, read JSONL log from workspace via ExecCommand: `cat .aot/logs/agent.jsonl`
- [ ] 8.2 Parse JSONL to find last assistant message with JSON verdict
- [ ] 8.3 Parse verdict as `{"pass": bool, "criteria": [...]}` struct
- [ ] 8.4 Include parsed criteria in VerificationResult.LLMVerdict
- [ ] 8.5 If verdict parsing fails, log warning but don't fail gate (LLM output is best-effort)
- [ ] 8.6 If verdict says pass=false, FAIL the gate with per-criterion failures

## 9. Fix 8: Gate 5 Archive Errors (was swallowed)

- [ ] 9.1 Remove `|| true` from archive command
- [ ] 9.2 Remove `2>&1` stderr redirect — capture stdout and stderr separately
- [ ] 9.3 Check ExecCommand exit code — non-zero means archive failed
- [ ] 9.4 If archive fails, include error in VerificationResult.FailureReport
- [ ] 9.5 Archive failure should NOT cause the entire verification to fail (it's still useful info) — but should be logged prominently

## 10. Fix 9: Remove All Error Swallowing

- [ ] 10.1 Grep for all `2>/dev/null` in activities_spec_driven.go and remove them
- [ ] 10.2 Grep for all `|| true` in activities_spec_driven.go and remove them
- [ ] 10.3 Grep for all `_ =` (discarded errors) on ExecCommand calls and handle them
- [ ] 10.4 Ensure every ExecCommand call logs stdout + stderr on failure

## 11. Fix Gate 2b: File Checks in Pod (not on worker)

- [ ] 11.1 Replace `os.Stat(fullPath)` calls with ExecCommand `test -f <path>` calls
- [ ] 11.2 This ensures file checks work on multi-node clusters (no hostPath dependency)

## 12. Enhance Execute Agent Context

- [ ] 12.1 Before starting execute agent, run `openspec instructions apply --change <id> --json` via ExecCommand
- [ ] 12.2 Parse instructions response to get task list and context file paths
- [ ] 12.3 Include the parsed task list in the execute agent's prompt (so it knows exactly what to do)

## 13. Testing

- [ ] 13.1 Unit test: parseOpenSpecJSON with all edge cases
- [ ] 13.2 Unit test: parseOpenSpecListResponse, parseOpenSpecValidateResponse, parseOpenSpecStatusResponse
- [ ] 13.3 Unit test: detectTestCommands extracts commands from real spec content
- [ ] 13.4 Unit test: LLM verdict parsing from JSONL content
- [ ] 13.5 Integration test: PlanRun with mock ExecCommand returns real validation errors
- [ ] 13.6 Integration test: VerifyRun gates fail correctly on each error path
