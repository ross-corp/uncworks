## Why

The AOT web UI uses a generic custom Tailwind design system with zinc-based surface tokens (surface-0/1/2/3), Inter/JetBrains Mono fonts, yellow accent (#facc15), and standard rounded corners. It looks like every other dark-mode developer tool. Meanwhile, the homelab project has a production-grade MU-TH-UR 6000 design system with 60+ Radix UI components, CVA variants, the Phosphor color palette (amber #FFB000 / green #4AF626 on pure black), CRT visual effects, IoskeleyMono monospace font, and zero border-radius industrial brutalism. Adopting this system gives AOT a distinctive, cohesive identity that matches the broader project aesthetic, eliminates duplicate design work, and provides a far richer component library than the hand-rolled CSS classes currently in use.

## What Changes

- **Replace all design tokens**: Remove custom CSS variables (--surface-*, --text-*, --border, --accent) and tailwind color extensions. Adopt MU-TH-UR 6000 token system (--background, --foreground, --primary, --secondary, --muted, --card, --border, --ring, --destructive) with oklch color values.
- **Adopt Radix UI component library**: Replace hand-rolled `.btn`, `.input-field` CSS classes and custom JSX components with the homelab's 60+ Radix-based components (Button, Card, Badge, Table, Dialog, Sheet, Sidebar, Select, Input, Tabs, Toast, etc.) using CVA variants.
- **Apply CRT effect system**: Integrate the fx-scanlines, fx-glow, fx-flicker, fx-glitch, fx-pulse, fx-box-glow, fx-noise utility classes for terminal-authentic visual effects on panels, headings, status indicators, and interactive elements.
- **Migrate fonts**: Replace Inter (sans) and JetBrains Mono (mono) with IoskeleyMono as the sole font for both sans and mono roles, with uppercase tracking-widest body text.
- **Remove border-radius**: Set all radius tokens to 0px for the industrial brutalist aesthetic.
- **Update tailwind.config.ts**: Replace the current custom theme extension with MU-TH-UR @theme inline configuration using Tailwind v4 syntax.
- **Update package.json dependencies**: Add Radix UI primitives, class-variance-authority, clsx, tailwind-merge, tw-animate-css.

## Capabilities

### New Capabilities
- `design-tokens-migration`: Migration of all color, typography, spacing, and border-radius tokens from the custom system to the MU-TH-UR 6000 Phosphor palette and industrial design language.
- `component-library-adoption`: Replacement of all custom AOT components with the homelab Radix UI component library, including CVA variant adoption and component API migration.
- `effect-system`: Integration of the MU-TH-UR CRT visual effects (scanlines, glow, flicker, glitch, pulse, noise) into the AOT web UI with appropriate placement rules.

### Modified Capabilities
<!-- No existing specs are being modified; this is a net-new design system adoption. -->

## Impact

- **Every web component**: All 20+ components in `web/src/components/` must be rewritten to use Radix primitives and MU-TH-UR variants.
- **CSS**: `web/src/index.css` fully replaced with MU-TH-UR globals.css content. All `.btn`, `.btn-primary`, `.btn-ghost`, `.btn-danger`, `.input-field` classes removed.
- **Tailwind config**: `web/tailwind.config.ts` entirely rewritten for Tailwind v4 with @theme inline block.
- **package.json**: ~25 new Radix UI dependencies, CVA, clsx, tailwind-merge, tw-animate-css added. Inter/JetBrains Mono font imports removed.
- **Font assets**: IoskeleyMono font files must be added to the project.
- **Storybook**: `.storybook/preview.ts` and theme config updated for MU-TH-UR aesthetic.
- **E2E tests**: Visual selectors may need updating if component structure changes (data-slot attributes from Radix).
- **Affected files**: `web/src/index.css`, `web/tailwind.config.ts`, `web/package.json`, `web/src/App.tsx`, `web/src/components/*.tsx`, `web/.storybook/*`.
