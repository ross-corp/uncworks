# semantic-colors Specification

## Purpose
TBD - created by archiving change observability-ux-overhaul. Update Purpose after archive.
## Requirements
### Requirement: Unified role color system via CSS custom properties
The system SHALL define semantic role colors as CSS custom properties that adapt to light and dark modes, shared across all UI components.

#### Scenario: Role colors defined in globals.css
- **WHEN** the application loads in dark mode
- **THEN** CSS custom properties `--role-manage`, `--role-implement`, `--role-system`, `--role-user`, `--role-delegate`, `--role-error` SHALL be defined with HSL values
- **AND** the same properties SHALL have different values in the `:root` (light) and `.dark` selectors

#### Scenario: Activity feed uses semantic colors
- **WHEN** a log entry with label "implement" is rendered in the activity feed
- **THEN** its label text color SHALL use `var(--role-implement)` (emerald in dark, darker emerald in light)
- **AND** a log entry with label "manage" SHALL use `var(--role-manage)` (blue)
- **AND** a log entry with label "system" SHALL use `var(--role-system)` (amber)

#### Scenario: Trace waterfall uses same semantic colors
- **WHEN** a span with role "implement" is rendered in the trace waterfall
- **THEN** its left border and text SHALL use the same `var(--role-implement)` color as the activity feed

### Requirement: Label rename from impl to implement
The system SHALL display "implement" (not "impl") as the role label in the activity feed, and "manage" / "implement" in trace span names.

#### Scenario: Activity feed shows full role names
- **WHEN** a tool call entry from the execute stage is displayed
- **THEN** the label SHALL read "implement" (not "impl" or "neph")

#### Scenario: Backend span names use manage/implement
- **WHEN** the sidecar creates a span during the plan stage
- **THEN** the span name SHALL start with "manage." (not "unc.")
- **WHEN** the sidecar creates a span during the execute stage
- **THEN** the span name SHALL start with "implement." (not "neph.")

### Requirement: Dark/light mode toggle in global layout
The system SHALL display a dark/light mode toggle button in the Layout footer, visible on every page.

#### Scenario: Toggle visible on run list
- **WHEN** the user views the run list page
- **THEN** a sun/moon toggle button SHALL be visible in the footer

#### Scenario: Toggle visible on run detail
- **WHEN** the user views a run detail page
- **THEN** the same toggle button SHALL be in the footer

