## Purpose

Define the three-view architecture with keyboard-driven navigation and URL routing.

## ADDED Requirements

### Requirement: Three full-screen views with URL routing
The UI SHALL have three views at `/` (run list), `/new` (new run), and `/run/:id` (run detail), each occupying the full screen.

#### Scenario: Default view is run list
- **WHEN** the user navigates to `/`
- **THEN** the run list view is displayed with all agent runs

#### Scenario: Deep link to run detail
- **WHEN** the user navigates to `/run/ar-ju91iv`
- **THEN** the run detail view is displayed for that run

#### Scenario: Browser back/forward works
- **WHEN** the user navigates from run list to run detail and presses browser back
- **THEN** the run list view is displayed

### Requirement: k9s-style keyboard navigation
The UI SHALL support keyboard navigation: j/k to move selection, enter to drill in, esc to go back, number keys for tabs, / for filter, n for new run.

#### Scenario: j/k moves selection in run list
- **WHEN** the user presses j in the run list
- **THEN** the selection moves down one row

#### Scenario: Enter opens run detail
- **WHEN** the user presses enter on a selected run
- **THEN** the run detail view opens for that run

#### Scenario: Esc returns to previous view
- **WHEN** the user presses esc in run detail
- **THEN** the run list view is displayed

#### Scenario: Number keys switch tabs in run detail
- **WHEN** the user presses 2 in run detail
- **THEN** the files tab is displayed

### Requirement: Command palette via cmdk
The UI SHALL provide a command palette triggered by ⌘K that allows searching runs, switching views, toggling theme, and executing actions.

#### Scenario: Command palette opens
- **WHEN** the user presses ⌘K
- **THEN** the command palette overlay appears with a search input
