## ADDED Requirements

### Requirement: Jump-to-latest button when scrolled up
The ActivityFeed SHALL display a "Jump to latest ↓" button when the user has scrolled up from the bottom.

#### Scenario: Button appears on scroll up
- **WHEN** the user scrolls up more than 100px from the bottom of the feed
- **THEN** a "Jump to latest ↓" button appears fixed at the bottom of the feed area

#### Scenario: Button scrolls to bottom and hides
- **WHEN** the user clicks the Jump to latest button
- **THEN** the feed scrolls to the bottom smoothly
- **AND** the button disappears

### Requirement: Error toasts on async failures
All async operations in ActivityFeed SHALL show error toasts on failure rather than failing silently.

#### Scenario: Failed stream connection shows toast
- **WHEN** the activity feed stream connection fails
- **THEN** a toast shows "Failed to connect to activity feed" with a Retry button
