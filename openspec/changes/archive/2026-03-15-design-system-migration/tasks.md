## 1. Foundation — Dependencies and Build Setup

- [x] 1.1 Install Radix UI dependencies: @radix-ui/react-slot, @radix-ui/react-dialog, @radix-ui/react-alert-dialog, @radix-ui/react-select, @radix-ui/react-scroll-area, @radix-ui/react-tabs, @radix-ui/react-separator, @radix-ui/react-label, @radix-ui/react-popover, @radix-ui/react-dropdown-menu, @radix-ui/react-collapsible, @radix-ui/react-accordion, @radix-ui/react-progress, @radix-ui/react-checkbox, @radix-ui/react-tooltip, @radix-ui/react-switch, @radix-ui/react-toggle, @radix-ui/react-toggle-group, @radix-ui/react-avatar, @radix-ui/react-hover-card, @radix-ui/react-menubar, @radix-ui/react-navigation-menu, @radix-ui/react-radio-group, @radix-ui/react-slider, @radix-ui/react-context-menu, @radix-ui/react-aspect-ratio
- [x] 1.2 Install styling dependencies: class-variance-authority, clsx, tailwind-merge, tw-animate-css
- [x] 1.3 Upgrade Tailwind CSS from v3 to v4: update tailwindcss package, update postcss config if needed, update vite config for Tailwind v4 plugin
- [x] 1.4 Install sonner (toast library used by MU-TH-UR toaster component)
- [x] 1.5 Verify all dependencies install without peer dependency conflicts by running `npm install`

## 2. Font Migration

- [x] 2.1 Obtain IoskeleyMono font files (.woff2, .woff) from the homelab project or font source
- [x] 2.2 Create `web/public/fonts/` directory and copy IoskeleyMono font files into it
- [x] 2.3 Add @font-face declarations for IoskeleyMono in the CSS (weight 400, 500, 600, 700 if available)
- [x] 2.4 Remove the Google Fonts @import for Inter and JetBrains Mono from web/src/index.css
- [x] 2.5 Verify IoskeleyMono renders correctly in the browser dev tools

## 3. Design Tokens Migration

- [x] 3.1 Replace web/src/index.css entirely: remove all :root custom properties (--surface-*, --text-*, --border*, --accent*, --danger, --warning, --success, --info) and replace with MU-TH-UR :root block containing --background, --foreground, --card, --card-foreground, --popover, --popover-foreground, --primary, --primary-foreground, --secondary, --secondary-foreground, --muted, --muted-foreground, --accent, --accent-foreground, --destructive, --destructive-foreground, --border, --input, --ring, --radius, --glow-primary, --glow-secondary using oklch values
- [x] 3.2 Add @theme inline block to CSS with Tailwind v4 token mappings: --font-sans, --font-mono, --color-background, --color-foreground, --color-primary, --color-primary-foreground, --color-secondary, --color-secondary-foreground, --color-muted, --color-muted-foreground, --color-accent, --color-accent-foreground, --color-destructive, --color-destructive-foreground, --color-border, --color-input, --color-ring, --radius-sm/md/lg/xl all set to 0px
- [x] 3.3 Add @layer base block: set * to border-border and rounded-none, set body to bg-background text-foreground uppercase tracking-widest font-mono
- [x] 3.4 Remove the @layer components block containing .input-field, .btn, .btn-primary, .btn-ghost, .btn-danger, .scrollbar-hidden
- [x] 3.5 Add CRT effects layer (@layer utilities): fx-scanlines, fx-noise, fx-glow, fx-glow-amber, fx-glow-danger, fx-box-glow, fx-box-glow-warning, fx-box-glow-danger, fx-panel
- [x] 3.6 Add CRT animation keyframes: fx-flicker (6s), fx-glitch (4s), fx-pulse-glow (3s), fx-border-flash (1s), fx-text-flicker (0.1s) — all wrapped in @media (prefers-reduced-motion: no-preference)
- [x] 3.7 Add MU-TH-UR scrollbar styles: ::-webkit-scrollbar (6px), ::-webkit-scrollbar-track (background + border), ::-webkit-scrollbar-thumb (muted, 0px radius, amber on hover)
- [x] 3.8 Replace or rewrite web/tailwind.config.ts: remove all theme.extend.colors (surface, edge, txt, accent, danger, warning, success, info) and theme.extend.fontFamily. For Tailwind v4, this file may become minimal or replaced by the @theme inline block in CSS

## 4. Component Library Setup

- [x] 4.1 Create `web/src/lib/utils.ts` with the cn() utility function: import clsx and twMerge, export cn(...inputs) that calls twMerge(clsx(inputs))
- [x] 4.2 Create `web/src/components/ui/` directory for MU-TH-UR components
- [x] 4.3 Copy button.tsx from homelab/apps/lib/ui-components/src/components/ui/button.tsx to web/src/components/ui/button.tsx, update import path for cn() to ../../lib/utils
- [x] 4.4 Copy card.tsx from homelab ui-components to web/src/components/ui/card.tsx, update cn() import
- [x] 4.5 Copy badge.tsx from homelab ui-components to web/src/components/ui/badge.tsx, update cn() import
- [x] 4.6 Copy table.tsx from homelab ui-components to web/src/components/ui/table.tsx, update cn() import
- [x] 4.7 Copy dialog.tsx from homelab ui-components to web/src/components/ui/dialog.tsx, update cn() import
- [x] 4.8 Copy alert-dialog.tsx from homelab ui-components to web/src/components/ui/alert-dialog.tsx, update cn() import
- [x] 4.9 Copy sheet.tsx from homelab ui-components to web/src/components/ui/sheet.tsx, update cn() import
- [x] 4.10 Copy sidebar.tsx from homelab ui-components to web/src/components/ui/sidebar.tsx, update cn() import
- [x] 4.11 Copy input.tsx from homelab ui-components to web/src/components/ui/input.tsx, update cn() import
- [x] 4.12 Copy select.tsx from homelab ui-components to web/src/components/ui/select.tsx, update cn() import
- [x] 4.13 Copy label.tsx from homelab ui-components to web/src/components/ui/label.tsx, update cn() import
- [x] 4.14 Copy tabs.tsx from homelab ui-components to web/src/components/ui/tabs.tsx, update cn() import
- [x] 4.15 Copy scroll-area.tsx from homelab ui-components to web/src/components/ui/scroll-area.tsx, update cn() import
- [x] 4.16 Copy skeleton.tsx from homelab ui-components to web/src/components/ui/skeleton.tsx, update cn() import
- [x] 4.17 Copy toast.tsx, toaster.tsx, and use-toast.ts from homelab ui-components to web/src/components/ui/, update cn() imports
- [x] 4.18 Copy sonner.tsx from homelab ui-components to web/src/components/ui/sonner.tsx, update cn() import
- [x] 4.19 Copy separator.tsx from homelab ui-components to web/src/components/ui/separator.tsx, update cn() import
- [x] 4.20 Copy tooltip.tsx from homelab ui-components to web/src/components/ui/tooltip.tsx, update cn() import
- [x] 4.21 Copy collapsible.tsx from homelab ui-components to web/src/components/ui/collapsible.tsx, update cn() import
- [x] 4.22 Copy accordion.tsx from homelab ui-components to web/src/components/ui/accordion.tsx, update cn() import
- [x] 4.23 Copy progress.tsx from homelab ui-components to web/src/components/ui/progress.tsx, update cn() import
- [x] 4.24 Copy dropdown-menu.tsx from homelab ui-components to web/src/components/ui/dropdown-menu.tsx, update cn() import
- [x] 4.25 Copy popover.tsx from homelab ui-components to web/src/components/ui/popover.tsx, update cn() import
- [x] 4.26 Copy checkbox.tsx from homelab ui-components to web/src/components/ui/checkbox.tsx, update cn() import
- [x] 4.27 Copy alert.tsx from homelab ui-components to web/src/components/ui/alert.tsx, update cn() import
- [x] 4.28 Copy spinner.tsx from homelab ui-components to web/src/components/ui/spinner.tsx, update cn() import
- [x] 4.29 Copy empty.tsx from homelab ui-components to web/src/components/ui/empty.tsx, update cn() import
- [x] 4.30 Verify all copied components compile without TypeScript errors by running `tsc --noEmit`

## 5. Component Migration — Structural Components

- [x] 5.1 Rewrite Layout.tsx: replaced header styling with MU-TH-UR tokens (border-border, text-foreground, fx-glow on headings), replaced input-field with Input component, replaced btn-primary with Button component, added fx-scanlines to main content area.
- [x] 5.2 Rewrite Sidebar.tsx: replaced all surface/txt/edge classes with MU-TH-UR equivalents (bg-background, bg-muted, bg-card, text-foreground, text-muted-foreground, border-border). Applied fx-glow to active nav items and title. Added tracking-widest to section headers.
- [x] 5.3 Rewrite App.tsx: added fx-flicker class to the outermost div wrapping the entire app.

## 6. Component Migration — Data Display

- [x] 6.1 Rewrite AgentRunTable.tsx: replaced all surface/txt/edge classes with MU-TH-UR tokens. Replaced btn-ghost/btn-primary with Button component. Used Badge for spec indicator. Action menu uses bg-card, bg-muted, text-foreground, text-muted-foreground, text-destructive.
- [x] 6.2 Rewrite StatusBadge.tsx: replaced custom badge spans with MU-TH-UR Badge component. Mapped: running->secondary+animate-pulse, succeeded->default+secondary colors, failed->destructive+fx-glitch, pending->outline+primary, cancelled->secondary+opacity-50. Backend and ModelTier badges also migrated.
- [x] 6.3 Rewrite EventsView.tsx: replaced all surface/txt/edge classes with MU-TH-UR tokens. Added fx-scanlines to the events container.
- [x] 6.4 Rewrite TraceTimeline.tsx: replaced surface/txt classes with MU-TH-UR tokens. Applied fx-scanlines to timeline container. Used Badge for diff indicator. Applied fx-glow to selected span.
- [x] 6.5 Rewrite ReposView.tsx: replaced input-field with Input component, btn-primary/btn-ghost with Button component. All colors use MU-TH-UR tokens.

## 7. Component Migration — Detail Panels and Editors

- [x] 7.1 Rewrite AgentRunDetailPanel.tsx: replaced all surface/txt/edge classes with MU-TH-UR tokens. Applied fx-scanlines to detail panel container. Used Button component for all actions. Used fx-glow on detail name heading. Tab bar uses text-foreground/text-muted-foreground with bg-primary active indicator.
- [x] 7.2 Rewrite SpecEditor.tsx: replaced surface/txt classes with MU-TH-UR tokens (border-border, bg-card). Updated font to IoskeleyMono.
- [x] 7.3 Rewrite WorkspaceEditor.tsx: replaced all input-field with Input component, all btn-primary/btn-ghost/btn-danger with Button component variants. All colors use MU-TH-UR tokens.
- [x] 7.4 Rewrite DiffViewer.tsx: replaced all color classes with MU-TH-UR tokens (text-secondary for additions, text-destructive for deletions, text-primary for hunks, bg-muted for headers, bg-background for content).

## 8. Component Migration — Log and Terminal Views

- [x] 8.1 Rewrite LogViewer.tsx: replaced surface/txt classes with MU-TH-UR tokens. Applied fx-scanlines and fx-noise to the log container.
- [x] 8.2 Rewrite LogViewerInner.tsx: updated xterm theme to pure black background (#000000) with amber foreground (#FFB000). Updated font to IoskeleyMono.
- [x] 8.3 Rewrite ShellTerminal.tsx: replaced surface/txt classes with MU-TH-UR tokens. Applied fx-scanlines to the terminal wrapper div.
- [x] 8.4 Rewrite ShellTerminalInner.tsx: updated xterm theme to pure black background with amber foreground. Updated font to IoskeleyMono. Status bar uses bg-muted, text-muted-foreground/60. Status colors use text-primary (connecting), text-secondary (connected), text-destructive (disconnected).

## 9. Component Migration — File Browser

- [x] 9.1 Rewrite FileExplorer.tsx: replaced all surface/txt classes with MU-TH-UR tokens (bg-background, bg-card, border-border, text-muted-foreground/60, text-destructive).
- [x] 9.2 Rewrite FileTree.tsx: replaced all surface/txt classes with MU-TH-UR tokens (text-muted-foreground, hover:bg-muted, text-muted-foreground/60).
- [x] 9.3 Rewrite FilePreview.tsx: replaced all surface/txt classes with MU-TH-UR tokens (border-border, bg-muted, text-muted-foreground/60). Updated font to IoskeleyMono.

## 10. Component Migration — Forms and Dialogs

- [x] 10.1 Rewrite AgentRunForm.tsx: replaced all input-field with Input component, all btn-primary/btn-ghost with Button component. Form container uses bg-card, border-border. Labels use text-muted-foreground. Select elements use inline MU-TH-UR styled classes. Heading uses fx-glow.
- [x] 10.2 Rewrite ConfirmDialog.tsx: replaced all surface/txt/edge classes with MU-TH-UR tokens. Used Button variant="ghost" for cancel, Button variant="destructive" for confirm. Applied fx-glitch to the dialog container and fx-glow to heading.
- [x] 10.3 Rewrite GitHubModal.tsx: replaced all input-field with Input component, all btn-primary/btn-ghost with Button component. All colors use MU-TH-UR tokens. Heading uses fx-glow.
- [x] 10.4 Rewrite Toast.tsx: replaced all color references with MU-TH-UR tokens (text-secondary/border-secondary for success, text-destructive/border-destructive for error, text-primary/border-primary for info). Removed rounded-lg (radius is 0px globally). Kept existing ToastProvider/useToast API for backward compatibility.

## 11. Component Migration — Utility Components

- [x] 11.1 Rewrite Skeleton.tsx: replaced bg-surface-2 with bg-muted. Replaced border-edge with border-border. Replaced bg-surface-0 with bg-background in SkeletonDetail. Removed rounded class (globally 0px).
- [x] 11.2 Rewrite ErrorBoundary.tsx: replaced btn-ghost with inline MU-TH-UR ghost button classes. Applied fx-glitch to the error container and fx-glow to heading. Used border-destructive/30 bg-destructive/5 for error styling.

## 12. Storybook Updates

- [x] 12.1 Update web/.storybook/preview.ts: kept import of index.css (which now contains MU-TH-UR globals). Added dark background decorator with mu-th-ur black (#000000).
- [x] 12.2 Update all .stories.tsx files to reflect the new component APIs: Updated Layout.stories.tsx (replaced surface/txt/edge classes with MU-TH-UR equivalents), ErrorBoundary.stories.tsx (replaced text-txt-secondary with text-muted-foreground), Toast.stories.tsx (replaced btn-primary with Button component). Other stories (StatusBadge, AgentRunTable, Skeleton, AgentRunDetailPanel, EventsView, ConfirmDialog, Sidebar, AgentRunForm) only pass props to the migrated components and needed no changes.
- [x] 12.3 Verify all stories render correctly in Storybook with MU-TH-UR styling by running `npm run storybook`

## 13. Global Search-and-Replace Cleanup

- [x] 13.1 Search all .tsx files for remaining `bg-surface-` class references and replace with MU-TH-UR equivalents (bg-background, bg-card, bg-muted)
- [x] 13.2 Search all .tsx files for remaining `text-txt-` class references and replace with MU-TH-UR equivalents (text-foreground, text-muted-foreground)
- [x] 13.3 Search all .tsx files for remaining `border-edge` class references and replace with border-border or border-primary
- [x] 13.4 Search all .tsx files for remaining `text-danger`, `text-warning`, `text-success`, `text-info` class references and replace with text-destructive, text-primary, text-secondary respectively
- [x] 13.5 Search all .tsx files for remaining `rounded` class references and remove them (border-radius is globally 0px)
- [x] 13.6 Search all .tsx files for remaining `.btn` and `.input-field` class references and replace with component usage
- [x] 13.7 Search all .tsx files for remaining `font-sans` references and verify they resolve to IoskeleyMono

## 14. Testing and Verification

- [x] 14.1 Run `npm run build` in web/ and verify zero build errors
- [x] 14.2 Run `tsc --noEmit` in web/ and verify zero TypeScript errors
- [x] 14.3 Run `npm run dev` and visually verify the app renders with MU-TH-UR aesthetic: pure black background, amber text, green secondary, no rounded corners, IoskeleyMono font, uppercase body text
- [x] 14.4 Verify fx-scanlines effect is visible on Card/panel components
- [x] 14.5 Verify fx-glow effect is visible on headings and active nav items
- [x] 14.6 Verify fx-flicker animation runs on the app wrapper
- [x] 14.7 Verify all form inputs render with MU-TH-UR Input component styling (black bg, amber border on focus)
- [x] 14.8 Verify all modals/dialogs render with AlertDialog/Dialog components (black bg, amber borders)
- [x] 14.9 Verify the sidebar renders with MU-TH-UR Sidebar component
- [x] 14.10 Verify table views render with MU-TH-UR Table component styling
- [x] 14.11 Verify toast notifications render with MU-TH-UR styling
- [x] 14.12 Verify scrollbars use MU-TH-UR custom scrollbar styling (6px, black track, muted thumb)
- [x] 14.13 Verify that `prefers-reduced-motion: reduce` disables all CRT animations (flicker, glitch, pulse)
- [x] 14.14 Update E2E test selectors in web/e2e/*.spec.ts if component structure changes break existing CSS selectors or data attributes
- [x] 14.15 Run E2E tests (`npx playwright test`) and verify all pass or document failures with fix plan
