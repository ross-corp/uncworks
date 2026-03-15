## ADDED Requirements

### Requirement: Radix UI primitives replace all custom component markup
The system SHALL use Radix UI primitives as the foundation for all interactive components. All hand-rolled HTML/CSS component patterns (`.btn`, `.btn-primary`, `.btn-ghost`, `.btn-danger`, `.input-field`) SHALL be removed and replaced with Radix-based components from the copied homelab ui-components library.

#### Scenario: No legacy CSS component classes remain
- **WHEN** the index.css @layer components block is inspected
- **THEN** the `.btn`, `.btn-primary`, `.btn-ghost`, `.btn-danger`, and `.input-field` class definitions SHALL NOT exist

#### Scenario: Button elements use Radix Slot pattern
- **WHEN** a button is rendered in the UI
- **THEN** it SHALL use the `<Button>` component backed by `@radix-ui/react-slot` with CVA variants

### Requirement: CVA variant system for all component styling
The system SHALL use class-variance-authority (CVA) for all component variant definitions. Components SHALL define their visual variants (size, color, state) through CVA's `cva()` function rather than ad-hoc className string concatenation or CSS class stacking.

#### Scenario: Button component uses CVA variants
- **WHEN** the Button component source is inspected
- **THEN** it SHALL define variants using `cva()` with at minimum: variant (default, destructive, outline, secondary, ghost, link, terminal) and size (default, sm, lg, icon)

#### Scenario: Badge component uses CVA variants
- **WHEN** the Badge component source is inspected
- **THEN** it SHALL define variants using `cva()` for visual states (default, secondary, destructive, outline)

### Requirement: cn() utility for class merging
The system SHALL provide a `cn()` utility function that combines `clsx` and `tailwind-merge` for safe Tailwind class composition. All components SHALL use `cn()` for className merging instead of raw template literals or string concatenation.

#### Scenario: cn utility exists and is importable
- **WHEN** a component needs to merge classNames
- **THEN** it SHALL import `cn` from a shared lib/utils module

#### Scenario: cn correctly resolves Tailwind conflicts
- **WHEN** `cn("px-4", "px-6")` is called
- **THEN** the result SHALL be `"px-6"` (tailwind-merge deduplication)

### Requirement: StatusBadge migrated to Badge component
The system SHALL replace the custom `StatusBadge.tsx` with the MU-TH-UR `<Badge>` component. Status states (running, completed, failed, pending, cancelled) SHALL map to Badge variants with appropriate MU-TH-UR colors: running -> secondary (green), completed -> default (amber), failed -> destructive, pending -> outline, cancelled -> secondary with muted opacity.

#### Scenario: Running status displays green badge
- **WHEN** an agent run with status "running" is displayed
- **THEN** the badge SHALL use the secondary variant (Nostromo green)

#### Scenario: Failed status displays destructive badge
- **WHEN** an agent run with status "failed" is displayed
- **THEN** the badge SHALL use the destructive variant

### Requirement: AgentRunTable migrated to Table components
The system SHALL replace the custom `AgentRunTable.tsx` with MU-TH-UR `<Table>`, `<TableHeader>`, `<TableBody>`, `<TableRow>`, `<TableHead>`, and `<TableCell>` components. The table SHALL have no border-radius, use border-border for cell dividers, and display text in the foreground (amber) color.

#### Scenario: Table renders with MU-TH-UR styling
- **WHEN** the agent run table is rendered
- **THEN** it SHALL use the Table component hierarchy (Table > TableHeader > TableRow > TableHead, Table > TableBody > TableRow > TableCell)

#### Scenario: Table has no rounded corners
- **WHEN** the table is rendered
- **THEN** border-radius SHALL be 0px on all table elements

### Requirement: ConfirmDialog migrated to AlertDialog
The system SHALL replace the custom `ConfirmDialog.tsx` with the MU-TH-UR `<AlertDialog>` component. The alert dialog SHALL use AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogAction, and AlertDialogCancel sub-components.

#### Scenario: Destructive confirmation uses AlertDialog
- **WHEN** a destructive action requires user confirmation
- **THEN** the system SHALL render an AlertDialog with amber-bordered content on a pure black backdrop

### Requirement: Toast notifications migrated to Sonner-based Toaster
The system SHALL replace the custom `Toast.tsx` with the MU-TH-UR Sonner-based `<Toaster>` component and `useToast` hook. Toast notifications SHALL appear with MU-TH-UR styling (amber borders, black background, no border-radius).

#### Scenario: Toast renders with MU-TH-UR styling
- **WHEN** a toast notification is triggered
- **THEN** it SHALL appear with black background, amber text, no border-radius, and an amber border

### Requirement: Sidebar migrated to MU-TH-UR Sidebar component
The system SHALL replace the custom `Sidebar.tsx` with the MU-TH-UR `<Sidebar>` component. The sidebar SHALL support collapsible behavior, navigation items with active states using fx-glow, and the MU-TH-UR industrial styling.

#### Scenario: Sidebar renders with terminal aesthetic
- **WHEN** the application layout sidebar is rendered
- **THEN** it SHALL use black background, amber text, no border-radius, and border-border right edge

#### Scenario: Active sidebar item has glow effect
- **WHEN** a sidebar navigation item is active/selected
- **THEN** it SHALL display with fx-glow text effect

### Requirement: GitHubModal migrated to Dialog
The system SHALL replace the custom `GitHubModal.tsx` with the MU-TH-UR `<Dialog>` component, using DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, and DialogFooter sub-components.

#### Scenario: GitHub modal uses Dialog component
- **WHEN** the GitHub configuration modal is opened
- **THEN** it SHALL render as a Dialog with MU-TH-UR styling (black background, amber borders, no radius)

### Requirement: Form inputs use MU-TH-UR Input and Select components
The system SHALL replace all `<input className="input-field">` patterns with the MU-TH-UR `<Input>` component and all custom select elements with the MU-TH-UR `<Select>` component. Form layouts in AgentRunForm.tsx and WorkspaceEditor.tsx SHALL use `<Label>`, `<Input>`, `<Select>`, and `<Button>` components.

#### Scenario: Text inputs render with MU-TH-UR styling
- **WHEN** a text input field is rendered
- **THEN** it SHALL use the Input component with black background, amber border on focus, no border-radius

#### Scenario: Select dropdowns render with MU-TH-UR styling
- **WHEN** a select dropdown is rendered
- **THEN** it SHALL use the Radix Select component with MU-TH-UR popover styling

### Requirement: Layout restructured with MU-TH-UR primitives
The system SHALL restructure `Layout.tsx` to use MU-TH-UR `<Sidebar>`, `<Card>`, and panel components. The layout SHALL use the SidebarProvider/SidebarInset pattern from the MU-TH-UR component library.

#### Scenario: Layout uses SidebarProvider pattern
- **WHEN** the main application layout renders
- **THEN** it SHALL wrap content in SidebarProvider with Sidebar and SidebarInset for the main content area

### Requirement: Scroll areas use MU-TH-UR ScrollArea
The system SHALL replace custom scrollbar-hidden CSS and overflow containers with the MU-TH-UR `<ScrollArea>` component backed by `@radix-ui/react-scroll-area`. The ScrollArea SHALL use the custom MU-TH-UR scrollbar styling (6px width, black track, muted thumb, amber hover).

#### Scenario: Log viewer uses ScrollArea
- **WHEN** the log viewer component renders scrollable content
- **THEN** it SHALL use the ScrollArea component with MU-TH-UR scrollbar styling

### Requirement: Skeleton loading states use MU-TH-UR Skeleton
The system SHALL replace the custom `Skeleton.tsx` with the MU-TH-UR `<Skeleton>` component. Loading skeleton elements SHALL use the muted background color with no border-radius.

#### Scenario: Skeleton has MU-TH-UR styling
- **WHEN** a skeleton loading placeholder is rendered
- **THEN** it SHALL have muted background color, no border-radius, and animate with a pulse effect

### Requirement: All required Radix UI packages are installed
The system SHALL include all Radix UI packages required by the copied components in `web/package.json` dependencies. At minimum: @radix-ui/react-slot, @radix-ui/react-dialog, @radix-ui/react-alert-dialog, @radix-ui/react-select, @radix-ui/react-scroll-area, @radix-ui/react-tabs, @radix-ui/react-separator, @radix-ui/react-label, @radix-ui/react-popover, @radix-ui/react-dropdown-menu, @radix-ui/react-collapsible, @radix-ui/react-accordion, @radix-ui/react-progress, @radix-ui/react-checkbox, @radix-ui/react-tooltip.

#### Scenario: Package.json includes all Radix dependencies
- **WHEN** `web/package.json` dependencies are inspected
- **THEN** all required @radix-ui/* packages SHALL be present

#### Scenario: Dependencies install without errors
- **WHEN** `npm install` is run in the web directory
- **THEN** all Radix UI packages SHALL resolve and install without peer dependency conflicts
