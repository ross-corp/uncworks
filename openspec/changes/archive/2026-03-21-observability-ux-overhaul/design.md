## Architecture

### Semantic Color System

CSS custom properties in `globals.css` that adapt to light/dark mode:

```css
:root {
  --role-manage: 217 91% 60%;      /* blue */
  --role-implement: 160 84% 39%;   /* emerald */
  --role-system: 38 92% 50%;       /* amber */
  --role-user: 215 14% 34%;        /* slate */
  --role-delegate: 263 70% 50%;    /* violet */
  --role-error: 0 84% 60%;         /* red */
}

.dark {
  --role-manage: 213 94% 68%;
  --role-implement: 160 84% 45%;
  --role-system: 38 92% 50%;
  --role-user: 215 14% 65%;
  --role-delegate: 263 70% 60%;
  --role-error: 0 84% 60%;
}
```

Usage in components via tailwind arbitrary values or a shared `roleStyles` config:

```ts
const ROLE_STYLES = {
  manage:    { text: "text-[hsl(var(--role-manage))]",    bg: "bg-[hsl(var(--role-manage)/0.1)]",    border: "border-[hsl(var(--role-manage)/0.4)]" },
  implement: { text: "text-[hsl(var(--role-implement))]", bg: "bg-[hsl(var(--role-implement)/0.1)]", border: "border-[hsl(var(--role-implement)/0.4)]" },
  system:    { text: "text-[hsl(var(--role-system))]",    bg: "bg-[hsl(var(--role-system)/0.1)]",    border: "border-[hsl(var(--role-system)/0.4)]" },
  user:      { text: "text-[hsl(var(--role-user))]",      bg: "bg-[hsl(var(--role-user)/0.1)]",      border: "border-[hsl(var(--role-user)/0.4)]" },
  delegate:  { text: "text-[hsl(var(--role-delegate))]",  bg: "bg-[hsl(var(--role-delegate)/0.1)]",  border: "border-[hsl(var(--role-delegate)/0.4)]" },
  error:     { text: "text-[hsl(var(--role-error))]",     bg: "bg-[hsl(var(--role-error)/0.1)]",     border: "border-[hsl(var(--role-error)/0.4)]" },
};
```

Shared across ActivityFeed, TraceTimeline, RunStatusBadge, and FeatureDetailView.

### Span Naming Convention

Backend (`gateway.go`) changes:

```
BEFORE                          AFTER
spanPrefix() → "unc"/"neph"    spanPrefix() → "manage"/"implement"
Name: prefix + ".tool"          Name: prefix + "." + toolName  (e.g. "implement.write")
Name: prefix + ".thought"       Name: prefix + ".thought"      (unchanged)
Name: prefix + ".started"       Name: prefix + ".started"      (unchanged)
```

The tool kind (write, bash, read, ask_user) is already captured in `activeToolSpanName`. The span name becomes `implement.write` instead of `implement.tool`.

### Trace Detail Panel Layout

Replace inline `SpanDetail` expansion with a right split panel:

```
┌───────────────────────────────────────────────────────────────┐
│ Traces · ar-ltfntp · 160 spans                               │
├────────────────────────────┬──────────────────────────────────┤
│ WATERFALL (60%)            │ DETAIL PANEL (40%)               │
│                            │                                 │
│ Click a span to see detail │ [shown when a span is selected] │
│                            │                                 │
│ Rows:                      │ Sections:                       │
│  - role color (left border)│  1. Header: name + duration     │
│  - span name (role.op)     │  2. Metadata grid (key/value)   │
│  - duration bar            │  3. Content (thinking text or   │
│  - DIFF badge              │     tool input/output)          │
│  - stage separator lines   │  4. Diff viewer (if hasDiff)    │
│                            │                                 │
├────────────────────────────┴──────────────────────────────────┤
│ ResizablePanelGroup from shadcn/ui                            │
└───────────────────────────────────────────────────────────────┘
```

Implementation: Use shadcn `ResizablePanelGroup` / `ResizablePanel` / `ResizableHandle` for the split. The detail panel renders when `selectedSpanId` is set, empty state ("Click a span") otherwise.

### Activity Feed Label Mapping

```
BEFORE          AFTER
"impl"          "implement"
"manage"        "manage" (unchanged)
"system"        "system" (unchanged)
"user"          "user" (unchanged)
"delegate"      "delegate" (unchanged)
```

Labels use `ROLE_STYLES` for consistent colors.

### Dark/Light Toggle

Move from RunDetailView into Layout footer. A small `sun/moon` button in the bottom-right of the global footer bar. Visible on every view.

```
┌────────────────────────────────────────────────────────────┐
│ [page content]                                             │
├────────────────────────────────────────────────────────────┤
│ 1 logs · 2 traces · 3 files · 4 shell      ☀/☽  i info   │
└────────────────────────────────────────────────────────────┘
```

### Stage Separator Lines

In the waterfall, when the role changes between consecutive spans (e.g., `manage.*` → `implement.*`), render a horizontal separator with the stage name:

```
  manage.thought       ████████  1.2s
  manage.write         █  5ms
  ─── EXECUTE ──────────────────────────
  implement.thought    ██████████  2.1s
  implement.write      ██  12ms
```

Detected by comparing `span.metadata.stage` between consecutive flat spans.
