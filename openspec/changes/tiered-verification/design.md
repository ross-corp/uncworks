## Context

The current verification system in `VerifyRun` runs five sequential gates: task completion, spec validation, file existence, test commands, and an LLM judge. The LLM judge is a single `StartAgent` call with a prompt asking the agent to evaluate the implementation. This has two problems. First, the judge uses the same model that executed the task (same `StartAgent` path, same sidecar), creating a self-review bias. Second, questions the implement agent wrote in its output are never surfaced to anyone during the run — nobody reads the agent's output log during verify.

The existing retry loop in `runSpecDrivenPipeline` feeds `lastFailureReport` (a short string from `VerificationResult.FailureReport`) back into the implement agent's retry prompt using a generic template: "PREVIOUS ATTEMPT FAILED: {report}. Fix the issues..." This is not actionable enough when the failure is a nuanced spec mismatch rather than a structural problem.

This change restructures verification into three tiers: automated structural checks (fast, no LLM), a manage agent review session (agent-as-reviewer), and HITL escalation (only when the manage agent decides).

## Goals / Non-Goals

**Goals:**
- Run structural checks first and skip the LLM review when structural checks fail
- Replace the single-prompt LLM judge with a full manage agent session that has tool access
- Route implement agent questions through the manage agent to the human
- Replace the generic retry template with the manage agent's detailed per-scenario feedback
- Use the manage model tier (typically stronger) for the reviewer

**Non-Goals:**
- Changing the structural checks themselves (gates 1-3 are preserved as-is)
- Adding new structural checks beyond what already exists
- Changing the implement agent's behavior or tools during execute
- Modifying the plan stage

## Decisions

### 1. Restructure VerifyRun: structural gates first, then manage agent session

The current `VerifyRun` runs gates 1-4 sequentially, where gate 4 (LLM judge) runs even if the diff is trivially wrong. Restructure so gates 1-3 (task completion, spec validation, file existence, test commands) execute first as "Tier 1: Structural pre-screening." If any structural gate fails, `VerifyRun` returns immediately with the failure — the manage agent session is never started.

Only when all structural gates pass does `VerifyRun` proceed to "Tier 2: Manage agent review." This saves the cost and latency of an LLM session on obvious failures (no files changed, tests fail, tasks not checked off).

The code change is minimal: the existing gates 1-3 already short-circuit on failure. The only structural change is renaming the section comment from "Gate 4: LLM judge" to "Tier 2: Manage agent review" and replacing the simple prompt with the full manage agent session described below.

### 2. Manage agent review prompt construction

The manage agent's review prompt is built from four components, assembled in `VerifyRun` before calling `StartAgent`:

**Component 1: Git diff summary.** Already fetched by the current code (`git diff HEAD~1 --stat`). Expand this to include `git diff HEAD~1` (full diff, not just stat) truncated to 8000 characters. The manage agent can also read files directly via tools, but the diff in the prompt gives it an overview.

**Component 2: Spec scenarios.** Read all `spec.md` files under `/workspace/openspec/changes/{changeName}/specs/*/spec.md`. Concatenate them into the prompt. The manage agent needs the full WHEN/THEN scenarios to evaluate each one.

**Component 3: Implement agent output excerpts.** Read `/workspace/.aot/logs/agent.jsonl` — this is the implement agent's structured log. Extract the last 200 lines (configurable). Parse JSONL to find text content (assistant messages, tool calls, tool results). Include a summarized excerpt in the prompt. The manage agent uses this to understand what the implement agent did, what it was confused about, and whether it asked questions.

**Component 4: Previous review feedback (on retries).** Pass the previous attempt's `ReviewFeedback` from the workflow via a new field on `VerifyRunInput`. On the first attempt this is empty. On retries, it contains the manage agent's prior analysis. The manage agent can check whether the implement agent addressed the feedback.

The assembled prompt looks like:

```
You are the manage agent reviewing the implementation of OpenSpec change '{changeName}'.
You are a senior engineer reviewing a junior engineer's PR. Be thorough but fair.

## Git Diff
{full diff, truncated to 8000 chars}

## Spec Scenarios
{concatenated spec.md files}

## Implement Agent Output (last 200 lines)
{parsed excerpts from agent.jsonl}

{if retries: ## Previous Review Feedback\n{prior feedback}}

## Instructions
1. Evaluate each WHEN/THEN scenario against the implementation.
2. Read source files and run commands if needed to verify behavior.
3. Check the implement agent's output for unanswered questions — if you find questions you cannot answer from context, call ask_user to escalate to the human.
4. Output your verdict as JSON: {"pass": true/false, "criteria": [...], "feedback": "..."}
   - criteria: array of {scenario, pass, explanation} for each spec scenario
   - feedback: detailed per-scenario analysis (used as retry prompt if fail)
```

### 3. Implement agent output log reading

Add a new helper function `readImplementAgentLog(ctx, sidecarClient, agentRunName, workDir) string` that:

1. Calls `execInSidecar` to run `tail -200 /workspace/.aot/logs/agent.jsonl` (the implement agent's log lives at this path because the implement agent ran first in the same pod).
2. Parses each JSONL line to extract meaningful content: assistant text, tool call names/inputs, tool results (truncated). Skips binary content, large file reads, and repetitive entries.
3. Returns a formatted string suitable for inclusion in the review prompt.

If the log doesn't exist or is empty (e.g., implement agent crashed before writing), return a placeholder string: "(no implement agent output available)".

The implement agent and verify agent share the same pod filesystem. The implement agent writes to `/workspace/.aot/logs/agent.jsonl` during execute. When verify runs afterward, the file is still there on the PVC.

### 4. Manage agent review replaces the generic retry template

Currently, the workflow's retry loop uses `lastFailureReport` (from `VerificationResult.FailureReport`) to build the retry prompt:

```go
prompt = fmt.Sprintf("PREVIOUS ATTEMPT FAILED:\n%s\n\nFix the issues...", lastFailureReport)
```

The `FailureReport` is a short string like "LLM judge failed: scenario X: explanation." This is not enough for the implement agent to know what to fix.

Add a new field `ReviewFeedback string` to `VerificationResult`. The manage agent's detailed per-scenario analysis goes here. In the workflow, prefer `ReviewFeedback` over `FailureReport` for the retry prompt:

```go
if verifyOutput.Result.ReviewFeedback != "" {
    lastFailureReport = verifyOutput.Result.ReviewFeedback
} else {
    lastFailureReport = verifyOutput.Result.FailureReport
}
```

The retry prompt template also changes from the generic one to:

```
MANAGE AGENT REVIEW FAILED (attempt {N}):

{review feedback — per-scenario analysis from the manage agent}

Fix the issues identified above and complete the OpenSpec change '{changeName}'.
Read specs at /workspace/openspec/changes/{changeName}/
Mark ALL tasks [x] in tasks.md when complete.

Original task: {prompt}
```

This gives the implement agent specific, actionable feedback rather than a generic instruction.

### 5. Manage model vs implement model

The existing dual-model configuration (`manageModelTier` / `implementModelTier` in `PipelineConfigInput`) maps directly to this design. The implement agent uses `implementModelTier` (fast/cheap for volume work). The manage agent review uses `manageModelTier` (stronger for code review).

Currently, the verify stage's `StartAgent` call doesn't specify which model tier to use — it inherits from the verify stage config. Change the `VerifyRunInput` to accept a `ManageModel string` field. The workflow populates this from the pipeline config's manage model tier. The verify activity passes this to `StartAgent` when starting the manage agent session.

If `ManageModel` is empty, fall back to the verify stage's default model. This preserves backward compatibility with configs that don't specify a manage model.

### 6. aot-determinism.ts role policy update for verify stage

The current `aot-determinism.ts` blocks `ask_user` for `role === "implement"` but has no stage-specific policy for the verify stage. The manage agent running during verify needs to call `ask_user` to escalate questions.

The change is to the role policy section in the `tool_call` handler. The manage agent during verify is already `role: "manage"`, so the existing implement block doesn't apply. However, we need to confirm that the manage role during verify has no blanket block on `ask_user`. Currently it doesn't — the manage role only blocks writes outside `/workspace/openspec/` and `/workspace/.aot/`. So `ask_user` already works for manage agents.

The only code change needed in `aot-determinism.ts` is documentation/comment clarity: add a comment in the role policy section noting that the manage agent can call `ask_user` during verify for HITL escalation. No logic change required — the existing policies already permit it.

### 7. HITL state transitions during verify

When the manage agent calls `ask_user` during its review session:

1. The `askUser` function in `aot-determinism.ts` writes `/workspace/.aot/input/question.json` and polls for `/workspace/.aot/input/response.txt`.
2. The sidecar detects the question file and transitions the agent run to `waiting_for_input` state (existing behavior).
3. The UI shows the question in the HITL banner (existing behavior).
4. The human responds via the UI, which calls `SendInput` on the sidecar, which writes `response.txt`.
5. The `askUser` function reads the response and returns it to the manage agent.
6. The manage agent incorporates the answer and continues its review.

The verify activity's `pollUntilAgentDone` loop sees the agent in `waiting_for_input` state and continues polling (it doesn't treat `waiting_for_input` as a terminal state). The activity records heartbeats during this time to prevent Temporal timeout.

No changes to the HITL infrastructure are needed — the existing `ask_user` tool, question/response file protocol, sidecar state management, and UI banner all work as-is. The only new behavior is that HITL can now occur during the verify stage (previously it was only during plan/execute).

### 8. New fields on VerifyRunInput and VerificationResult

**VerifyRunInput additions:**
- `ManageModel string` — the model to use for the manage agent review session
- `PreviousReviewFeedback string` — feedback from the prior verify attempt (empty on first attempt)

**VerificationResult additions:**
- `ReviewFeedback string` — the manage agent's full per-scenario review analysis

The workflow populates `PreviousReviewFeedback` from the prior attempt's `ReviewFeedback`:

```go
verifyInput := VerifyRunInput{
    // ... existing fields ...
    ManageModel:            manageCfg.Model,
    PreviousReviewFeedback: lastReviewFeedback,
}
```

And reads it back:

```go
lastReviewFeedback = verifyOutput.Result.ReviewFeedback
if lastReviewFeedback != "" {
    lastFailureReport = lastReviewFeedback
} else {
    lastFailureReport = verifyOutput.Result.FailureReport
}
```

## Risks / Trade-offs

- **[Risk] Manage agent review takes longer than the single-prompt judge.** A full agent session with tool access can take 2-5 minutes vs 30 seconds for a single prompt. Mitigation: structural pre-screening catches obvious failures fast. The manage agent only runs when structural checks pass, so the additional time is spent on cases that need genuine code review.
- **[Risk] Manage agent calls ask_user during verify, blocking the pipeline.** If the human doesn't respond within 5 minutes, `askUser` returns "No response received." The manage agent must handle this gracefully — the review prompt instructs it to make a best-effort decision if HITL times out. Mitigation: the 5-minute timeout in `aot-determinism.ts` prevents indefinite blocking.
- **[Risk] Implement agent JSONL log is very large.** Some runs produce thousands of JSONL lines. Mitigation: only read the last 200 lines (tail). The manage agent can read the full log via tools if it needs more context.
- **[Trade-off] ReviewFeedback increases payload size in Temporal.** The `VerificationResult` travels through Temporal as an activity output. Long review feedback (multi-KB) increases the payload. This is acceptable — Temporal supports payloads up to 2 MB, and review feedback is typically 1-5 KB.
- **[Trade-off] No logic change needed in aot-determinism.ts.** The existing role policies already permit `ask_user` for manage agents. This means we're relying on implicit permission rather than explicit allow. Acceptable because the manage role's policy is already well-defined and intentionally permissive for `ask_user`.
