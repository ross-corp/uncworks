# ui-views Specification

## Purpose
TBD - created by archiving change ui-rewrite. Update Purpose after archive.
## Requirements
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

### Requirement: Polling effects guard against stale state updates
All React components that use `setInterval` for polling SHALL use a `cancelled` flag to prevent state updates on unmounted components. The pattern MUST be: declare `let cancelled = false` at the start of the effect, set `cancelled = true` in the cleanup function, and check `if (cancelled) return` before every `setState` call inside the polling callback.

#### Scenario: Unmounted component does not update state
- **WHEN** a component with a polling interval is unmounted
- **WHEN** an in-flight fetch completes after the cleanup runs
- **THEN** no setState is called
- **THEN** no "state update on unmounted component" warning is emitted

### Requirement: App does not crash on single-view errors
The application layout SHALL wrap the main view outlet with an ErrorBoundary component so that an uncaught error in one view does not crash the entire application.

#### Scenario: View-level error is contained
- **WHEN** a view component throws an uncaught error during render
- **THEN** the ErrorBoundary renders a fallback UI
- **THEN** the GlobalNav and other shell components remain functional

### Requirement: Destructive actions use AlertDialog for confirmation
All destructive actions (deleting a run, archiving a run with workspace deletion) SHALL use the existing `AlertDialog` component for confirmation instead of `window.confirm()`.

#### Scenario: Delete run shows confirmation dialog
- **WHEN** a user clicks "Delete" on a run
- **THEN** an AlertDialog appears with a description of the action
- **THEN** the deletion proceeds only after the user confirms in the dialog

#### Scenario: Archive with workspace deletion shows confirmation
- **WHEN** a user triggers archive with workspace deletion
- **THEN** an AlertDialog appears warning that the workspace will be deleted
- **THEN** the action proceeds only after explicit confirmation

### Requirement: Persistent labeled filter bar in RunListView
RunListView SHALL show a persistent filter bar with labeled controls rather than hidden vim-key-activated modes.

#### Scenario: Filter bar always visible
- **WHEN** the user views the run list
- **THEN** a filter input is visible with a field selector (Name / State / Stage / Model)
- **AND** vim keys (/, ?, ', ") still work as shortcuts to activate the corresponding field

#### Scenario: Active filter shown with clear button
- **WHEN** a filter is active
- **THEN** the filter bar shows what field and value is filtering
- **AND** an "×" button clears all active filters at once

### Requirement: RunListView row column order optimized for scanning
Run rows SHALL order columns: status badge → name → PR+CI (unified) → cost → diff → model → age.

#### Scenario: PR and CI status shown as unified chips
- **WHEN** a run has a PR URL
- **THEN** a "PR" chip is shown in the external-status column
- **WHEN** a run has CI status
- **THEN** a "CI ✓" or "CI ✗" chip is shown adjacent to the PR chip in the same column

### Requirement: Error toasts on all async failures in RunListView
All silent catch blocks in RunListView SHALL be replaced with error toasts.

#### Scenario: Fetch failure shows toast
- **WHEN** the run list fetch fails
- **THEN** a toast shows "Failed to load runs" and the list shows the last known state

#### Scenario: Bulk archive failure shows toast
- **WHEN** a bulk archive API call fails
- **THEN** a toast shows "Archive failed — try again"

### Requirement: Bulk archive shows feedback
Bulk archive operations SHALL show progress and completion toasts.

#### Scenario: Archive operation shows in-progress toast
- **WHEN** the user confirms bulk archive
- **THEN** a toast immediately shows "Archiving N runs..."
- **AND** on completion shows "N runs archived"

