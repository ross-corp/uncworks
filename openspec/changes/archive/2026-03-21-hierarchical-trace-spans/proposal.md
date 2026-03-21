## Why

Traces are currently a flat list of spans with no parent-child hierarchy. The "EXECUTE" stage separator is a visual hack — a CSS divider wedged between spans rather than a true hierarchical relationship. This means you can't collapse/expand stages, can't see stage-level aggregate metrics (total duration, token usage, tool count), and the waterfall looks nothing like Grafana Tempo or Langfuse. Engineers can't answer basic questions: "How long did the plan stage take?" "How many tokens did the execute stage use?" "Which attempt succeeded?"

## What Changes

- Create **stage parent spans** (PLAN, EXECUTE, VERIFY) from the Temporal workflow, with start/end times and aggregate metadata
- Link all child spans to their stage parent via `parentId`
- Add **rich metadata** to spans following OTel GenAI semantic conventions: token usage, model name, cost estimate, context utilization, tool counts
- Add a **trace root span** representing the entire pipeline run
- Support **retry cycles** as separate EXECUTE/VERIFY parent spans with attempt numbers
- Render the waterfall with **collapsible parent rows** showing aggregate stats
- Show **stage summary** in the detail panel: total tokens, total cost, tool success/failure ratio

## Capabilities

### New Capabilities
- `stage-parent-spans`: Temporal workflow creates parent spans for each pipeline stage with timing, attempt number, and aggregate metadata. Child spans link via parentId.
- `span-token-metadata`: LLM thought spans include token usage (input, output, cache), model name, and estimated cost following OTel gen_ai semantic conventions.
- `trace-root-span`: A single root span per run representing the entire pipeline, with aggregate stats rolled up from all stages.

### Modified Capabilities
- `trace-detail-panel`: Detail panel shows stage-level aggregates when a parent span is selected (total duration, total tokens, tool count, cost). Remove the flat stage separator CSS hack.

## Impact

- **Temporal workflow** (`workflow_spec_driven.go`): Emit stage parent spans via a new `WriteTraceSpan` activity or by writing to a shared trace file
- **Sidecar** (`gateway.go`): Accept `parentSpanId` in StartAgent request, set on all child spans
- **API** (`traces.go`): No changes needed — already reads spans.jsonl and serves them
- **Frontend** (`TraceTimeline.tsx`): Already has tree-building via parentId — will naturally render hierarchy. Needs collapsible parent rows and aggregate display.
- **Proto**: Add `parentSpanId` field to StartAgentRequest
- **Cost data**: Read from LiteLLM response headers or pi's JSONL token usage events
