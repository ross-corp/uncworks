## Why

The current UI was built incrementally as features were added — a dashboard with modals, split panes, and inline styles. It works but feels like a monitoring dashboard rather than a tool for working with AI agents. Users interact with agents through forms and raw log viewers instead of natural conversation. The 6,400 lines of custom components include duplicate code (LogsTab in two files), custom CRT effects that don't fit production use, a hand-rolled command palette, and 157 inline styles. The UI should feel like k9s meets a chat interface — keyboard-driven, resource-centric, with AI conversation as the primary interaction model.

## What Changes

- **Rewrite the frontend** from a dashboard layout to three keyboard-navigable views: run list, new run input, and run detail with live activity feed.
- **k9s-style navigation**: j/k to move, enter to drill in, esc to go back, number keys for tabs, `:` for commands, `/` for filter.
- **Chat-based run creation**: prompt input with optional "Refine with AI" chat conversation before launching. Full spec mode via tab toggle.
- **Live activity feed** replaces the raw log viewer: timestamped entries showing agent messages, tool calls with expandable inputs/results, inline code diffs, and system events.
- **Off-the-shelf components**: replace custom CommandPalette with `cmdk`, add `react-markdown` + `rehype-highlight` for agent output, `nuqs` for URL state, `vaul` for mobile drawers.
- **Drop CRT effects** entirely. Clean shadcn zinc defaults with full theme support — all shadcn themes available, light/dark toggle, preference saved to localStorage.
- **URL-based routing**: `/`, `/new`, `/run/:id`, `/run/:id/files`, etc. Replace custom `useRoute()`.
- **~510 lines of custom components** (ActivityFeed, ToolCallCard, DiffBlock, StageProgress, CommandInput, ChatMessage, RunStatusBadge) vs. 6,400 currently.
- Remove IconRail, SplitPane, custom CommandPalette, LogViewer/LogViewerInner, duplicate DetailPane, AgentRunForm modal, all CRT CSS.

## Capabilities

### New Capabilities
- `ui-views`: Three-view architecture (run list, new run, run detail) with keyboard navigation and URL routing.
- `ui-activity-feed`: Live agent activity feed with structured entries (messages, tool calls, diffs, results) replacing raw log viewer.
- `ui-chat-input`: Chat-based run creation with optional AI refinement before launch.
- `ui-theming`: Full shadcn theme support with light/dark toggle and localStorage preference persistence.

### Modified Capabilities

None — this is a frontend-only rewrite. No API or backend changes.

## Impact

- `web/src/` — Full rewrite of components/, hooks/, pages/, styles/
- `web/package.json` — Add cmdk, react-markdown, rehype-highlight, nuqs, vaul. Remove unused deps.
- `web/src/index.css` — Replace CRT effects with clean shadcn theme tokens. Support all shadcn themes.
- `web/src/styles/muthr.css` — Delete entirely
- `web/e2e/` — Update all Playwright tests for new selectors and navigation patterns
- `web/.storybook/` — Update stories for new components
- `docs/mockups/` — Reference mockups for all views (already created)
