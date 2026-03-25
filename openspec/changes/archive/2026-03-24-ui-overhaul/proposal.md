## Why

The UNCWORKS frontend has grown organically through iterative feature additions, leaving gaps in usability, consistency, and completeness. Key metrics (cost, diff stats, PR links) are computed in the trace system but never surfaced at the run list level. Jobs accumulate forever with no way to archive them. The form uses raw HTML elements instead of the 25 shadcn components already installed. The terminal and Monaco editor themes are hardcoded to dark regardless of site theme. Progressive (spec-driven) runs use a single model but the pipeline has distinct manage and implement roles that benefit from different models. These issues compound into a frontend that works but doesn't feel like a production tool.

## What Changes

- Add run list columns: total cost, +/- lines changed, clickable PR badge (with target repo), dual model display (manage/implement)
- Add archive functionality: soft-delete hides from UI, deletes associated PVC, toggle to show/hide archived, mass-select for bulk archive
- Add dual model selectors in Progressive mode: separate pickers for manage and implement agents (implement defaults to manage's model)
- Replace plain textareas with Monaco editor for prompt and spec editing (markdown syntax highlighting, theme-aware)
- Replace raw HTML selects/inputs/buttons with shadcn equivalents (Select, Input, Button, Tabs, Badge, Checkbox, DropdownMenu)
- Fix theme consistency: Monaco and xterm.js terminal follow site dark/light mode
- Add theme picker (light/dark/system selector visible in header or settings)
- Fix "Loading activity..." stuck state in run detail logs panel
- Inline status badges with feature group headers in run list
- Add cost aggregation to run status (sum of per-span costs from traces)

## Capabilities

### New Capabilities
- `archive-runs`: Archive/unarchive runs with PVC cleanup, mass-select, show/hide toggle
- `run-list-metrics`: Run list columns for cost, diff stats, PR badge, dual model display
- `monaco-editors`: Monaco editor for prompt and spec editing with markdown highlighting
- `theme-consistency`: Terminal and Monaco themes follow site dark/light mode, theme picker UI
- `dual-model-config`: Separate manage/implement model selectors for progressive runs

### Modified Capabilities
- None

## Impact

- **Modified**: `web/src/views/RunListView.tsx` — new columns, archive toggle, mass select, inline badges
- **Modified**: `web/src/views/NewRunView.tsx` — Monaco editors, dual model selectors, shadcn components
- **Modified**: `web/src/views/RunDetailView.tsx` — fix logs loading, theme-aware Monaco/terminal
- **Modified**: `web/src/views/Layout.tsx` — theme picker in header
- **Modified**: `web/src/components/ShellTerminalInner.tsx` — theme-aware xterm configuration
- **Modified**: `web/src/components/FilePreview.tsx` — dynamic Monaco theme based on site theme
- **Modified**: `web/src/components/TraceTimeline.tsx` — export cost aggregation for run-level display
- **Modified**: `web/src/types/agent-run.ts` — archive field, dual model fields, cost aggregation types
- **Modified**: `internal/server/grpc.go` — archive API endpoints, cost aggregation
- **Modified**: `internal/controller/agentrun_controller.go` — PVC cleanup on archive
- **Modified**: `deploy/crds/agentrun-crd.yaml` — archived field on status
- **Dependencies**: Monaco editor already installed. shadcn components already installed. No new deps needed.
