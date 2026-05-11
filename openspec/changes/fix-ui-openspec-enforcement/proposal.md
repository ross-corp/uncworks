## Why

The UNCWORKS UI has accumulated several usability regressions: the ChainRunDetailView has a redundant Timeline tab, mislabeled tabs, and awkward centering in the Runs sub-tab; the RunListView shows no approval-mode indicator; and the trace detail panel omits tool output, making debugging opaque. Compounding this, OpenSpec is installed and enforced in the codebase but is not being applied to agent runs at all — the agent templates create runs with no openspec context, meaning verifications have no structured task list to check against. These two issues (broken UI and missing OpenSpec enforcement) both block reliable demos and production use.

## What Changes

- **Remove** the Timeline tab from ChainRunDetailView (redundant with Runs tab; same data, confuses users)
- **Rename** the "DAG" tab to "Overview" in ChainRunDetailView
- **Fix** the Runs sub-tab in ChainRunDetailView: remove `max-w-2xl mx-auto` centering so content fills the panel width
- **Add** approval-mode indicator ("LLM judge" / "HITL") next to the kind badge in RunListView's unified run rows
- **Add** tool output/result display in the trace detail panel alongside existing tool input
- **Enforce** OpenSpec in agent run templates: all templates shall inject `openspec` CLI instructions into the agent's prompt and pass `--openspec-change` so the verification stage has a structured task list to gate against

## Capabilities

### New Capabilities

- `approval-mode-badge`: RunListView unified rows SHALL show the approval mode ("llm-judge" or "hitl") as a secondary badge next to the kind badge
- `trace-tool-output`: The trace detail panel SHALL display the tool's output/result (`toolOutput` from span metadata) alongside tool input, collapsed by default

### Modified Capabilities

- `chain-dag-viz`: Remove the Timeline view tab requirement (redundant with Runs tab); rename the "DAG" tab label to "Overview"
- `run-verification`: Agent run templates SHALL pass openspec change context so that verification's `openspec list` gate has a valid change to check against

## Impact

- `web/src/views/ChainRunDetailView.tsx` — tab changes (remove timeline, rename dag)
- `web/src/views/RunListView.tsx` — add approval-mode badge to `UnifiedRunRow`
- `web/src/components/TraceTimeline.tsx` — add tool output section to `SpanDetail`
- Agent run templates in the database / `cmd/uncworks/` — update to include openspec invocation in prompts
- `internal/temporal/workflow_spec_driven.go` / activities — enforce openspec change creation before execute stage
