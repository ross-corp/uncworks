## Why

The traces and activity feed have inconsistent naming (unc/neph vs impl/manage), no unified color system, broken inline span expansion (diffs show badge but no content, elements overlap), missing content in spans (no thinking text, no tool kind), and the dark/light toggle disappeared from most views. Engineers can't tell at a glance what the manager vs implementer agent did, can't see diffs, and can't read thinking. The UI needs to match the quality of Grafana/Datadog trace viewers.

## What Changes

- Rename all `unc`/`neph` references to `manage`/`implement` across backend spans and frontend labels
- Introduce a semantic color system: color = actor role (manage=blue, implement=emerald, system=amber, user=slate, error=red), icons/badges = action type
- Replace inline trace expansion with a right split detail panel (Grafana Tempo pattern)
- Show thinking text content in `*.thought` span details
- Show tool kind in span names: `implement.write`, `manage.bash` instead of generic `implement.tool`
- Wire diff viewer in detail panel — fetch and render diffs when span has `hasDiff`
- Move dark/light mode toggle to global Layout footer
- Add stage separator lines in the waterfall view
- Use CSS custom properties for role colors that adapt to light/dark mode
- Add Playwright tests for trace interaction, activity feed rendering, and theme toggle

## Capabilities

### New Capabilities
- `semantic-colors`: Unified CSS custom property color system for roles (manage, implement, system, user, delegate, error) that adapts to light/dark mode and is shared across activity feed, traces, and status badges
- `trace-detail-panel`: Right split panel for span details with metadata grid, thinking text, tool input/output, and diff viewer — replaces buggy inline expansion
- `trace-span-naming`: Backend span naming convention `{role}.{operation}` with tool kind included (manage.thought, implement.write, implement.bash)

### Modified Capabilities
- None

## Impact

- **Backend** (`internal/sidecar/gateway.go`): Rename `spanPrefix()` from unc/neph to manage/implement, include tool name in span name
- **Frontend** (`ActivityFeed.tsx`): Use semantic color CSS vars, rename impl→implement, manage stays
- **Frontend** (`TraceTimeline.tsx`): Replace inline expand with split panel, wire diff fetching, show content
- **Frontend** (`Layout.tsx`): Add global theme toggle
- **CSS** (`globals.css`): Add `--role-*` custom properties for light and dark modes
- **Tests** (`web/e2e/`): Playwright tests for trace click→detail, activity labels, theme toggle
