## ADDED Requirements

### Requirement: shadcn component adoption
The system SHALL use shadcn/ui components for standard UI patterns instead of custom implementations.

#### Scenario: tabs component
- **WHEN** the run detail view renders tab navigation
- **THEN** it SHALL use shadcn `Tabs` component instead of custom button bar

#### Scenario: info panel
- **WHEN** the user presses "i" to view run metadata
- **THEN** it SHALL display in a shadcn `Sheet` (slide-over panel) instead of inline div

#### Scenario: stage progress
- **WHEN** viewing a spec-driven run's stage progress bar
- **THEN** it SHALL use shadcn `Progress` with `Badge` components

### Requirement: consistent agent labels
The system SHALL use "manage" and "impl" labels consistently across all UI surfaces.

#### Scenario: activity feed labels
- **WHEN** viewing activity entries
- **THEN** agent-manage entries SHALL show "manage" in blue and agent-implement entries SHALL show "impl" in green

#### Scenario: thinking indicator
- **WHEN** an agent is actively thinking
- **THEN** the thinking indicator SHALL show the agent's role label (manage or impl)

### Requirement: professional layout
The system SHALL use consistent spacing, typography, and color tokens from the active shadcn theme.

#### Scenario: theme switching
- **WHEN** the user selects a theme from the command palette or theme toggle
- **THEN** all components SHALL reflect the theme change immediately using CSS custom properties
