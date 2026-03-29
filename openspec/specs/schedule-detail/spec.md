# schedule-detail Specification

## Purpose
TBD - created by archiving change ui-overhaul-v2. Update Purpose after archive.
## Requirements
### Requirement: Schedule detail view at /schedules/:name
A new route SHALL exist for viewing and editing a schedule's configuration and execution history.

#### Scenario: Detail view shows human-readable cron
- **WHEN** the user navigates to /schedules/:name
- **THEN** the cron expression is shown alongside its human-readable translation (e.g., "0 9 * * *" → "Daily at 9:00 AM UTC")

#### Scenario: Cron expression is editable
- **WHEN** the user clicks "Edit" on the cron expression
- **THEN** a cron editor appears with frequency and time selectors
- **AND** saving updates the schedule via the API

#### Scenario: Execution history shows recent runs
- **WHEN** the schedule detail view loads
- **THEN** the last 10 executions are shown in a table with: date, status badge, duration, link to run

#### Scenario: Suspend/resume from detail view
- **WHEN** the user clicks Suspend or Resume
- **THEN** the schedule's suspended state is toggled
- **AND** the UI updates immediately to reflect the new state

### Requirement: Human-readable cron in ScheduleListView
ScheduleListView SHALL display a human-readable cron translation alongside or instead of the raw cron string.

#### Scenario: Cron tooltip shows human-readable translation
- **WHEN** the user hovers over a cron expression in the schedule list
- **THEN** a tooltip shows the human-readable equivalent (e.g., "Every Monday at 9am UTC")

#### Scenario: Next scheduled time shown as relative time
- **WHEN** a next execution time is available
- **THEN** it is shown as relative time (e.g., "in 3 hours") not absolute datetime

