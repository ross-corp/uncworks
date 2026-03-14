## Why

Users cannot understand what agents are doing. With the spec orchestration model, a single spec run spawns a senior agent which spawns junior agents, forming a tree. Each agent produces a trace — a timeline of spans (think, edit, test, tool-call). Today there is no way to see the tree structure, no way to track which agents are running or failed, and no way to drill from a high-level orchestration view into the details of a specific agent's activity. The existing trace timeline and diff viewer are functional but unstyled and disconnected from the orchestration hierarchy.

Users need three things: (1) see how work decomposes across agents, (2) monitor live progress as agents execute, (3) inspect what changed when agents modify files. Without these, the system is a black box — you submit a spec and wait.

## What Changes

- **Orchestration graph component**: A tree/DAG visualization showing spec → senior → junior relationships. Each node displays run ID, status (Pending/Running/Succeeded/Failed/Cancelled), current activity, and elapsed time. Clicking a node drills into that run's detail panel. The graph updates in real-time as runs start, progress, and complete.
- **Enhanced trace timeline (v2)**: Horizontal timeline showing agent activity spans (think, edit, test, tool-call) with the MU-TH-UR aesthetic — phosphor glow on active spans, CRT scanline overlay, monospace labels. Clicking a tool-call span that modified files opens the diff viewer. Timeline integrates with the orchestration graph: selecting a node in the graph shows that run's timeline.
- **Diff viewer (v2)**: Monaco-based side-by-side diff editor for viewing file changes from tool-call spans. Syntax highlighting for all common languages. Accessible from both the trace timeline (click a file-modifying span) and the completion summary.
- **Live status indicators**: Phosphor pulse animation on active graph nodes. DataStream hex waterfall on nodes receiving output. Radar sweep on the root spec node while orchestration is in progress. All animations respect prefers-reduced-motion.
- **Completion summary view**: When a spec run finishes, a summary panel shows all agents' results (pass/fail), total files changed, aggregated diffs, duration breakdown, and a final status banner. Designed as the peak-end moment — visually satisfying with MU-TH-UR boot-complete aesthetic.

## Capabilities

### New Capabilities
- `orchestration-graph`: Tree/DAG component rendering spec → senior → junior run hierarchy with live status, click-to-drill, and real-time updates via WebSocket/SSE.
- `trace-timeline-v2`: Redesigned horizontal timeline with MU-TH-UR aesthetic, span detail panels, clickable diffs, and integration with the orchestration graph.
- `diff-viewer-v2`: Monaco-based side-by-side diff viewer with syntax highlighting, accessible from timeline spans and completion summary.
- `live-status-indicators`: Animated status indicators (phosphor pulse, radar sweep, data stream) on graph nodes reflecting real-time agent state.
- `completion-summary`: Aggregated results view on spec completion showing all agents, all diffs, duration breakdown, and final status.

### Modified Capabilities
- `web-event-streaming`: Extended to support graph-level events (run spawned, run status changed) in addition to per-run events.

## Impact

- `web/src/components/OrchestrationGraph.tsx` — new tree/DAG visualization component
- `web/src/components/OrchestrationNode.tsx` — individual node component with status indicators
- `web/src/components/TraceTimeline.tsx` — rewrite with MU-TH-UR aesthetic and span detail panels
- `web/src/components/DiffViewer.tsx` — new Monaco-based diff viewer component
- `web/src/components/CompletionSummary.tsx` — new aggregated results component
- `web/src/components/LiveIndicators.tsx` — phosphor pulse, radar sweep, data stream animations
- `web/src/hooks/useOrchestrationGraph.ts` — hook for graph data fetching and real-time updates
- `web/src/hooks/useTraceSpans.ts` — hook for fetching trace spans for a selected run
- `web/src/pages/SpecRunPage.tsx` — new page combining graph + timeline + detail panel
- `web/src/stores/graph-store.ts` — reactive store for orchestration graph state
- `web/src/styles/muthr.css` — MU-TH-UR design tokens (phosphor green, CRT effects, scanlines)
- `internal/server/grpc.go` — new RPC endpoints for graph data (GetSpecRunGraph, WatchSpecRunGraph)
- `proto/aot/api/v1/api.proto` — new messages and RPCs for orchestration graph
- `gen/` — regenerate from proto
