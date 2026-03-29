# global-nav Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Persistent sidebar navigation
The app SHALL render a persistent left sidebar visible on all routes, containing primary navigation links for Runs, Projects, Chains, and Schedules.

#### Scenario: Sidebar visible on all routes
- **WHEN** a user navigates to any route in the app
- **THEN** the sidebar is visible on the left side of the viewport
- **AND** the current route's nav item is highlighted with an accent background and bold text

#### Scenario: Sidebar shows live counts
- **WHEN** the sidebar is visible
- **THEN** each nav item displays a badge with the count of active items (running runs, total projects, etc.)
- **AND** counts update on the same polling interval as the relevant list view

#### Scenario: Sidebar collapse to icon-only mode
- **WHEN** the user clicks the collapse toggle
- **THEN** the sidebar shrinks to ~50px showing only icons
- **AND** the collapse state persists in localStorage across page reloads

#### Scenario: Sidebar expanded by default on wide viewports
- **WHEN** viewport width is >= 1024px and no stored preference exists
- **THEN** the sidebar renders expanded (200px)
- **AND** on viewports < 1024px, it renders collapsed by default

### Requirement: Breadcrumb location context
The app SHALL display a breadcrumb in the main content area header for all detail views.

#### Scenario: Breadcrumb on run detail
- **WHEN** the user is on /run/:id
- **THEN** a breadcrumb shows "Runs / [run name]" with "Runs" as a clickable link to /

#### Scenario: Breadcrumb on project detail
- **WHEN** the user is on /projects/:name
- **THEN** a breadcrumb shows "Projects / [project name]" with "Projects" clickable

### Requirement: Remove scattered nav buttons from individual views
Each view SHALL NOT contain navigation buttons for jumping to other top-level sections.

#### Scenario: RunListView no longer has Projects/Chains/Schedules buttons
- **WHEN** the user views the run list
- **THEN** the header contains only "Runs" title, count, filters, and "+ New Run"
- **AND** navigation to Projects/Chains/Schedules is via the sidebar only

