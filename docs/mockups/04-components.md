# Component Inventory

What to use off-the-shelf vs. what to build.

## Off-the-Shelf (shadcn/ui — already have)

| Component | Used For | Status |
|-----------|----------|--------|
| Button | Actions, submit, cancel | Have |
| Input | Text fields, search, filter | Have |
| Select | Model picker, repo picker | Have |
| Dialog | Confirmations, settings | Have |
| Tabs | Could use for view switching | Have |
| Badge | Status indicators | Have |
| Tooltip | Keyboard shortcut hints | Have |
| Scroll Area | Long content panels | Have |
| Separator | Section dividers | Have |
| Skeleton | Loading states | Have |
| Progress | Stage progress bar | Have |
| Popover | Inline menus | Have |
| Card | Container for content blocks | Have |
| Alert | Warnings, errors | Have |

## Off-the-Shelf (add these)

| Library | Used For | Package |
|---------|----------|---------|
| cmdk | Command palette (replace custom) | `cmdk` |
| react-markdown | Render agent markdown responses | `react-markdown` |
| rehype-highlight | Syntax highlighting in markdown | `rehype-highlight` |
| vaul | Drawer for mobile detail view | `vaul` |
| nuqs | URL state for filters/views | `nuqs` |
| sonner | Toast notifications | Already have |

## Build (custom, minimal)

| Component | What It Does | LOC Estimate |
|-----------|-------------|--------------|
| ActivityFeed | Timestamped entries: user/agent/tool/result/system | ~150 |
| ToolCallCard | Expandable tool call with input + result | ~80 |
| DiffBlock | Inline code diff (green/red lines) | ~60 |
| StageProgress | Plan → Execute → Verify progress bar | ~40 |
| CommandInput | `:` and `/` prefix command bar | ~100 |
| ChatMessage | User/agent message bubble | ~50 |
| RunStatusBadge | Status dot + text (●  running, ✓  ok, ✗  fail) | ~30 |

Total custom: ~510 lines (vs. current 6,400 in components/).

## Remove (current components that go away)

| Component | Why |
|-----------|-----|
| IconRail | Replaced by keyboard shortcuts + command bar |
| SplitPane | Replaced by full-screen view switching |
| CommandPalette (custom) | Replaced by cmdk |
| LogViewer + LogViewerInner (xterm) | Replaced by ActivityFeed |
| RunList (table) | Rebuilt as simpler k9s-style list |
| DetailPane + RunDetail | Merged into single RunDetail view |
| AgentRunForm (modal) | Replaced by inline NewRun view |
| CRT effects (fx-*) | Dropped — clean shadcn defaults |
| muthr.css | Dropped — graph uses standard styles |

## Theme

- shadcn defaults (zinc neutral, clean borders, subtle shadows)
- Dark mode as default, light mode available
- No CRT effects, no scanlines, no glow
- Monospace font for code/logs/activity, sans-serif for UI chrome
- Zero custom CSS beyond shadcn tokens (everything via Tailwind utilities)

## Router

Replace custom `useRoute()` with URL-based state:

```
/             → Run list
/new          → New run input
/run/:id      → Run detail (activity tab)
/run/:id/files → Run detail (files tab)
/run/:id/shell → Run detail (shell tab)
```

Use `nuqs` for filter state in URL params: `/?status=failed&model=qwen3`

## Comments

<!-- @unc: -->
<!-- @claude: -->
