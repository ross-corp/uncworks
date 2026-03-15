## ADDED Requirements

### Requirement: RunList renders as an HTML table with fixed column layout
The RunList component SHALL render runs as rows in an `<table>` element with `table-layout: fixed`. Each row SHALL display: status dot (20px column), run ID in monospace (100px), prompt text truncated with ellipsis (flex), repository name (120px), phase text (80px), and relative time (60px). Rows SHALL be 32px tall.

#### Scenario: Table renders with correct columns
- **WHEN** the RunList component mounts with run data
- **THEN** each run renders as a `<tr>` with 6 `<td>` cells
- **AND** the table uses `table-layout: fixed` CSS

#### Scenario: Long prompts are truncated
- **WHEN** a run's prompt text exceeds the available column width
- **THEN** the prompt cell shows truncated text with CSS `text-overflow: ellipsis`
- **AND** the full prompt is available via the title attribute

### Requirement: 15-20 runs visible without scrolling on 1080p
The table SHALL display 15-20 rows within a 1080p viewport (1920x1080) without requiring vertical scrolling. Row height (32px) and header height SHALL be calibrated to achieve this density.

#### Scenario: Dense display on standard monitor
- **GIVEN** a 1080p viewport with browser chrome
- **WHEN** 20 runs exist
- **THEN** all 20 rows are visible without scrolling the table body

### Requirement: Status dot colors match semantic tokens
Each run's status dot SHALL use the semantic color token for its phase. Green (`--color-success`) for succeeded, blue pulsing (`--color-active`) for running, amber (`--color-warning`) for pending, red (`--color-error`) for failed, gray (`--color-neutral`) for cancelled.

#### Scenario: Running run shows pulsing blue dot
- **WHEN** a run has phase "Running"
- **THEN** its status dot uses `--color-active` with a CSS pulse animation

#### Scenario: Failed run shows red dot
- **WHEN** a run has phase "Failed"
- **THEN** its status dot uses `--color-error` with no animation

### Requirement: Selected row is visually distinct
The selected row SHALL have a `bg-accent/10` background and a 2px left border using `--color-accent`. Only one row can be selected at a time.

#### Scenario: Clicking a row selects it
- **WHEN** user clicks a row
- **THEN** that row receives the selected style
- **AND** any previously selected row loses the selected style

#### Scenario: Double-clicking a row opens detail pane
- **WHEN** user double-clicks a row
- **THEN** that row is selected
- **AND** the split-pane detail opens for that run

### Requirement: Alternating row backgrounds for scannability
Table rows SHALL alternate between transparent background and `bg-muted/5` background to aid visual scanning across columns.

#### Scenario: Row striping
- **WHEN** the table renders multiple rows
- **THEN** odd rows have transparent background
- **AND** even rows have `--color-muted` at 5% opacity
