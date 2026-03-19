## Context

The current web frontend is a React 19 + Vite + Tailwind v4 app with 28 shadcn/ui primitives, 40 feature components (~6,400 LOC), and 15 custom hooks. It uses a dashboard layout with an icon rail, split pane, modals, and CRT visual effects. Mockups for the rewrite are at `docs/mockups/` (4 files covering run list, new run, run detail, and component inventory).

## Goals / Non-Goals

**Goals:**
- k9s-style keyboard-driven navigation (j/k, enter, esc, number keys, :, /)
- Three full-screen views: run list, new run, run detail
- Live agent activity feed with structured entries (not raw logs)
- Chat-based run creation with optional AI refinement
- Full shadcn theme support (all built-in themes, light/dark toggle, localStorage persistence)
- Off-the-shelf everything possible (cmdk, react-markdown, nuqs, vaul)
- URL-based routing for deep linking and browser history
- Under 2,000 LOC of custom components (down from 6,400)

**Non-Goals:**
- Mobile-first design (desktop keyboard-driven is primary, mobile is functional but not optimized)
- Real-time WebSocket streaming of agent output (keep SSE/polling for now, upgrade later)
- Redesigning the API or data model (frontend-only rewrite)
- Custom design system or theme builder (use shadcn's built-in themes as-is)

## Decisions

### Decision 1: Three views, not a dashboard

Replace the current IconRail + SplitPane + Modal architecture with three full-screen views:
- `/` — Run list (k9s-style table)
- `/new` — New run input (prompt/spec + optional chat)
- `/run/:id` — Run detail (activity feed + tabbed sub-views)

Navigation: keyboard-first (j/k, enter, esc, number keys) with mouse as fallback.

**Rationale:** k9s proves that a resource list → detail → action flow is more efficient than a dashboard with everything visible at once. Full-screen views give each context room to breathe.

### Decision 2: cmdk for command palette

Replace the 315-line custom CommandPalette with the `cmdk` library (2KB, widely used, accessible).

**Rationale:** cmdk is the de facto standard for ⌘K palettes in React apps. Shadcn has a cmdk-based component pattern. No reason to maintain a custom one.

### Decision 3: Activity feed as the primary run view

The default tab when viewing a run is a structured activity feed — not raw container logs. Each entry is typed:
- `user` — the original prompt
- `agent` — model text responses (rendered as markdown)
- `tool_call` — tool name + expandable input JSON
- `tool_result` — collapsible output (truncated for long results)
- `diff` — inline code diff (green/red lines) for write tool calls
- `system` — stage transitions, agent start/end

The existing `/api/v1/runs/{id}/logs/structured` endpoint provides this data. Raw logs available via tab 2.

**Rationale:** Users care about what the agent did, not sidecar stderr. The structured log endpoint was built specifically for this.

### Decision 4: Chat refinement uses the LiteLLM proxy

The "Refine with AI" feature in the new run view calls the LiteLLM proxy directly from the frontend to have a conversation about the spec before launching. This is a separate LLM call — not an agent run. The model used is the same as the selected run model.

**Rationale:** Quick, lightweight, no K8s resources needed. Just a chat completion call to help the user refine their prompt.

### Decision 5: Full shadcn theming with localStorage

Support all shadcn built-in themes. Use `next-themes` pattern (or lightweight equivalent):
- Theme preference stored in `localStorage` key `aot-theme`
- CSS variables swapped at `:root` level
- Light/dark toggle in the command palette or via keyboard shortcut
- Default: system preference, fallback to dark

**Rationale:** shadcn already defines CSS variables for every theme. Just swap the variables. No custom CSS needed.

### Decision 6: nuqs for URL state

Use `nuqs` for type-safe URL search params:
- `/?status=failed` — filter by status
- `/?model=qwen3:8b` — filter by model
- `/run/ar-ju91iv?tab=files` — deep link to specific tab

**Rationale:** URL state means bookmarkable views, browser back/forward works, and shareable links. nuqs is the standard for this in React.

### Decision 7: Incremental migration, not big-bang

Build the new views alongside the old ones behind a feature flag or route prefix. Switch over when the new views are complete and tested.

**Rationale:** Avoids breaking the working UI during development. Can A/B test both versions.

## Risks / Trade-offs

- **Losing the CRT aesthetic** — some users may prefer the distinctive visual style. Mitigated by: it's just CSS variables, a "retro" theme could be added later as a shadcn theme.
- **cmdk learning curve** — minor, well-documented library.
- **Fewer components but more complex ones** — the ActivityFeed component will be the most complex custom component. Mitigated by: clear type system from the structured log API.
- **Migration period** — two UIs coexist briefly. Mitigated by: route-based switching, old UI untouched until new is ready.
