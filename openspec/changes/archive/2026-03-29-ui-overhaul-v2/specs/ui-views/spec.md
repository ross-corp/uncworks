## ADDED Requirements

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
