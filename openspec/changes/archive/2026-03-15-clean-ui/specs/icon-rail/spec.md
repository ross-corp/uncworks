## ADDED Requirements

### Requirement: 48px fixed-width left rail
The icon rail SHALL be a fixed 48px-wide vertical strip on the left edge of the viewport. It SHALL span the full viewport height. Icons SHALL be stacked vertically at the top with consistent spacing.

#### Scenario: Rail renders at correct width
- **WHEN** the application loads
- **THEN** a 48px-wide rail is visible on the left
- **AND** the main content area fills the remaining width

### Requirement: Filter icon opens status popover
The filter icon (funnel) SHALL open a small popover when clicked. The popover SHALL contain radio buttons for status filtering: All, Active, Done, Failed. Selecting a filter SHALL update the run list immediately and close the popover.

#### Scenario: Open filter popover
- **WHEN** user clicks the filter funnel icon
- **THEN** a popover appears adjacent to the icon with status radio options

#### Scenario: Apply filter
- **WHEN** user selects "Active" in the filter popover
- **THEN** the run list filters to show only active runs
- **AND** the popover closes
- **AND** the filter icon shows a visual indicator that a filter is active

#### Scenario: Clear filter
- **WHEN** user selects "All" in the filter popover
- **THEN** all runs are shown
- **AND** the filter active indicator is removed

### Requirement: New run icon opens create form
The plus icon SHALL open the create run dialog when clicked. This is the same dialog triggered by the "New Run" command in the command palette.

#### Scenario: Open create run form
- **WHEN** user clicks the plus icon
- **THEN** the create run dialog opens

### Requirement: Theme icon toggles dark/light
The theme icon SHALL display a sun icon in dark mode and a moon icon in light mode. Clicking SHALL toggle between themes immediately.

#### Scenario: Toggle to dark mode
- **WHEN** the current theme is light
- **AND** user clicks the theme icon (moon)
- **THEN** the theme switches to dark mode
- **AND** the icon changes to sun

### Requirement: Tooltips on hover
Each icon SHALL display a tooltip on hover showing its label: "Filter", "New Run", "Theme". Tooltips SHALL appear after a short delay (200ms) and position to the right of the rail.

#### Scenario: Tooltip appears
- **WHEN** user hovers over the filter icon for 200ms
- **THEN** a tooltip reading "Filter" appears to the right of the icon
