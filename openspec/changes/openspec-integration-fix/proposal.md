## Why

A line-by-line audit of the spec-driven pipeline reveals 9 critical issues. The pipeline claims to use OpenSpec for plan validation, task tracking, and verification — but most of it is faked or broken:

1. **PlanRun returns hardcoded `SpecsValid: true`** (line 68-71) without calling any OpenSpec CLI command
2. **No `openspec init` in workspace** — the agent is told to use OpenSpec CLI but the workspace may not be initialized
3. **No pre-execute artifact check** — nothing verifies the plan actually produced artifacts before starting execution
4. **Verify Gate 1 silently passes on command failure** — if `openspec list` fails, TotalTasks=0 and the gate passes instead of failing
5. **Verify Gate 2 unconditionally sets `ValidationValid = true`** (line 165) — this overwrites the actual validation check, meaning validation can NEVER fail
6. **Verify Gate 3 is a stub** — `detectTestCommands()` returns nil, no test commands are ever extracted or run
7. **Verify Gate 4 discards the LLM judge's output** — the agent runs and writes a JSON verdict but nobody reads it (`_ = pollUntilAgentDone`)
8. **Verify Gate 5 swallows archive errors** — `|| true` means archive failure is invisible
9. **All gates use `2>/dev/null` and python3 piped JSON parsing** — errors are hidden, parsing is fragile

The net effect: the verify stage **always passes** unless OpenSpec list finds incomplete tasks (Gate 1) or a file referenced in a spec doesn't exist (Gate 2b). Gates 2, 3, 4, and 5 cannot cause failure. The pipeline succeeds by accident (the agent does good work) rather than by design (Temporal verifies programmatically).

## What Changes

- **Fix 1 (PlanRun validation)**: After agent completes, call `openspec validate --json` and `openspec status --json` via ExecCommand. Parse responses in Go. Only return SpecsValid=true when both pass.
- **Fix 2 (OpenSpec init)**: Run `openspec init --tools pi --force` in workspace at start of PlanRun if no config exists.
- **Fix 3 (Pre-execute check)**: Verify change directory, proposal.md, specs/, and tasks.md exist before starting execute agent.
- **Fix 4 (Gate 1 error handling)**: Fail when `openspec list` command fails or returns no change data. Don't treat TotalTasks=0 as a pass.
- **Fix 5 (Gate 2 validation bug)**: Remove the unconditional `result.ValidationValid = true` on line 165. Let the actual validation result stand.
- **Fix 6 (Gate 3 test extraction)**: Parse spec WHEN/THEN scenarios for command references (backtick-wrapped commands with "run", "execute", "pass" keywords). Execute found commands.
- **Fix 7 (Gate 4 LLM verdict)**: Read the verify agent's structured log output after it completes. Parse the JSON verdict from the JSONL file. Include per-scenario results in VerificationResult.
- **Fix 8 (Gate 5 archive)**: Remove `|| true`. Check exit code. Include archive errors in failure report.
- **Fix 9 (Error handling)**: Remove all `2>/dev/null`. Replace python3 inline JSON piping with Go-native JSON parsing. Create `parseOpenSpecJSON` helper that strips text prefixes and unmarshals.

## Capabilities

### New Capabilities

None.

### Modified Capabilities
- `run-pipeline`: Add openspec init, real plan validation, pre-execute artifact check.
- `run-verification`: Fix all 5 gates to use real error handling, real validation, real test extraction, real LLM verdict parsing, real archive reporting.

## Impact

- `internal/temporal/activities_spec_driven.go` — Major rewrite of PlanRun and VerifyRun
- `internal/temporal/workflow_spec_driven.go` — Add pre-execute artifact check
- `internal/sidecar/gateway.go` — No changes needed (system prompts are already correct)
- Tests for all 9 fixes
