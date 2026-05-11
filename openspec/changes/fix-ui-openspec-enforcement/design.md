## Context

ChainRunDetailView currently has three tabs: "DAG", "Runs", and "Timeline". The Timeline tab duplicates the Runs tab content (same data, different layout) causing confusion. The "DAG" label doesn't communicate that the tab is the primary overview view. The Runs sub-tab wraps content in `max-w-2xl mx-auto`, centering it unnecessarily in a full-panel context.

RunListView's unified run rows show a kind badge ("one-shot", "chain", "scheduled") but no approval-mode indicator, leaving users unable to quickly identify HITL vs. LLM-judge runs.

The trace detail panel (SpanDetail in TraceTimeline.tsx) shows tool input in a collapsible section but omits the tool's output/result (`toolOutput` metadata field), making it impossible to inspect what a tool returned without reading raw logs.

OpenSpec is installed and used in the verification workflow (`workflow_spec_driven.go`), but agent run templates do not pass a `--change` name or inject openspec instructions into the agent prompt. This means the verification stage's `openspec list` gate always fails (no change to check), and the task-tracking benefit of openspec is lost.

## Goals / Non-Goals

**Goals:**
- Simplify ChainRunDetailView to two tabs: "Overview" (formerly DAG) and "Runs"
- Fix Runs sub-tab to fill panel width
- Surface approval mode in RunListView unified rows
- Show tool output alongside tool input in trace detail panel
- Ensure agent run templates pass openspec change context so verification's task-gate works

**Non-Goals:**
- Redesigning the chain execution model or DAG layout algorithm
- Adding new openspec schemas or changing the spec-driven workflow architecture
- Changing how LLM judge invocation works

## Decisions

### 1. Remove Timeline tab, not consolidate it into Runs

The Timeline tab shows the same data (step name, phase, duration, start time) as the Runs tab. Rather than merging them, we remove Timeline entirely. The Runs tab is already the canonical step list and also handles navigation to individual runs.

Alternatives considered: keeping Timeline with a different layout (Gantt-style). Rejected — not enough differentiation to justify maintenance burden and a third tab.

### 2. Rename "DAG" to "Overview"

"Overview" communicates the tab is the primary landing view (graph + status at a glance). "DAG" is a technical implementation detail that doesn't resonate with non-engineers.

### 3. Tool output as a collapsible section after tool input

Tool output (`toolOutput` from span metadata) is added as a new collapsible section in SpanDetail, positioned immediately after "Tool Input". Both default to collapsed to keep the panel compact. This mirrors the existing pattern used for toolInput.

### 4. OpenSpec change context injected as a prompt suffix in templates

Agent run templates will include a standard suffix instructing the agent to use openspec and specifying the change name. The `--change` flag (or equivalent env var) will be injected by the workflow so `openspec list` in the verification stage finds the right change.

The simplest enforcement path: update the `AgentRunSpec` to carry an `openspecChange` field, populate it when a run is created from a template that has openspec context, and pass it through to the verification activity.

## Risks / Trade-offs

- [Risk] Old runs without an openspecChange field will still have no task gate → Mitigation: verification falls back gracefully (skips openspec check if field is empty), so existing runs aren't broken
- [Risk] Removing Timeline tab breaks user muscle memory → Mitigation: the Runs tab already shows the same information; no behavior is lost
- [Risk] Tool output strings may be very large (e.g., bash output of thousands of lines) → Mitigation: cap display at 256 lines with a "truncated" notice; full output available in logs
