## Context

OpenSpec specs are the source of truth for what agent runs verify against. When specs describe features that don't exist (12 themes) or use wrong paths (`/workspace/src/`), the verify phase fails or produces misleading results. These two specs were written during early design and never updated after implementation diverged.

## Goals / Non-Goals

**Goals:**
- ui-theming spec matches the actual light/dark/system implementation
- sidecar-exec spec uses correct `/workspace/<repo>/` path layout
- No other specs reference the stale `/workspace/src/` path

**Non-Goals:**
- Adding new theming features (just documenting what exists)
- Changing the actual workspace path layout

## Decisions

### Decision 1: Replace 12-theme requirement with mode toggle

The ui-theming spec will require light mode, dark mode, and a system-preference toggle. This matches the current `ThemeProvider` implementation which uses `class="dark"` on the root element.

**Rationale:** The 12-theme requirement was aspirational. Shipping 2 modes that work well is better than specifying 12 that don't exist.

### Decision 2: Use `/workspace/<repo>/` as canonical path

The sidecar mounts repos at `/workspace/<repo-name>/`. The old `/workspace/src/` was from a single-repo prototype. All path examples in specs should use the multi-repo layout.
