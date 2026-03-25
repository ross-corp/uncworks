## ADDED Requirements

### Requirement: Unified button scale
All interactive buttons throughout the app SHALL use shadcn Button with `size="sm"` (default) or `size="icon"` for icon-only buttons. Raw `<button>` elements SHALL only be used for icon/toggle controls that are not actions (e.g., collapse toggle, theme cycler).

#### Scenario: No custom height classes on Button
- **WHEN** any Button component is rendered
- **THEN** it does NOT have className containing h-6, h-7, h-8, or h-9

#### Scenario: Destructive actions
- **WHEN** a delete/destructive action button is rendered
- **THEN** it uses `variant="destructive"` or `variant="ghost"` with red text

### Requirement: Consistent view header
Every list and detail view SHALL have a header with height h-12 (py-0 flex items-center), border-b, px-4, with breadcrumb on the left and primary action on the right.

#### Scenario: Header structure
- **WHEN** any view renders
- **THEN** the header div has classes `h-12 border-b flex items-center px-4`

#### Scenario: No custom py in headers
- **WHEN** any view header is rendered
- **THEN** it does NOT have py-2 or py-3 on the outer header div

### Requirement: Consistent row padding
All list row items SHALL use `px-4 py-2.5` for padding, `hover:bg-muted/30` for hover, and `border-b border-border/40` for separator.

#### Scenario: Row padding
- **WHEN** a list row renders
- **THEN** it has px-4 py-2.5 (not py-3 or py-1)

### Requirement: Text scale
Content text SHALL use `text-sm`. Metadata/secondary text SHALL use `text-xs text-muted-foreground`. No `text-[10px]` or `text-[11px]` SHALL appear anywhere in the codebase.

#### Scenario: No micro text sizes
- **WHEN** any view renders
- **THEN** no element has className containing text-[10px] or text-[11px]

### Requirement: Single toast system
All toast notifications SHALL use sonner (`import { toast } from "sonner"`). The custom `useToast` hook SHALL NOT be imported in any view or component.

#### Scenario: Error toast
- **WHEN** an API call fails
- **THEN** toast.error(message) is called via sonner

#### Scenario: Success toast
- **WHEN** a create/delete/save succeeds
- **THEN** toast.success(message) is called via sonner
