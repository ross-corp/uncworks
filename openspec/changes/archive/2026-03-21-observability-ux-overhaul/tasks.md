## 1. Backend — Rename and Span Naming

- [x] 1.1 Rename `spanPrefix()` in `gateway.go`: return "manage" for plan/verify, "implement" for execute/single
- [x] 1.2 Change tool span name from `prefix + ".tool"` to `prefix + "." + toolName` (e.g., `implement.write`)
- [x] 1.3 Add tool input summary to span metadata: file path for write, command for bash (truncated 200 chars)
- [x] 1.4 Update span type contract test to expect "manage"/"implement" prefixes instead of "unc"/"neph"

## 2. Semantic Color System

- [x] 2.1 Add `--role-manage`, `--role-implement`, `--role-system`, `--role-user`, `--role-delegate`, `--role-error` CSS custom properties to `globals.css` with light and dark mode values
- [x] 2.2 Create shared `web/src/lib/role-styles.ts` exporting `ROLE_STYLES` config (text, bg, border classes using CSS vars)
- [x] 2.3 Update ActivityFeed to use `ROLE_STYLES` for all label colors
- [x] 2.4 Rename "impl" label to "implement" in ActivityFeed `buildDisplayEntries` and `EntryRow`
- [x] 2.5 Update TraceTimeline `SPAN_TYPE_STYLES` to derive colors from role (parse span name prefix) instead of span type

## 3. Trace Detail Panel

- [x] 3.1 Install shadcn `resizable` component (`npx shadcn@latest add resizable`)
- [x] 3.2 Rewrite TraceTimeline layout: wrap waterfall + detail panel in resizable split (default 60/40)
- [x] 3.3 Move SpanDetail from inline expansion to the right resizable panel
- [x] 3.4 Remove inline expand/collapse logic (expandedSpanIds state, row height calculation for expanded)
- [x] 3.5 Add empty state to detail panel: "Click a span to view details"
- [x] 3.6 Wire diff fetching in detail panel: fetch `/traces/{spanId}/diff` on span select, show DiffViewer
- [x] 3.7 Add thinking text section: show span metadata content
- [x] 3.8 Add tool input/output section: show toolInput from metadata, formatted as code block

## 4. Waterfall Enhancements

- [x] 4.1 Add stage separator lines: detect stage change between consecutive spans, render divider
- [x] 4.2 Color waterfall bars by role (manage=blue, implement=emerald) instead of by span type
- [x] 4.3 Show span name as `role.operation` in label column

## 5. Dark/Light Mode

- [x] 5.1 Move theme toggle from RunDetailView header to Layout footer (visible on all pages)
- [x] 5.2 Remove duplicate toggle from RunDetailView
- [x] 5.3 Verify semantic colors adapt correctly in both modes

## 6. Playwright Tests

- [x] 6.1 Add `web/e2e/traces.spec.ts`: click span row, verify detail panel appears with metadata
- [x] 6.2 Add `web/e2e/activity-feed.spec.ts`: verify role labels show "implement", "manage", "system" with correct colors
- [x] 6.3 Add `web/e2e/theme-toggle.spec.ts`: verify toggle exists in footer, click switches dark/light, verify CSS class on html element
