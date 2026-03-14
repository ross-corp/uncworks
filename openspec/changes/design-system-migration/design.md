## Context

The AOT web UI (`web/`) is a React + Vite + Tailwind CSS v3 application with ~20 custom components using hand-rolled CSS utility classes (`.btn`, `.input-field`) and custom CSS variables for theming. The homelab project (`/home/tristan/Documents/Github/rosh-bullshit/homelab/apps/lib/`) contains two packages relevant to this migration:

1. **`@homelab/design-tokens`** (`homelab/apps/lib/design-tokens/`): CSS custom properties (`base.css`), Tailwind v4 globals (`globals.css`), and TypeScript token exports (`colors.ts`, `typography.ts`, `spacing.ts`).
2. **`@homelab/ui-components`** (`homelab/apps/lib/ui-components/`): 60+ Radix UI components with CVA variants, all styled for the MU-TH-UR 6000 terminal aesthetic.

The AOT app currently uses Tailwind v3. The homelab design system uses Tailwind v4 with `@theme inline` blocks and `@import 'tailwindcss'` syntax. This migration must account for this version difference.

### Current AOT Token System
| AOT Token | CSS Variable | Value |
|---|---|---|
| surface-0 | --surface-0 | #09090b |
| surface-1 | --surface-1 | #18181b |
| surface-2 | --surface-2 | #27272a |
| surface-3 | --surface-3 | #3f3f46 |
| edge | --border | #27272a |
| txt-primary | --text-primary | #fafafa |
| txt-secondary | --text-secondary | #a1a1aa |
| txt-tertiary | --text-tertiary | #71717a |
| accent | --accent | #facc15 |
| font-sans | Inter | system sans-serif |
| font-mono | JetBrains Mono | monospace |

### Target MU-TH-UR 6000 Token System
| MU-TH-UR Token | CSS Variable | Value |
|---|---|---|
| background | --background | oklch(0 0 0) — pure black |
| foreground | --foreground | oklch(0.85 0.15 75) — amber |
| primary | --primary | oklch(0.85 0.15 75) — amber |
| secondary | --secondary | oklch(0.80 0.18 145) — green |
| card | --card | oklch(0.05 0 0) |
| border | --border | oklch(0.3 0 0) |
| muted | --muted | oklch(0.15 0 0) |
| destructive | --destructive | oklch(0.55 0.22 25) |
| font-sans | IoskeleyMono | monospace |
| font-mono | IoskeleyMono | monospace |
| radius | --radius | 0px |

## Goals / Non-Goals

**Goals:**
- Fully adopt the MU-TH-UR 6000 design system so the AOT web UI is visually indistinguishable from other homelab UIs.
- Replace all custom CSS component classes with Radix UI primitives + CVA variants.
- Integrate CRT effects (scanlines, glow, flicker) for an authentic terminal aesthetic.
- Upgrade from Tailwind v3 to Tailwind v4 as part of the migration.
- Maintain all existing functionality; this is a visual-only change with no business logic modifications.

**Non-Goals:**
- Publishing the design tokens or component library as an npm package. Components are copied directly.
- Adding new features or pages to the AOT web UI.
- Modifying the homelab design system itself.
- Supporting light mode. MU-TH-UR is dark-only by design.
- Migrating backend code or protobuf definitions.

## Decisions

### D1: Component integration strategy — Copy into project

**Decision**: Copy the homelab `ui-components/src/components/ui/` directory into `web/src/components/ui/` and the design-tokens CSS files into `web/src/styles/`. Do NOT use npm workspaces, git submodules, or monorepo linking.

**Rationale**: The two repos (uncworks, homelab) are separate git repositories with no shared package manager workspace. Creating a cross-repo dependency adds build complexity, version coupling, and CI/CD entanglement. Copying gives full ownership and the ability to diverge if needed. The component library is stable and changes infrequently.

**Alternatives considered**:
- *npm workspace link*: Requires monorepo restructuring across two separate git repos. Rejected.
- *Git submodule*: Adds operational complexity for a one-time adoption. Rejected.
- *npm publish from homelab*: Adds CI/CD pipeline and versioning overhead for a single consumer. Overkill.

### D2: Tailwind version — Upgrade to v4

**Decision**: Upgrade from Tailwind v3 to Tailwind v4 as part of this migration.

**Rationale**: The MU-TH-UR globals.css uses Tailwind v4 syntax (`@import 'tailwindcss'`, `@theme inline`, `@custom-variant`). Backporting to v3 would require rewriting the entire token system. Upgrading now avoids maintaining a compatibility layer.

**Alternatives considered**:
- *Stay on v3 and rewrite tokens*: Doubles the work and creates drift from the source design system. Rejected.

### D3: Token mapping strategy — Direct replacement with MU-TH-UR variables

**Decision**: Replace all AOT CSS variables with MU-TH-UR variables. Provide no backwards-compatibility aliases.

**Token mapping**:
| AOT Usage | Replacement |
|---|---|
| `bg-surface-0` | `bg-background` |
| `bg-surface-1` | `bg-card` |
| `bg-surface-2` | `bg-muted` |
| `bg-surface-3` | `bg-muted` (darker variant via opacity) |
| `text-txt-primary` | `text-foreground` |
| `text-txt-secondary` | `text-muted-foreground` |
| `text-txt-tertiary` | `text-muted-foreground/50` |
| `border-edge` | `border-border` |
| `border-edge-strong` | `border-primary` |
| `bg-accent` / `text-accent` | `bg-primary` / `text-primary` |
| `text-danger` | `text-destructive` |
| `text-success` | `text-secondary` (green) |
| `text-warning` | `text-primary` (amber) |
| `text-info` | `text-secondary` |
| `font-sans` | `font-sans` (now IoskeleyMono) |
| `font-mono` | `font-mono` (IoskeleyMono) |
| `rounded` | Removed (all 0px) |

### D4: Component replacement mapping

**Decision**: Map each AOT component to its MU-TH-UR equivalent:

| AOT Component | MU-TH-UR Replacement | Notes |
|---|---|---|
| `.btn` / `.btn-primary` classes | `<Button variant="terminal">` | Default variant is terminal |
| `.btn-ghost` | `<Button variant="ghost">` | |
| `.btn-danger` | `<Button variant="destructive">` | |
| `.input-field` class | `<Input>` component | |
| `StatusBadge.tsx` | `<Badge>` with variant | Map status to variant colors |
| `AgentRunTable.tsx` | `<Table>` components | Use Table, TableHeader, TableRow, TableCell |
| `ConfirmDialog.tsx` | `<AlertDialog>` | Use AlertDialog from Radix |
| `Toast.tsx` | `<Toaster>` + `useToast` | Sonner-based toast system |
| `Skeleton.tsx` | `<Skeleton>` | Direct replacement |
| `Sidebar.tsx` | `<Sidebar>` | Use MU-TH-UR Sidebar component |
| `Layout.tsx` | Restructure with `<Sidebar>` + panels | |
| `AgentRunDetailPanel.tsx` | `<Sheet>` or `<Card>` | Side panel pattern |
| `AgentRunForm.tsx` | `<Card>` + `<Input>` + `<Select>` + `<Button>` | Form composition |
| `DiffViewer.tsx` | `<Card>` + `<ScrollArea>` | Wrap existing diff logic |
| `EventsView.tsx` | `<Table>` + `<Badge>` | Event list |
| `FileExplorer.tsx` | `<Accordion>` or `<Collapsible>` + tree | |
| `FileTree.tsx` | `<Collapsible>` tree | |
| `FilePreview.tsx` | `<Card>` + `<ScrollArea>` | |
| `GitHubModal.tsx` | `<Dialog>` | |
| `LogViewer.tsx` / `LogViewerInner.tsx` | `<Card>` + `<ScrollArea>` with fx-scanlines | Terminal output |
| `ShellTerminal.tsx` / `ShellTerminalInner.tsx` | `<Card>` wrapper with fx-scanlines + fx-flicker | xterm.js container |
| `SpecEditor.tsx` | `<Card>` + `<Tabs>` | Monaco editor wrapper |
| `TraceTimeline.tsx` | `<Card>` + custom timeline | |
| `WorkspaceEditor.tsx` | `<Card>` + `<Input>` + `<Select>` | |
| `ReposView.tsx` | `<Table>` + `<Badge>` | |
| `ErrorBoundary.tsx` | `<Alert>` with destructive variant | |

### D5: Font delivery — Self-hosted IoskeleyMono

**Decision**: Bundle IoskeleyMono font files in `web/public/fonts/` and load via `@font-face` in CSS. Remove the Google Fonts import for Inter and JetBrains Mono.

**Rationale**: IoskeleyMono is not available on Google Fonts. Self-hosting ensures availability without external CDN dependency.

### D6: CRT effect placement rules

**Decision**: Apply effects systematically based on element type:
- `fx-scanlines`: Main content panels, Card components, terminal outputs, log viewers.
- `fx-glow`: Primary headings (h1, h2), active nav items, status indicators.
- `fx-flicker`: Root body or main app wrapper for ambient terminal feel.
- `fx-glitch`: Error states, destructive action confirmations.
- `fx-pulse`: Active/running status badges, loading indicators.
- `fx-box-glow`: Focused input fields, selected cards, active sidebar items.
- `fx-noise`: Background panels for texture.

## Risks / Trade-offs

- **[Tailwind v3 to v4 breaking changes]** -> Mitigated by doing a full rewrite of the CSS file rather than incremental migration. The `@theme inline` block replaces `tailwind.config.ts` theme extension entirely.
- **[Component API drift]** -> Copied components may diverge from the homelab source over time. Mitigated by keeping the copy as close to upstream as possible and documenting the copy date.
- **[xterm.js + CRT effects conflict]** -> The `fx-scanlines` pseudo-element overlay may interfere with xterm.js canvas rendering. Mitigated by applying effects to the container wrapper, not the xterm element itself, and testing thoroughly.
- **[Monaco editor theme mismatch]** -> Monaco has its own theming system. A custom MU-TH-UR Monaco theme must be created to match the amber/green phosphor palette. This is additional work but essential for consistency.
- **[E2E test breakage]** -> Component structure changes (divs becoming Radix primitives with data-slot attributes) may break CSS selectors in Playwright tests. All E2E tests must be updated.
- **[Font rendering]** -> IoskeleyMono with uppercase + tracking-widest may reduce information density compared to Inter. Monitor readability in data-heavy views (tables, logs).
- **[Performance]** -> CRT animations (fx-flicker, fx-pulse) run continuously. Mitigated by respecting `prefers-reduced-motion` (already handled in globals.css) and limiting animation to non-data-critical elements.

## Migration Plan

1. **Phase 1 — Foundation**: Install dependencies, copy design tokens, replace index.css, update tailwind config, add fonts. App will look broken but build.
2. **Phase 2 — Component library**: Copy ui-components into project, set up the `cn()` utility, verify components render in isolation (Storybook).
3. **Phase 3 — Component migration**: Replace each AOT component one-by-one, starting with Layout/Sidebar (structural), then forms/inputs, then tables/lists, then modals/dialogs, then status indicators.
4. **Phase 4 — Effects**: Apply fx-* classes to appropriate elements per D6 placement rules.
5. **Phase 5 — Polish**: Create Monaco theme, update Storybook theme, fix E2E tests, visual QA.

**Rollback**: Git revert. All changes are in the web/ directory. No backend changes, no database migrations, no API changes.

## Open Questions

- **Q1**: Should the IoskeleyMono font files be committed to the repo or fetched from a shared asset location? (Tentative: commit to repo for simplicity.)
- **Q2**: Is there an existing Monaco editor theme in the homelab project that can be reused, or must one be created from scratch?
- **Q3**: Should Storybook stories be updated in this change or deferred to a follow-up?
