## Context

The AOT web UI is being rebuilt from scratch as a MU-TH-UR 6000 command center. The design system migration has established the visual language (phosphor green, CRT scanlines, amber accents, Radix primitives). The spec orchestration model has established the data model (specs contain run graphs, runs contain trace spans). The graph visualization has established the rendering approach (DAG layout, animated edges, glow effects). This change composes all three into a unified interface.

The existing UI is a three-column layout: sidebar with filters, table of runs, detail panel with tabs. The new UI is a four-zone command center: left navigator, center workspace with switchable views, right detail panel, and a persistent bottom log stream.

## Goals / Non-Goals

**Goals:**
- Every interaction feels like piloting a system, not browsing a dashboard
- Spec-first navigation: users think in specs, not individual runs
- Multiple simultaneous views: see the graph AND the logs at the same time
- Zero-latency view switching within the workspace
- Keyboard-driven workflow for power users
- Responsive down to 1024px width (collapse panels, not break layout)

**Non-Goals:**
- Mobile support below 1024px — this is a workstation tool
- Offline mode — requires live connection to the backend
- Multi-user collaboration (cursors, presence) — future change
- Plugin/extension system — future change

## Decisions

### 1. CSS Grid four-zone layout

The command center uses a single CSS Grid on the root container:

```
grid-template-areas:
  "nav  workspace  detail"
  "nav  logs       logs"
grid-template-columns: 280px 1fr 320px
grid-template-rows: 1fr 200px
```

The navigator is fixed-width left. The workspace fills the center. The detail panel is fixed-width right. The log stream spans the bottom across workspace and detail. All zone boundaries are resizable via drag handles (CSS `resize` or a lightweight splitter like `react-resizable-panels`).

**Why not flexbox?** CSS Grid gives explicit 2D zone control. The log stream spanning two columns is natural in Grid, awkward in nested flex. The named areas make the responsive collapse straightforward — just redefine `grid-template-areas` at breakpoints.

**Laws of UX applied:**
- Aesthetic-Usability: the grid structure creates visual order that makes the command center feel authoritative
- Fitts's Law: each zone is a large target area; no tiny tabs to switch between major sections

### 2. Workspace view switching (center zone)

The center workspace contains exactly one active view at a time, selected via a tab bar at the top of the zone:

| View | Component | Trigger |
|------|-----------|---------|
| Graph | OrchestrationGraph | Click spec in navigator |
| Timeline | TraceTimeline | Double-click run node |
| Diff | DiffViewer | Click span in timeline |
| Files | FileExplorer | Ctrl+F or tab click |
| Shell | ShellTerminal | Ctrl+T or tab click |

Views are lazily mounted but kept alive (hidden, not unmounted) to preserve state. Switching from Graph to Files and back should not lose graph scroll position or selected node.

**Why not multiple simultaneous center views?** Split-pane in the center adds layout complexity and fights with the already four-zone layout. The bottom log stream provides the "second view" most users want alongside any workspace view. If users need side-by-side, they can use the detail panel on the right.

**Laws of UX applied:**
- Hick's Law: five views, not fifteen. Progressive disclosure — start with the graph, drill into timeline, then diff.
- Serial Position Effect: Graph (most used) is first tab; Shell (least used in normal flow) is last.

### 3. Selection model and state management

A single selection state drives the entire UI:

```typescript
interface SelectionState {
  specId: string | null;        // Selected spec in navigator
  runId: string | null;         // Selected run (node in graph or navigator)
  spanId: string | null;        // Selected span in timeline
  filePath: string | null;      // Selected file in explorer or diff
  activeView: 'graph' | 'timeline' | 'diff' | 'files' | 'shell';
}
```

Selection cascades: selecting a spec clears runId/spanId. Selecting a run clears spanId. Each zone reacts to the relevant slice of selection state:
- Navigator highlights the selected spec/run
- Workspace shows the view matching `activeView`
- Detail panel shows metadata for the most specific selection (span > run > spec)
- Log stream attaches to `runId`

State lives in a SolidJS store (extending the existing `createAgentStore`). No URL routing for internal view state — the URL reflects the selected spec/run (`/specs/:specId/runs/:runId`) but not the active workspace view or panel sizes.

**Laws of UX applied:**
- Flow: one selection drives everything. No mode confusion about what you're looking at.
- Tesler's Law: the complexity of agent orchestration (specs, runs, spans, files) is irreducible. The selection model makes it navigable without hiding it.

### 4. Navigator as spec tree

The left navigator is a tree:
```
spec-1 (3 runs: 2 passed, 1 active)
  ├── run-abc (Running ●)
  │   ├── child-run-1 (Succeeded ✓)
  │   └── child-run-2 (Running ●)
  └── run-def (Succeeded ✓)
spec-2 (1 run: 1 failed)
  └── run-ghi (Failed ✗)
```

Specs are top-level nodes fetched from the spec orchestration model. Runs nest under their spec, showing the orchestration tree (parent/child relationships). Status indicators use the MU-TH-UR palette: phosphor green for active, amber for succeeded, red glow for failed.

Clicking a spec selects it and switches the workspace to Graph view showing that spec's orchestration tree. Clicking a run selects it and shows its details in the right panel + its logs in the bottom stream.

**Why tree instead of flat list with filters?** The old sidebar had filters (status, date) over a flat list. But specs are the organizing concept now — users create specs and watch their runs. A tree maps directly to the data model. Filters can exist as a search/filter bar at the top of the navigator for large spec counts.

**Laws of UX applied:**
- Hick's Law: progressive disclosure. Start with specs (few), expand to runs (more), click into spans (many).

### 5. Log stream as persistent bottom panel

The log stream is always visible at the bottom, like a terminal in VS Code. It shows the output of the currently selected run (or the most recently active run if none selected). It uses xterm.js (already in the project) with the MU-TH-UR color palette.

The panel is resizable by dragging its top edge. It can be minimized to just the title bar (showing run ID and status) or maximized to take over the workspace area. A toggle button and Ctrl+L shortcut control visibility.

**Why always visible?** Logs are the primary feedback mechanism for agent runs. Hiding them behind a tab means users constantly switch between "what's happening" (graph) and "what's the output" (logs). The persistent bottom panel gives both simultaneously.

**Laws of UX applied:**
- Serial Position Effect: logs at the bottom — the last thing you see, the thing you check most often
- Flow: no context switch to see log output

### 6. Keyboard shortcut system

Global shortcuts registered via a custom `useKeyboardShortcuts` hook:

| Shortcut | Action |
|----------|--------|
| Ctrl+G | Switch workspace to Graph view |
| Ctrl+L | Toggle log stream (minimize/restore) |
| Ctrl+F | Switch workspace to Files view |
| Ctrl+T | Switch workspace to Shell view |
| Ctrl+K | Open command palette |
| Escape | Close command palette / deselect |
| Arrow keys | Navigate tree in navigator (when focused) |

Shortcuts are disabled when focus is inside an input/textarea/terminal to avoid conflicts. The command palette (Ctrl+K) provides fuzzy search over all actions, specs, and runs — a power-user escape hatch.

**Laws of UX applied:**
- Flow: keyboard shortcuts eliminate the friction of mouse targeting for frequent actions
- Fitts's Law: keyboard shortcuts have "infinite target size" — no aiming required

### 7. Theme integration

The MU-TH-UR design system is applied via CSS custom properties on `:root`:

```css
--mu-bg-primary: #0a0a0a;
--mu-bg-surface: #111111;
--mu-bg-elevated: #1a1a1a;
--mu-text-primary: #00ff41;
--mu-text-secondary: #00cc33;
--mu-text-muted: #338033;
--mu-accent: #ff8c00;
--mu-error: #ff0040;
--mu-glow: 0 0 10px rgba(0, 255, 65, 0.3);
--mu-scanline: repeating-linear-gradient(
  0deg,
  rgba(0, 0, 0, 0.15) 0px,
  rgba(0, 0, 0, 0.15) 1px,
  transparent 1px,
  transparent 2px
);
--mu-font-mono: 'JetBrains Mono', 'Fira Code', monospace;
```

Every component uses these variables. The scanline overlay is applied via a `::after` pseudo-element on the root container. Phosphor glow is applied to active/focused elements. Radix components are themed via their CSS variable integration.

### 8. Responsive behavior

Three breakpoints:

| Width | Layout |
|-------|--------|
| >= 1440px | Full four-zone layout |
| 1024-1439px | Navigator collapses to icon rail (expandable on hover), detail panel becomes an overlay |
| < 1024px | Not supported (show "use a wider screen" message) |

Panel collapse is handled by redefining `grid-template-areas` and `grid-template-columns` at each breakpoint. Collapsed panels have a toggle button to expand as overlays. The workspace and log stream always remain visible.

**Laws of UX applied:**
- Fitts's Law: on smaller screens, click targets stay large by collapsing panels rather than shrinking them

## Risks / Trade-offs

- **[Risk] Performance with many specs/runs in navigator tree** — Virtualize the tree with a library like `@tanstack/solid-virtual`. Only render visible nodes. Lazy-load run children on expand.
- **[Risk] View state loss on workspace switch** — Keep all views mounted but hidden (`display: none`). This uses more memory but preserves scroll position, terminal state, and graph layout.
- **[Risk] xterm.js + resizable panels** — xterm.js needs explicit `fit()` calls when its container resizes. Wire the resize observer from the splitter to call `terminal.fit()`.
- **[Risk] Keyboard shortcut conflicts** — Disable global shortcuts when focus is inside xterm.js or Monaco. Both capture their own keyboard events. Use a focus-tracking context to gate shortcut activation.
- **[Trade-off] Always-visible log stream reduces workspace height** — The default log stream height (200px) is a compromise. Users can minimize it to ~32px (title bar only) when they need more workspace.
- **[Trade-off] No split workspace** — Users who want side-by-side graph + diff must use graph in workspace + file preview in detail panel. True split-pane adds significant layout complexity for a niche use case.
