## Why

The current web UI is a flat list of runs with a detail panel. It was designed for monitoring individual agent executions, not for orchestrating autonomous coding work across specs, runs, and traces. After the design system migration (MU-TH-UR theme, Radix primitives), the spec orchestration model (specs as first-class entities with run graphs), and the graph visualization (DAG rendering, trace timelines), the UI needs a complete rebuild to become a command center for piloting agent orchestration — not a dashboard for watching it.

The existing components (AgentRunTable, AgentRunDetailPanel, Sidebar) are wired to the old flat data model and inline styles. They cannot be incrementally adapted to the new tree-structured spec/run/span navigation, multi-view workspace, or persistent log stream. A ground-up rebuild on the new design system and data model is the only path to a coherent product.

## What Changes

- **BREAKING**: Complete rewrite of `web/src/App.tsx` and all components. The existing layout (sidebar + table + detail panel) is replaced with a four-zone command center: left navigator, center workspace, right detail panel, bottom log stream.
- **SpecNavigator** replaces the flat run list. Specs are tree nodes; expanding a spec reveals its run graph. Orchestration status (active/completed/failed counts) shows inline.
- **Orchestration workspace** replaces the single detail panel. The center zone switches between five views: Graph (live orchestration DAG), Timeline (trace spans for a selected run), Diff (file changes for a selected span), Files (workspace browser), and Shell (terminal).
- **Detail panel** on the right shows contextual metadata for the selected entity — run info, span metadata, file preview — without taking over the workspace.
- **LogStream** is a persistent bottom panel (like a terminal in an IDE) showing live log output from the active or selected run. Always visible, resizable.
- **StatusBar** at the very bottom shows system health, active run count, connection status, and keyboard shortcut hints.
- **RunCreator** is redesigned using Radix Form components and the MU-TH-UR theme, with spec-aware defaults.
- **WorkspaceManager** provides preset management for saving and restoring panel layouts.
- All existing components (DiffViewer, FileExplorer, ShellTerminal, LogViewer, TraceTimeline) are reskinned with the MU-TH-UR design system (phosphor glow, CRT scanlines, amber/green palette).
- Keyboard shortcuts: Ctrl+L (logs), Ctrl+F (files), Ctrl+T (terminal), Ctrl+G (graph), Ctrl+K (command palette).
- Responsive behavior: panels collapse on smaller screens, bottom log stream becomes a toggle overlay.

## Capabilities

### New Capabilities
- `command-center-layout`: Four-zone CSS Grid layout (navigator, workspace, detail, log stream) with resizable panels and the MU-TH-UR theme applied globally.
- `spec-navigator`: Tree-based spec browser replacing the flat run list. Specs expand to show run graphs with live orchestration status.
- `run-creator-v2`: Redesigned run creation form using Radix Form components, with spec association, template presets, and MU-TH-UR styling.
- `status-bar`: Persistent bottom bar showing system health, active run count, WebSocket connection status, and keyboard shortcut hints.
- `keyboard-shortcuts`: Global keyboard shortcut system for view switching, panel toggling, and command palette access.
- `responsive-layout`: Adaptive panel behavior for smaller viewports — collapsible panels, overlay modes, and touch-friendly targets.

### Modified Capabilities

## Impact

- `web/src/App.tsx` — complete rewrite: CSS Grid command center layout, view router, keyboard shortcut provider
- `web/src/components/` — all existing components reskinned or replaced
- `web/src/components/SpecNavigator.tsx` — new: tree-based spec browser
- `web/src/components/OrchestrationGraph.tsx` — new: live DAG view in center workspace (wraps graph-visualization work)
- `web/src/components/TraceTimeline.tsx` — reskin: CRT scanlines, phosphor glow, MU-TH-UR palette
- `web/src/components/DiffViewer.tsx` — reskin: syntax highlighting with theme colors
- `web/src/components/FileExplorer.tsx` — reskin: tree + Monaco preview with MU-TH-UR theme
- `web/src/components/ShellTerminal.tsx` — reskin: xterm.js with MU-TH-UR palette
- `web/src/components/LogStream.tsx` — new: persistent bottom log panel (replaces LogViewer)
- `web/src/components/RunCreator.tsx` — new: Radix Form-based run creation (replaces AgentRunForm)
- `web/src/components/WorkspaceManager.tsx` — new: panel layout preset management
- `web/src/components/StatusBar.tsx` — new: bottom status bar
- `web/src/components/DetailPanel.tsx` — new: contextual right panel for selected entity
- `web/src/components/CommandPalette.tsx` — new: Ctrl+K command palette
- `web/src/hooks/useKeyboardShortcuts.ts` — new: global shortcut registration
- `web/src/hooks/useResponsiveLayout.ts` — new: viewport-aware panel state
- `web/src/types/` — updated types for spec tree, workspace views, panel state
- `web/src/index.css` — rewrite: CSS Grid layout, MU-TH-UR CSS custom properties, responsive breakpoints
- `web/package.json` — add dependencies: @radix-ui/react-form (if not already present), cmdk (command palette)
