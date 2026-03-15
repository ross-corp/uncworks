## ADDED Requirements

### Requirement: Detail view replaces feed content when opened
When a run is selected, the detail view SHALL replace the feed area with a full-width detail view. It SHALL NOT render as a narrow side panel alongside the feed. The feed is hidden while the detail view is open.

#### Scenario: Detail replaces feed
- **WHEN** the user clicks a RunCard in the feed
- **THEN** the feed SHALL be hidden
- **AND** the detail view SHALL render in the same content area at full width

#### Scenario: Detail is not a side panel
- **WHEN** the detail view is open
- **THEN** the feed and detail SHALL NOT be visible simultaneously in a split layout

### Requirement: Detail view header displays name, status, and close button
The detail view SHALL have a header containing the run name, the current status (using semantic color), and a close button (X icon) that returns to the feed.

#### Scenario: Header displays run information
- **WHEN** the detail view opens for a run named "fix-auth-bug" with phase "Running"
- **THEN** the header SHALL display "fix-auth-bug"
- **AND** the header SHALL display the status using the `--color-active` semantic token
- **AND** the header SHALL contain a close button (X icon)

#### Scenario: Close button returns to feed
- **WHEN** the user clicks the close button (X) in the detail header
- **THEN** the detail view SHALL close
- **AND** the feed SHALL become visible again

### Requirement: Detail view has a tab bar with Info, Logs, Files, Shell, and Traces tabs
The detail view SHALL contain a horizontal tab bar below the header with tabs: Info, Logs, Files, Shell, Traces. Clicking a tab SHALL display the corresponding content panel below the tab bar.

#### Scenario: Tab bar renders all tabs
- **WHEN** the detail view opens
- **THEN** a tab bar SHALL display tabs labeled "Info", "Logs", "Files", "Shell", "Traces"

#### Scenario: Info tab is active by default
- **WHEN** the detail view first opens
- **THEN** the "Info" tab SHALL be in the active/selected state
- **AND** the Info content panel SHALL be visible

#### Scenario: Clicking a tab switches content
- **WHEN** the user clicks the "Logs" tab
- **THEN** the "Logs" tab SHALL become active
- **AND** the Logs content panel SHALL be displayed
- **AND** the previously active tab/content SHALL be hidden

### Requirement: Back navigation via X button, Escape key, and breadcrumb
The user SHALL be able to return from the detail view to the feed using three methods: the X close button in the header, pressing the Escape key, or clicking a breadcrumb/back link.

#### Scenario: Escape key closes detail
- **WHEN** the detail view is open and no input element is focused
- **AND** the user presses the Escape key
- **THEN** the detail view SHALL close and the feed SHALL be shown

#### Scenario: Breadcrumb navigation
- **WHEN** the detail view is open
- **THEN** a breadcrumb or back link (e.g., "Runs / fix-auth-bug") SHALL be visible
- **AND** clicking the "Runs" portion SHALL close the detail and return to the feed

### Requirement: Progressive disclosure — summary first, drill into depth
The detail view SHALL follow progressive disclosure. The Info tab SHALL show a summary of the run (status, duration, repos, prompt, timestamps). Deeper information (full logs, file trees, terminal, traces) SHALL only be available when the user explicitly navigates to the corresponding tab.

#### Scenario: Info tab shows summary
- **WHEN** the Info tab is active
- **THEN** it SHALL display: run status, duration, repository list, the full prompt, creation timestamp, and completion timestamp (if applicable)

#### Scenario: Detailed content requires tab navigation
- **WHEN** the detail view first opens
- **THEN** the full log output SHALL NOT be loaded or visible until the user clicks the "Logs" tab
- **AND** the file tree SHALL NOT be loaded or visible until the user clicks the "Files" tab
- **AND** the shell terminal SHALL NOT be connected until the user clicks the "Shell" tab
- **AND** traces SHALL NOT be loaded until the user clicks the "Traces" tab
