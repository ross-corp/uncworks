## 1. Restructure VerifyRun: Structural Pre-screening (Tier 1)

- [ ] 1.1 In `activities_spec_driven.go`, add a section comment before Gate 1 labeling it "Tier 1: Structural pre-screening" — rename existing gate comments for clarity
- [ ] 1.2 Verify that gates 1-3 (task completion, spec validation, file existence, test commands) already short-circuit on failure and return before reaching the LLM judge — confirm no code change needed for short-circuit behavior
- [ ] 1.3 Add a heartbeat message after all structural checks pass: `activity.RecordHeartbeat(ctx, "structural checks passed, starting manage agent review")`
- [ ] 1.4 Add unit test: `TestVerifyRun_StructuralFailure_SkipsManageReview` — mock a failing test command, verify the manage agent `StartAgent` is never called

## 2. Add VerifyRunInput and VerificationResult Fields

- [ ] 2.1 Add `ManageModel string` field to `VerifyRunInput` in `workflow_spec_driven.go`
- [ ] 2.2 Add `PreviousReviewFeedback string` field to `VerifyRunInput`
- [ ] 2.3 Add `ReviewFeedback string` field to `VerificationResult` with JSON tag `"reviewFeedback,omitempty"`
- [ ] 2.4 In the workflow's verify input construction (line ~571), populate `ManageModel` from the manage stage config and `PreviousReviewFeedback` from the `lastReviewFeedback` variable

## 3. Build Manage Agent Review Prompt

- [ ] 3.1 Add helper function `readImplementAgentLog(ctx, sidecarClient, agentRunName, workDir) string` — calls `execInSidecar` to read `tail -200 .aot/logs/agent.jsonl`, parses JSONL lines to extract assistant text and tool call summaries, returns formatted string
- [ ] 3.2 Add helper function `readSpecScenarios(ctx, sidecarClient, agentRunName, specDir, changeName) string` — reads all `spec.md` files under the change's `specs/` directory, concatenates and returns them
- [ ] 3.3 Add helper function `buildManageReviewPrompt(changeName, gitDiff, specScenarios, implementLog, previousFeedback string) string` — assembles the four components into the structured review prompt described in the design
- [ ] 3.4 Expand the git diff fetch from `git diff HEAD~1 --stat` to `git diff HEAD~1` (full diff), truncated to 8000 characters via a `truncate(output, 8000)` call
- [ ] 3.5 Add unit test: `TestBuildManageReviewPrompt_FirstAttempt` — verify prompt includes diff, specs, implement log, and no previous feedback section
- [ ] 3.6 Add unit test: `TestBuildManageReviewPrompt_RetryAttempt` — verify prompt includes previous review feedback section
- [ ] 3.7 Add unit test: `TestReadImplementAgentLog_MissingFile` — verify graceful fallback when JSONL file doesn't exist

## 4. Replace LLM Judge with Manage Agent Review Session

- [ ] 4.1 In `VerifyRun`, replace the Gate 4 section (current LLM judge) with the manage agent review session: call `buildManageReviewPrompt`, then `StartAgent` with `stage: "verify"`, the manage model, and the assembled prompt
- [ ] 4.2 Set the `StartAgentRequest.Model` field to `input.ManageModel` (fall back to empty string for sidecar default if not set)
- [ ] 4.3 Poll the manage agent session using `pollUntilAgentDone` with heartbeats (reuse existing polling pattern from the current LLM judge code)
- [ ] 4.4 After the manage agent completes, read its output from `.aot/logs/agent.jsonl` (the verify agent's log) and parse the verdict JSON — reuse `parseLLMVerdict` (same JSON shape: `{pass, criteria}`)
- [ ] 4.5 Extract the `feedback` field from the manage agent's verdict JSON and store it in `result.ReviewFeedback`
- [ ] 4.6 On manage agent failure, populate `result.FailureReport` with a concise summary and `result.ReviewFeedback` with the detailed per-scenario analysis
- [ ] 4.7 Add unit test: `TestVerifyRun_ManageAgentReview_Pass` — mock sidecar responses for structural checks passing and manage agent returning a passing verdict
- [ ] 4.8 Add unit test: `TestVerifyRun_ManageAgentReview_Fail` — mock manage agent returning a failing verdict, verify `ReviewFeedback` is populated

## 5. Replace Generic Retry Prompt with Review Feedback

- [ ] 5.1 In `workflow_spec_driven.go`, add a `lastReviewFeedback string` variable alongside `lastFailureReport` in the retry loop
- [ ] 5.2 After verify fails, set `lastReviewFeedback = verifyOutput.Result.ReviewFeedback` and fall back: if `lastReviewFeedback != ""` use it as `lastFailureReport`, else keep `verifyOutput.Result.FailureReport`
- [ ] 5.3 Replace the retry prompt template from "PREVIOUS ATTEMPT FAILED:\n{report}\n\nFix the issues..." to "MANAGE AGENT REVIEW FAILED (attempt {N}):\n\n{review feedback}\n\nFix the issues identified above..."
- [ ] 5.4 Pass `lastReviewFeedback` into `VerifyRunInput.PreviousReviewFeedback` on subsequent verify calls
- [ ] 5.5 Add workflow test: `TestRetryLoop_UsesReviewFeedback` — verify that after a manage agent review failure, the implement agent's retry prompt contains the review feedback, not the generic template
- [ ] 5.6 Add workflow test: `TestRetryLoop_FallsBackToFailureReport` — verify that after a structural failure (no review feedback), the implement agent's retry prompt contains the structural failure report

## 6. Update aot-determinism.ts for Verify Stage HITL

- [ ] 6.1 In `aot-determinism.ts`, add a comment in the role-based tool policies section documenting that manage agents can call `ask_user` during verify for HITL escalation of implement agent questions
- [ ] 6.2 Verify that no existing policy blocks `ask_user` for `role === "manage"` in any stage — confirm the manage role's only restrictions are write/edit to non-openspec, non-.aot paths
- [ ] 6.3 Add integration test or manual verification: start a manage agent with `PI_STAGE=verify PI_ROLE=manage`, call `ask_user`, verify it is not blocked

## 7. Add Implement Output Log Reading to Verify Activity

- [ ] 7.1 In `VerifyRun`, after structural checks pass and before building the review prompt, call `readImplementAgentLog` to read the implement agent's JSONL output
- [ ] 7.2 Handle the case where the implement agent log path differs from the verify agent log path — the implement agent writes to `.aot/logs/agent.jsonl` during execute; the verify agent writes to the same path but as a different agent run ID. Ensure the implement log is read BEFORE starting the verify agent (which may overwrite the file)
- [ ] 7.3 If the sidecar uses agent-run-specific log paths (e.g., `.aot/logs/{agentRunName}.jsonl`), read the implement agent's log using the implement agent run name, not the verify agent run name
- [ ] 7.4 Add unit test: `TestReadImplementAgentLog_ParsesJSONL` — provide a sample JSONL file, verify the function extracts assistant messages and tool call summaries

## 8. UI: Show Review Feedback in Verification Panel

- [ ] 8.1 In `RunDetailView.tsx`, check `verificationResult.reviewFeedback` and render it in the verification panel when present
- [ ] 8.2 Display the review feedback in a collapsible section labeled "Manage Agent Review" with per-scenario pass/fail indicators from the `criteria` array
- [ ] 8.3 Show question routing state when the manage agent called `ask_user` during verify — display what question was asked and what the human answered (read from the run's HITL history)
- [ ] 8.4 Run `npx tsc --noEmit -p web/tsconfig.json` — verify the web UI compiles with the new `reviewFeedback` field on the verification result type

## 9. End-to-End Verification

- [ ] 9.1 Run `go test ./internal/temporal/...` — all existing and new verify/workflow tests pass
- [ ] 9.2 Run `go vet ./...` and `go build ./...` — no errors
- [ ] 9.3 Run `npx tsc --noEmit -p web/tsconfig.json` — web compiles
- [ ] 9.4 Manual test: trigger a spec-driven run where the implement agent leaves a question in output, verify the manage agent surfaces it during review
- [ ] 9.5 Manual test: trigger a spec-driven run that fails structural checks, verify the manage agent review is not started (check logs for absence of manage agent `StartAgent` call)
