## ADDED Requirements

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
