## Why

The current verification system has five gates (task completion, structural validation, file existence, test commands, LLM judge) but still passes runs where the agent hallucinated entirely wrong output. The root cause: the LLM judge uses the same model that executed the task, creating a "fox guarding the henhouse" problem. The judge asks "did I do a good job?" and answers "yes." There is also no path for the implement agent's questions to reach a reviewer — questions are written into output text that nobody reads during the run, leading to confused agents guessing instead of asking.

The verification system needs three tiers: automated structural checks (fast, no LLM), a manage agent review session (agent-as-reviewer, not just LLM-as-judge), and human-in-the-loop escalation (only when the manage agent decides it is needed). Questions should flow up the chain: implement agent asks manage agent, manage agent asks human. The manage agent acts as a senior engineer reviewing a junior engineer's PR — it reads the diff, checks the specs, investigates unclear areas, and produces actionable feedback.

## What Changes

### Tier 1: Structural pre-screening (automated, no LLM)
Keep the existing gates 1-3 (task completion, OpenSpec validation, file existence, test commands) but run them BEFORE the manage agent review. If structural checks fail, skip the expensive LLM review and retry immediately with the structural failure as context. This saves time and money on obvious failures (no files changed, tests fail, tasks not checked off).

### Tier 2: Manage agent as reviewer (replaces LLM judge)
Replace the current single-prompt LLM judge (Gate 4) with a full manage agent review session. The manage agent runs as a real agent with tool access — it reads the git diff, reads spec scenarios, examines modified files, and reads the implement agent's output log for unanswered questions. It uses the manage model (configurable, typically a stronger model than the implement model). The review produces detailed per-scenario feedback, not just pass/fail. On failure, the manage agent's review comments become the retry prompt for the implement agent — specific, actionable feedback instead of the generic "fix the issues" template.

### Tier 3: HITL escalation (manage agent decides)
The manage agent can call `ask_user` during its review session to escalate questions to the human. This happens when: the manage agent reads implement agent questions it cannot answer, when the spec is ambiguous and needs product clarification, or when the implementation is technically correct but the manage agent is unsure if it matches intent. The run enters `waiting_for_input` state. The human sees a contextualized question synthesized by the manage agent — not raw agent confusion. After the human responds, the manage agent incorporates the answer and either passes the run or triggers a retry with the clarification injected.

### Question routing: implement to manage to human
The implement agent is blocked from calling `ask_user` (existing policy). When it has questions, it writes them in its output. During the Tier 2 review, the manage agent reads the implement agent's structured log and identifies unanswered questions. It either answers them (injecting the answer into the retry prompt) or escalates to the human via `ask_user`. This creates a chain of command: implement (junior) asks manage (tech lead), manage asks human (product owner). Humans only see well-formed, contextualized questions.

### Different models for judge and executor
The existing dual-model config (`manageModelTier` / `implementModelTier`) maps directly to this design. The implement agent uses a fast/cheap model for volume work. The manage agent uses a stronger model for planning and reviewing. This mirrors the human pattern: junior developers write code, senior engineers review it. The reviewer catching mistakes justifies a more expensive model.

## Capabilities

### New Capabilities
- `manage-agent-review`: Replace the single-prompt LLM judge with a full manage agent review session during verify — reads diff, checks specs, examines files, produces detailed feedback
- `question-routing`: Manage agent reads implement agent output during review, identifies unanswered questions, either answers them or escalates to human via ask_user
- `structural-prescreening`: Run automated checks (tasks, validation, files, tests) before the manage review — skip expensive LLM review on obvious structural failures
- `review-feedback-loop`: Manage agent review comments become the retry prompt for the implement agent, replacing the generic "fix the issues" template

### Modified Capabilities
- None (existing verify gates are preserved and reordered, not removed)

## Impact

- **Modified**: `internal/temporal/activities_spec_driven.go` — restructure `VerifyRun` to run structural checks first, then start a manage agent review session instead of the current LLM judge prompt
- **Modified**: `internal/temporal/workflow_spec_driven.go` — pass implement agent's structured log path to verify activity, use manage agent review feedback as retry prompt instead of generic template
- **Modified**: `extensions/aot-determinism.ts` — allow manage agent to call `ask_user` during verify stage (currently only plan/execute stages have role policies)
- **Modified**: `internal/sidecar/gateway.go` — expose implement agent log path to verify agent for cross-agent output reading
- **New**: Review prompt construction — build the manage agent's review prompt with: git diff summary, spec scenarios, implement agent output excerpts, previous review feedback (on retries)
- **Modified**: `web/src/views/RunDetailView.tsx` — show manage agent review feedback in the verification panel, show question-routing state (implement asked X, manage answered/escalated Y)
- **Unchanged**: The 4 structural gates (tasks, validation, files, tests) — preserved as Tier 1 pre-screening
- **Unchanged**: HITL infrastructure (ask_user tool, waiting_for_input state, UI banner) — reused for Tier 3
- **Unchanged**: Dual model config (manageModelTier, implementModelTier) — already built, now properly utilized
