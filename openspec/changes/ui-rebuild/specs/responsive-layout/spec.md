## ADDED Requirements

### Requirement: Adaptive panel behavior for smaller viewports
The layout SHALL adapt at two breakpoints. At 1024-1439px width: the navigator SHALL collapse to an icon rail (48px wide) showing only spec status icons, expandable to full width on hover or click; the detail panel SHALL become a slide-over overlay instead of a fixed column. At below 1024px width: the application SHALL display a "Use a wider screen for the MU-TH-UR 6000 command center" message and not attempt to render the full UI.

#### Scenario: Medium viewport collapses navigator
- **WHEN** viewport width is between 1024px and 1439px
- **THEN** the navigator collapses to a 48px icon rail showing spec status icons
- **AND** hovering over or clicking the rail expands the full navigator as an overlay

#### Scenario: Medium viewport converts detail panel to overlay
- **WHEN** viewport width is between 1024px and 1439px
- **THEN** the detail panel is hidden by default
- **AND** selecting a run causes the detail panel to slide in as an overlay from the right
- **AND** clicking outside the overlay or pressing Escape closes it

#### Scenario: Small viewport shows unsupported message
- **WHEN** viewport width is below 1024px
- **THEN** the full UI is not rendered
- **AND** a centered message reads "Use a wider screen for the MU-TH-UR 6000 command center"

#### Scenario: Log stream remains visible at all supported widths
- **WHEN** viewport width is 1024px or wider
- **THEN** the log stream remains visible at the bottom of the layout
- **AND** it can still be minimized and restored

### Requirement: Panel toggle buttons
Each collapsible panel (navigator, detail, log stream) SHALL have a toggle button visible at all viewport sizes. The toggle button SHALL use an icon indicating the panel's expand/collapse state. On medium viewports, the toggle buttons SHALL be the primary way to show collapsed panels.

#### Scenario: Toggle navigator on medium viewport
- **WHEN** user clicks the navigator toggle button on a medium viewport
- **THEN** the navigator expands as an overlay with the full spec tree
- **AND** clicking the toggle again collapses it back to the icon rail

#### Scenario: Toggle detail panel
- **WHEN** user clicks the detail panel toggle button
- **THEN** the detail panel slides in from the right as an overlay
- **AND** it shows the currently selected entity's metadata
