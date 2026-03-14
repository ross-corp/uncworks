## ADDED Requirements

### Requirement: Four-zone CSS Grid command center layout
The web UI SHALL use a CSS Grid layout with four named zones: navigator (left), workspace (center), detail (right), and log stream (bottom). The grid SHALL be defined with `grid-template-areas` and support resizable zone boundaries via drag handles. The navigator SHALL default to 280px wide, the detail panel to 320px wide, and the log stream to 200px tall. The workspace SHALL fill remaining space.

#### Scenario: Initial layout renders all four zones
- **WHEN** the application loads
- **THEN** the navigator, workspace, detail panel, and log stream are all visible
- **AND** each zone occupies its configured default size

#### Scenario: Resize navigator panel
- **WHEN** user drags the navigator's right edge
- **THEN** the navigator width changes and the workspace adjusts to fill remaining space
- **AND** minimum width of 200px is enforced

#### Scenario: Resize log stream panel
- **WHEN** user drags the log stream's top edge
- **THEN** the log stream height changes and the workspace adjusts
- **AND** minimum height of 32px (title bar only) is enforced

#### Scenario: Minimize log stream
- **WHEN** user clicks the minimize button on the log stream title bar (or presses Ctrl+L)
- **THEN** the log stream collapses to 32px showing only the title bar with run ID and status
- **AND** pressing Ctrl+L again restores the previous height

### Requirement: MU-TH-UR theme applied globally
The root container SHALL apply MU-TH-UR CSS custom properties (background, text colors, glow effects, scanline overlay, monospace font). All child components SHALL inherit these properties. The scanline effect SHALL be rendered via a `::after` pseudo-element on the root container with `pointer-events: none`.

#### Scenario: Theme variables are available to all components
- **WHEN** any component renders
- **THEN** CSS custom properties `--mu-bg-primary`, `--mu-text-primary`, `--mu-glow`, `--mu-scanline`, and `--mu-font-mono` are defined and usable

#### Scenario: Scanline overlay is visible but non-interactive
- **WHEN** the application renders
- **THEN** a scanline overlay is visible across the entire viewport
- **AND** mouse clicks pass through the overlay to underlying components

### Requirement: Workspace view tab bar
The workspace zone SHALL display a tab bar at its top with tabs for Graph, Timeline, Diff, Files, and Shell. Clicking a tab SHALL switch the active view. The active tab SHALL be visually indicated with the MU-TH-UR accent color. Views SHALL remain mounted when inactive (hidden via `display: none`) to preserve internal state.

#### Scenario: Switch between workspace views
- **WHEN** user clicks the Files tab while Graph view is active
- **THEN** the Graph view is hidden (not unmounted) and the Files view is shown
- **AND** returning to Graph view restores its previous scroll position and selected node

#### Scenario: Tab bar reflects active view
- **WHEN** a workspace view is active
- **THEN** its tab is highlighted with the accent color and a bottom border glow
