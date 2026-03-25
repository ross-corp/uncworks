## ADDED Requirements

### Requirement: Terminal theme follows site theme
The xterm.js terminal SHALL use a color scheme that matches the site's current dark/light mode.

#### Scenario: Terminal in dark mode
- **WHEN** the site is in dark mode
- **THEN** the terminal SHALL use a dark background with light text matching the site's color palette

#### Scenario: Terminal in light mode
- **WHEN** the site is in light mode
- **THEN** the terminal SHALL use a light background with dark text

### Requirement: Theme picker is accessible
The system SHALL provide a visible theme toggle or picker that allows switching between light, dark, and system themes.

#### Scenario: Theme toggle in layout
- **WHEN** a user views any page
- **THEN** a theme toggle SHALL be visible (in the header or a consistent location)

#### Scenario: Theme persists across sessions
- **WHEN** a user selects a theme
- **THEN** the selection SHALL persist in localStorage and be applied on next visit

### Requirement: shadcn components replace raw HTML
Form elements throughout the UI SHALL use shadcn components for visual consistency.

#### Scenario: Select dropdowns use shadcn Select
- **WHEN** the New Run view renders model and orchestration selectors
- **THEN** they SHALL use shadcn Select or DropdownMenu components instead of raw HTML select elements

#### Scenario: Run detail tabs use shadcn Tabs
- **WHEN** the Run Detail view renders tab navigation
- **THEN** it SHALL use the shadcn Tabs component
