## Context

The AOT platform orchestrates agent runs in a tree structure: a spec run creates a senior agent, which spawns junior agents for subtasks. Each agent produces a trace of activity spans (think, edit, test, tool-call). The web UI currently shows a flat list of runs with basic detail views. There is no visualization of the orchestration hierarchy, no way to see how work decomposes, and no way to correlate agent activity across the tree. The existing trace timeline and diff viewer work but are unstyled and disconnected from the orchestration model.

The homelab design system provides MU-TH-UR-themed components: Radar (animated sweep), DataStream (hex waterfall), TerminalBoot (typing animation). The aesthetic should feel like monitoring a fleet of autonomous systems from a command center.

## Goals / Non-Goals

**Goals:**
- Visualize the spec → senior → junior orchestration tree with live status on each node
- Click any node to drill into that run's trace timeline and details
- Redesign the trace timeline with MU-TH-UR aesthetic (phosphor glow, CRT effects)
- Provide Monaco-based diff viewing for file-modifying tool-call spans
- Stream real-time status updates to the graph via WebSocket or SSE
- Show a satisfying completion summary when a spec run finishes
- All interactions respond within 400ms (Doherty Threshold)

**Non-Goals:**
- Force-directed or physics-based graph layout (too complex, tree is sufficient)
- Collaborative editing or multi-user cursors on the graph
- Historical graph replay (playback of past orchestration runs)
- Mobile-optimized layout (desktop command-center aesthetic is primary)
- Graph persistence or export to image/PDF

## Decisions

### 1. Simple vertical tree layout (not force-directed)

The orchestration graph uses a deterministic vertical tree layout. The spec node is at the top, senior agents below, juniors below that. Each level is horizontally centered with equal spacing.

**Why not force-directed?** The spec → senior → junior hierarchy is always a tree (not a general DAG). Force-directed layout adds complexity (spring simulation, stabilization) and produces non-deterministic positions that shift as nodes are added. A static tree layout is predictable, fast to compute, and easy to animate when new nodes appear.

**Layout algorithm:** Recursive top-down placement. Each node's x-position is the center of its children's extent. Leaf nodes are placed left-to-right with fixed spacing. This runs in O(n) and handles up to ~50 nodes (practical max for a spec run) without performance issues. Implemented as a pure function: `layoutTree(rootNode) → Map<nodeId, {x, y}>`.

### 2. Click node → detail panel slides in from right

Clicking a node in the graph opens a detail panel on the right side (40% width) showing that run's trace timeline, current output, and metadata. The graph shrinks to 60% width to accommodate. The panel uses the same RunDetailPage content but embedded inline.

**Why not navigate to a separate page?** Navigation breaks the spatial context. Users need to see the graph while inspecting a node's details — e.g., to understand why a junior failed while the senior continued. The split-panel pattern preserves both views.

**Transition:** Panel slides in with a 200ms ease-out animation. Graph nodes reflow to fit the reduced width with a matching 200ms transition. Active node gets a highlighted border (phosphor green glow).

### 3. Trace timeline redesign with MU-TH-UR aesthetic

The trace timeline is a horizontal bar chart where each span is a colored rectangle. Spans are laid out left-to-right by start time, with height indicating nesting depth.

**Visual treatment:**
- Background: dark (#0a0a0a) with faint CRT scanline overlay (repeating 2px gradient, 0.03 opacity)
- Active spans: phosphor green (#00ff41) with 0.6 opacity glow (box-shadow: 0 0 8px)
- Completed spans: dim green (#1a3a1a) with no glow
- Failed spans: amber (#ff6600) with pulse animation
- Span labels: monospace (JetBrains Mono), 11px, uppercase
- Hover: tooltip with span name, duration, and file list (if tool-call)
- Click on tool-call span: opens diff viewer below the timeline

**Performance:** Spans are virtualized — only visible spans are rendered as DOM elements. The timeline canvas handles up to 1000 spans per run without jank. Scroll position is preserved on updates.

### 4. Monaco diff editor for file changes

Tool-call spans that modify files store before/after file content. The diff viewer uses Monaco Editor's built-in diff mode (`monaco.editor.createDiffEditor`) for side-by-side comparison with syntax highlighting.

**Why Monaco over custom renderer?** Monaco handles syntax highlighting for 50+ languages, inline diff decorations, minimap navigation, and code folding out of the box. It's already a common dependency in web-based code tools. A custom renderer would need to reimplement all of this.

**Why not unified diff?** Side-by-side is more readable for code review. Monaco's diff editor shows both versions with aligned line numbers and change highlights. Users can switch to inline mode via a toggle if preferred.

**Loading strategy:** Monaco is loaded lazily (dynamic import) when the diff viewer first opens. Bundle size for Monaco is ~2MB, but it's only loaded on demand. A loading skeleton with CRT-style "LOADING DIFF..." text shows during the load.

### 5. Live updates via SSE (not raw WebSocket)

The graph receives real-time updates via Server-Sent Events (SSE). A new `WatchSpecRunGraph` RPC streams events: `node_added` (new agent spawned), `node_status_changed` (phase transition), `node_progress` (current activity text).

**Why SSE over WebSocket?** SSE is simpler — unidirectional (server → client), auto-reconnects natively, works through HTTP/2 multiplexing, and the existing ConnectRPC/gRPC-Web streaming already uses SSE semantics. No need for a separate WebSocket server.

**Event schema:**
- `node_added`: `{run_id, parent_run_id, spec_name, agent_type}`
- `node_status_changed`: `{run_id, phase, message}`
- `node_progress`: `{run_id, current_activity, elapsed_seconds}`

The client maintains a reactive graph store. Each event triggers a targeted update — not a full graph refetch. This keeps updates under 16ms (one frame).

### 6. Graph and timeline are two views of the same data

The graph is spatial (who spawned whom). The timeline is temporal (what happened when). Both reference the same underlying run data. Selecting a node in the graph filters the timeline to that run's spans. The timeline header shows which run is selected.

A future "unified timeline" could overlay all runs' spans on one timeline (senior and junior spans interleaved by wall-clock time), but that's out of scope. For now, one run's timeline at a time.

### 7. Completion summary as the peak-end moment

When all runs in a spec complete, a completion summary panel replaces the graph view. It shows:
- Status banner: "SPEC COMPLETE — ALL SYSTEMS NOMINAL" (or "SPEC COMPLETE — FAILURES DETECTED") with TerminalBoot typing animation
- Agent result table: each agent's final status, duration, files changed count
- Aggregated diff: expandable list of all modified files across all agents, each clickable to open in the diff viewer
- Duration breakdown: bar chart showing parallel and sequential time across agents

The summary is the last thing users see — per the Peak-End Rule, it should be the most polished and satisfying interaction. The TerminalBoot animation typing out the final status gives a sense of completion.

### 8. Laws of UX applied

- **Aesthetic-Usability Effect:** The MU-TH-UR aesthetic (phosphor green, CRT scanlines, monospace type) makes the monitoring interface feel purpose-built and trustworthy — like a real command center rather than a generic dashboard.
- **Doherty Threshold:** All interactions respond in <400ms. Graph node clicks show the detail panel instantly (optimistic, data streams in). Timeline span clicks show diff viewer skeleton immediately, Monaco loads lazily behind it.
- **Von Restorff Effect:** Active nodes glow green. Failed nodes pulse amber. Cancelled nodes dim out. The active/failed states are immediately distinguishable from the sea of completed nodes.
- **Law of Proximity:** In the graph, parent-child nodes are visually close. Sibling juniors are grouped horizontally. In the timeline, related spans (think → edit → test within one task) are adjacent.
- **Chunking:** Timeline spans are natural chunks of agent activity. The graph groups agents by level (spec → senior → junior). The completion summary groups results by agent.
- **Peak-End Rule:** The completion summary is designed as the peak moment. TerminalBoot typing animation, full results table, all diffs accessible — this is the payoff for watching agents work.

## Risks / Trade-offs

- **[Risk] Monaco bundle size (2MB)** → Mitigated by lazy loading. Only loaded when user opens a diff. Could further reduce by loading only needed language grammars, but premature optimization.
- **[Risk] Large orchestration trees (50+ nodes)** → The tree layout handles this, but the graph may need horizontal scrolling. Add zoom controls (fit-to-view button) if trees exceed viewport.
- **[Risk] SSE connection limits** → Browsers limit to 6 concurrent SSE connections per domain on HTTP/1.1. The system uses HTTP/2 (multiplexed), so this is not a concern. If HTTP/1.1 fallback is needed, consolidate to a single SSE stream per spec run.
- **[Risk] Timeline span count (1000+ spans)** → Mitigated by virtualization. Only visible spans are rendered. If a run produces >5000 spans, add a "show more" pagination.
- **[Risk] CRT aesthetic may reduce readability** → Scanline opacity is kept at 0.03 (barely visible). Phosphor glow is on interactive elements only, not body text. All text meets WCAG AA contrast ratios against the dark background.
- **[Trade-off] Split-panel vs. full-page navigation** → Split-panel is more complex to implement (responsive layout, panel transitions) but preserves spatial context. Worth the extra effort.
