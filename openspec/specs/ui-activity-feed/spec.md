# ui-activity-feed Specification

## Purpose
TBD - created by archiving change ui-rewrite. Update Purpose after archive.
## Requirements
### Requirement: Activity feed shows structured agent conversation
The run detail default view SHALL display a timestamped activity feed with typed entries: user messages, agent responses, tool calls, tool results, and system events.

#### Scenario: User prompt displayed
- **WHEN** a run has a user prompt
- **THEN** it appears as a "user" entry with the full prompt text

#### Scenario: Agent response rendered as markdown
- **WHEN** the agent produces a text response
- **THEN** it appears as an "agent" entry rendered as markdown with syntax highlighting

#### Scenario: Tool call with expandable input
- **WHEN** the agent calls a tool (bash, read, write, etc.)
- **THEN** it appears as a "tool" entry with the tool name and a collapsible JSON input section

#### Scenario: Tool result with truncation
- **WHEN** a tool returns output longer than 200 characters
- **THEN** the result is truncated with an "expand" button to show full output

#### Scenario: Write tool shows inline diff
- **WHEN** the agent uses the write tool to modify a file
- **THEN** the result shows an inline diff with green (added) and red (removed) lines

### Requirement: Activity feed auto-scrolls during streaming
The activity feed SHALL auto-scroll to the bottom when new entries arrive, unless the user has scrolled up.

#### Scenario: Auto-scroll while streaming
- **WHEN** new activity entries arrive and the user has not scrolled up
- **THEN** the feed scrolls to show the latest entry

#### Scenario: Scroll lock when user scrolls up
- **WHEN** the user scrolls up in the feed
- **THEN** auto-scroll stops until the user scrolls back to the bottom

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

