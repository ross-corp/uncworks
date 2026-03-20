## Context

The agent run creation flow in the web UI collects repos, prompt, backend, and optional fields but skips model tier entirely, always sending "default". The backend already supports all 7 tiers — this is purely a frontend gap.

## Goals / Non-Goals

**Goals:**
- Expose model tier selection in the new run form
- Default to "default" tier so existing behavior is unchanged
- Show cost/quality hints so users can make informed choices

**Non-Goals:**
- Adding new model tiers to the backend
- Changing the LiteLLM routing logic
- Adding model-specific configuration (temperature, max tokens, etc.)

## Decisions

### Decision 1: Use shadcn Select component

The codebase already uses shadcn/ui components. The Select component provides accessible, styled dropdowns consistent with the rest of the UI.

### Decision 2: Default to "default" tier

The dropdown defaults to "default" so users who don't care about model selection get the same behavior as before.

### Decision 3: Static cost hints

Cost hints are hardcoded strings in the options array (e.g., "Budget - lowest cost"). Dynamic pricing would require a new API endpoint, which is out of scope.

## Risks / Trade-offs

- **Static hints may become stale** if pricing changes. Acceptable since hints are qualitative ("lowest cost") not quantitative ("$0.01/1K tokens").
