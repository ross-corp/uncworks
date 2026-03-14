## 1. Foundation — Dependencies and Build Setup

- [ ] 1.1 Install Radix UI dependencies: @radix-ui/react-slot, @radix-ui/react-dialog, @radix-ui/react-alert-dialog, @radix-ui/react-select, @radix-ui/react-scroll-area, @radix-ui/react-tabs, @radix-ui/react-separator, @radix-ui/react-label, @radix-ui/react-popover, @radix-ui/react-dropdown-menu, @radix-ui/react-collapsible, @radix-ui/react-accordion, @radix-ui/react-progress, @radix-ui/react-checkbox, @radix-ui/react-tooltip, @radix-ui/react-switch, @radix-ui/react-toggle, @radix-ui/react-toggle-group, @radix-ui/react-avatar, @radix-ui/react-hover-card, @radix-ui/react-menubar, @radix-ui/react-navigation-menu, @radix-ui/react-radio-group, @radix-ui/react-slider, @radix-ui/react-context-menu, @radix-ui/react-aspect-ratio
- [ ] 1.2 Install styling dependencies: class-variance-authority, clsx, tailwind-merge, tw-animate-css
- [ ] 1.3 Upgrade Tailwind CSS from v3 to v4: update tailwindcss package, update postcss config if needed, update vite config for Tailwind v4 plugin
- [ ] 1.4 Install sonner (toast library used by MU-TH-UR toaster component)
- [ ] 1.5 Verify all dependencies install without peer dependency conflicts by running `npm install`

## 2. Font Migration

- [ ] 2.1 Obtain IoskeleyMono font files (.woff2, .woff) from the homelab project or font source
- [ ] 2.2 Create `web/public/fonts/` directory and copy IoskeleyMono font files into it
- [ ] 2.3 Add @font-face declarations for IoskeleyMono in the CSS (weight 400, 500, 600, 700 if available)
- [ ] 2.4 Remove the Google Fonts @import for Inter and JetBrains Mono from web/src/index.css
- [ ] 2.5 Verify IoskeleyMono renders correctly in the browser dev tools

## 3. Design Tokens Migration

- [ ] 3.1 Replace web/src/index.css entirely: remove all :root custom properties (--surface-*, --text-*, --border*, --accent*, --danger, --warning, --success, --info) and replace with MU-TH-UR :root block containing --background, --foreground, --card, --card-foreground, --popover, --popover-foreground, --primary, --primary-foreground, --secondary, --secondary-foreground, --muted, --muted-foreground, --accent, --accent-foreground, --destructive, --destructive-foreground, --border, --input, --ring, --radius, --glow-primary, --glow-secondary using oklch values
- [ ] 3.2 Add @theme inline block to CSS with Tailwind v4 token mappings: --font-sans, --font-mono, --color-background, --color-foreground, --color-primary, --color-primary-foreground, --color-secondary, --color-secondary-foreground, --color-muted, --color-muted-foreground, --color-accent, --color-accent-foreground, --color-destructive, --color-destructive-foreground, --color-border, --color-input, --color-ring, --radius-sm/md/lg/xl all set to 0px
- [ ] 3.3 Add @layer base block: set * to border-border and rounded-none, set body to bg-background text-foreground uppercase tracking-widest font-mono
- [ ] 3.4 Remove the @layer components block containing .input-field, .btn, .btn-primary, .btn-ghost, .btn-danger, .scrollbar-hidden
- [ ] 3.5 Add CRT effects layer (@layer utilities): fx-scanlines, fx-noise, fx-glow, fx-glow-amber, fx-glow-danger, fx-box-glow, fx-box-glow-warning, fx-box-glow-danger, fx-panel
- [ ] 3.6 Add CRT animation keyframes: fx-flicker (6s), fx-glitch (4s), fx-pulse-glow (3s), fx-border-flash (1s), fx-text-flicker (0.1s) — all wrapped in @media (prefers-reduced-motion: no-preference)
- [ ] 3.7 Add MU-TH-UR scrollbar styles: ::-webkit-scrollbar (6px), ::-webkit-scrollbar-track (background + border), ::-webkit-scrollbar-thumb (muted, 0px radius, amber on hover)
- [ ] 3.8 Replace or rewrite web/tailwind.config.ts: remove all theme.extend.colors (surface, edge, txt, accent, danger, warning, success, info) and theme.extend.fontFamily. For Tailwind v4, this file may become minimal or replaced by the @theme inline block in CSS

## 4. Component Library Setup

- [ ] 4.1 Create `web/src/lib/utils.ts` with the cn() utility function: import clsx and twMerge, export cn(...inputs) that calls twMerge(clsx(inputs))
- [ ] 4.2 Create `web/src/components/ui/` directory for MU-TH-UR components
- [ ] 4.3 Copy button.tsx from homelab/apps/lib/ui-components/src/components/ui/button.tsx to web/src/components/ui/button.tsx, update import path for cn() to ../../lib/utils
- [ ] 4.4 Copy card.tsx from homelab ui-components to web/src/components/ui/card.tsx, update cn() import
- [ ] 4.5 Copy badge.tsx from homelab ui-components to web/src/components/ui/badge.tsx, update cn() import
- [ ] 4.6 Copy table.tsx from homelab ui-components to web/src/components/ui/table.tsx, update cn() import
- [ ] 4.7 Copy dialog.tsx from homelab ui-components to web/src/components/ui/dialog.tsx, update cn() import
- [ ] 4.8 Copy alert-dialog.tsx from homelab ui-components to web/src/components/ui/alert-dialog.tsx, update cn() import
- [ ] 4.9 Copy sheet.tsx from homelab ui-components to web/src/components/ui/sheet.tsx, update cn() import
- [ ] 4.10 Copy sidebar.tsx from homelab ui-components to web/src/components/ui/sidebar.tsx, update cn() import
- [ ] 4.11 Copy input.tsx from homelab ui-components to web/src/components/ui/input.tsx, update cn() import
- [ ] 4.12 Copy select.tsx from homelab ui-components to web/src/components/ui/select.tsx, update cn() import
- [ ] 4.13 Copy label.tsx from homelab ui-components to web/src/components/ui/label.tsx, update cn() import
- [ ] 4.14 Copy tabs.tsx from homelab ui-components to web/src/components/ui/tabs.tsx, update cn() import
- [ ] 4.15 Copy scroll-area.tsx from homelab ui-components to web/src/components/ui/scroll-area.tsx, update cn() import
- [ ] 4.16 Copy skeleton.tsx from homelab ui-components to web/src/components/ui/skeleton.tsx, update cn() import
- [ ] 4.17 Copy toast.tsx, toaster.tsx, and use-toast.ts from homelab ui-components to web/src/components/ui/, update cn() imports
- [ ] 4.18 Copy sonner.tsx from homelab ui-components to web/src/components/ui/sonner.tsx, update cn() import
- [ ] 4.19 Copy separator.tsx from homelab ui-components to web/src/components/ui/separator.tsx, update cn() import
- [ ] 4.20 Copy tooltip.tsx from homelab ui-components to web/src/components/ui/tooltip.tsx, update cn() import
- [ ] 4.21 Copy collapsible.tsx from homelab ui-components to web/src/components/ui/collapsible.tsx, update cn() import
- [ ] 4.22 Copy accordion.tsx from homelab ui-components to web/src/components/ui/accordion.tsx, update cn() import
- [ ] 4.23 Copy progress.tsx from homelab ui-components to web/src/components/ui/progress.tsx, update cn() import
- [ ] 4.24 Copy dropdown-menu.tsx from homelab ui-components to web/src/components/ui/dropdown-menu.tsx, update cn() import
- [ ] 4.25 Copy popover.tsx from homelab ui-components to web/src/components/ui/popover.tsx, update cn() import
- [ ] 4.26 Copy checkbox.tsx from homelab ui-components to web/src/components/ui/checkbox.tsx, update cn() import
- [ ] 4.27 Copy alert.tsx from homelab ui-components to web/src/components/ui/alert.tsx, update cn() import
- [ ] 4.28 Copy spinner.tsx from homelab ui-components to web/src/components/ui/spinner.tsx, update cn() import
- [ ] 4.29 Copy empty.tsx from homelab ui-components to web/src/components/ui/empty.tsx, update cn() import
- [ ] 4.30 Verify all copied components compile without TypeScript errors by running `tsc --noEmit`

## 5. Component Migration — Structural Components

- [ ] 5.1 Rewrite Layout.tsx: replace custom layout markup with SidebarProvider + Sidebar + SidebarInset pattern from MU-TH-UR. Apply fx-flicker to the root app wrapper.
- [ ] 5.2 Rewrite Sidebar.tsx: replace custom sidebar with MU-TH-UR Sidebar component (SidebarHeader, SidebarContent, SidebarMenu, SidebarMenuItem, SidebarMenuButton). Apply fx-glow to the active nav item.
- [ ] 5.3 Rewrite App.tsx: wrap the app in the Toaster provider for sonner-based toast notifications. Add fx-flicker class to the outermost div.

## 6. Component Migration — Data Display

- [ ] 6.1 Rewrite AgentRunTable.tsx: replace custom table markup with MU-TH-UR Table, TableHeader, TableBody, TableRow, TableHead, TableCell components. Remove all bg-surface-*, text-txt-*, border-edge class references.
- [ ] 6.2 Rewrite StatusBadge.tsx: replace custom badge with MU-TH-UR Badge component. Map statuses: running -> variant="secondary" + fx-pulse, completed -> variant="default", failed -> variant="destructive" + fx-glitch, pending -> variant="outline", cancelled -> variant="secondary" with opacity-50.
- [ ] 6.3 Rewrite EventsView.tsx: replace custom event list with Table + Badge components. Use fx-scanlines on the events container Card.
- [ ] 6.4 Rewrite TraceTimeline.tsx: wrap in MU-TH-UR Card component. Replace surface/txt color classes with MU-TH-UR tokens. Apply fx-scanlines to the timeline container.
- [ ] 6.5 Rewrite ReposView.tsx: replace custom repo list with Table + Badge components. Use MU-TH-UR color tokens for all text and backgrounds.

## 7. Component Migration — Detail Panels and Editors

- [ ] 7.1 Rewrite AgentRunDetailPanel.tsx: replace custom panel with MU-TH-UR Sheet or Card component. Replace all surface/txt/edge class references. Apply fx-scanlines to the content area.
- [ ] 7.2 Rewrite SpecEditor.tsx: wrap Monaco editor in MU-TH-UR Card + Tabs components. Replace surface/txt classes. Create a MU-TH-UR Monaco editor theme with amber (#FFB000) and green (#4AF626) syntax highlighting on pure black background.
- [ ] 7.3 Rewrite WorkspaceEditor.tsx: replace form markup with Card + Input + Select + Label + Button components. Remove all .input-field and .btn class references.
- [ ] 7.4 Rewrite DiffViewer.tsx: wrap in MU-TH-UR Card + ScrollArea. Replace color classes with MU-TH-UR tokens.

## 8. Component Migration — Log and Terminal Views

- [ ] 8.1 Rewrite LogViewer.tsx: wrap in MU-TH-UR Card + ScrollArea. Apply fx-scanlines and fx-noise to the log container. Replace all surface/txt color classes.
- [ ] 8.2 Rewrite LogViewerInner.tsx: update all className references from surface/txt/edge tokens to MU-TH-UR equivalents (bg-background, text-foreground, border-border).
- [ ] 8.3 Rewrite ShellTerminal.tsx: wrap xterm.js container in MU-TH-UR Card. Apply fx-scanlines and fx-flicker to the terminal wrapper div (NOT to the xterm canvas element). Replace surface/txt classes.
- [ ] 8.4 Rewrite ShellTerminalInner.tsx: update color class references to MU-TH-UR tokens.

## 9. Component Migration — File Browser

- [ ] 9.1 Rewrite FileExplorer.tsx: replace custom file explorer with MU-TH-UR Card + Collapsible/Accordion for tree structure. Replace color classes.
- [ ] 9.2 Rewrite FileTree.tsx: use MU-TH-UR Collapsible for expand/collapse behavior. Apply fx-glow to selected file item.
- [ ] 9.3 Rewrite FilePreview.tsx: wrap in MU-TH-UR Card + ScrollArea. Replace color classes.

## 10. Component Migration — Forms and Dialogs

- [ ] 10.1 Rewrite AgentRunForm.tsx: replace form with Card + Label + Input + Select + Button components. Remove all .input-field and .btn-primary class references. Use MU-TH-UR Button variant="terminal" for the submit action.
- [ ] 10.2 Rewrite ConfirmDialog.tsx: replace custom modal with MU-TH-UR AlertDialog (AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogAction, AlertDialogCancel). Use Button variant="destructive" for the confirm action.
- [ ] 10.3 Rewrite GitHubModal.tsx: replace custom modal with MU-TH-UR Dialog (DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter). Replace form inputs with MU-TH-UR Input component.
- [ ] 10.4 Rewrite Toast.tsx: replace custom toast implementation with MU-TH-UR Toaster (sonner-based). Update all toast trigger call sites throughout the app to use the new useToast hook or sonner toast() function.

## 11. Component Migration — Utility Components

- [ ] 11.1 Rewrite Skeleton.tsx: replace with MU-TH-UR Skeleton component. Update all import paths in consuming files.
- [ ] 11.2 Rewrite ErrorBoundary.tsx: use MU-TH-UR Alert component with variant="destructive" for the error display. Apply fx-glitch to the error content.

## 12. Storybook Updates

- [ ] 12.1 Update web/.storybook/preview.ts: import the new MU-TH-UR globals.css instead of the old index.css. Set dark background decorator.
- [ ] 12.2 Update all .stories.tsx files to reflect the new component APIs: StatusBadge.stories.tsx, AgentRunTable.stories.tsx, Skeleton.stories.tsx, AgentRunDetailPanel.stories.tsx, ErrorBoundary.stories.tsx, EventsView.stories.tsx, ConfirmDialog.stories.tsx, Toast.stories.tsx, Sidebar.stories.tsx, Layout.stories.tsx, AgentRunForm.stories.tsx
- [ ] 12.3 Verify all stories render correctly in Storybook with MU-TH-UR styling by running `npm run storybook`

## 13. Global Search-and-Replace Cleanup

- [ ] 13.1 Search all .tsx files for remaining `bg-surface-` class references and replace with MU-TH-UR equivalents (bg-background, bg-card, bg-muted)
- [ ] 13.2 Search all .tsx files for remaining `text-txt-` class references and replace with MU-TH-UR equivalents (text-foreground, text-muted-foreground)
- [ ] 13.3 Search all .tsx files for remaining `border-edge` class references and replace with border-border or border-primary
- [ ] 13.4 Search all .tsx files for remaining `text-danger`, `text-warning`, `text-success`, `text-info` class references and replace with text-destructive, text-primary, text-secondary respectively
- [ ] 13.5 Search all .tsx files for remaining `rounded` class references and remove them (border-radius is globally 0px)
- [ ] 13.6 Search all .tsx files for remaining `.btn` and `.input-field` class references and replace with component usage
- [ ] 13.7 Search all .tsx files for remaining `font-sans` references and verify they resolve to IoskeleyMono

## 14. Testing and Verification

- [ ] 14.1 Run `npm run build` in web/ and verify zero build errors
- [ ] 14.2 Run `tsc --noEmit` in web/ and verify zero TypeScript errors
- [ ] 14.3 Run `npm run dev` and visually verify the app renders with MU-TH-UR aesthetic: pure black background, amber text, green secondary, no rounded corners, IoskeleyMono font, uppercase body text
- [ ] 14.4 Verify fx-scanlines effect is visible on Card/panel components
- [ ] 14.5 Verify fx-glow effect is visible on headings and active nav items
- [ ] 14.6 Verify fx-flicker animation runs on the app wrapper
- [ ] 14.7 Verify all form inputs render with MU-TH-UR Input component styling (black bg, amber border on focus)
- [ ] 14.8 Verify all modals/dialogs render with AlertDialog/Dialog components (black bg, amber borders)
- [ ] 14.9 Verify the sidebar renders with MU-TH-UR Sidebar component
- [ ] 14.10 Verify table views render with MU-TH-UR Table component styling
- [ ] 14.11 Verify toast notifications render with MU-TH-UR styling
- [ ] 14.12 Verify scrollbars use MU-TH-UR custom scrollbar styling (6px, black track, muted thumb)
- [ ] 14.13 Verify that `prefers-reduced-motion: reduce` disables all CRT animations (flicker, glitch, pulse)
- [ ] 14.14 Update E2E test selectors in web/e2e/*.spec.ts if component structure changes break existing CSS selectors or data attributes
- [ ] 14.15 Run E2E tests (`npx playwright test`) and verify all pass or document failures with fix plan
