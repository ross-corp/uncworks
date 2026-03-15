## ADDED Requirements

### Requirement: RunCard component displays run status and metadata at a glance
The RunCard component SHALL render a single agent run as a card containing: a status dot (colored circle), the run name (bold), the repository name (muted), a one-line prompt preview (truncated), and a relative time-ago label. The status dot color SHALL use semantic color tokens (success/active/warning/error/neutral).

#### Scenario: RunCard renders all required fields
- **WHEN** a RunCard is rendered with a run object
- **THEN** the card SHALL display the run name in bold, the repo name in muted text, the first line of the prompt truncated to a single line with ellipsis, and a relative time label (e.g., "3m ago", "2h ago")

#### Scenario: Status dot reflects run phase
- **WHEN** a RunCard is rendered for a run with phase "Running"
- **THEN** the status dot SHALL use the `--color-active` semantic token (blue) and SHALL have a CSS pulse animation
- **WHEN** a RunCard is rendered for a run with phase "Succeeded"
- **THEN** the status dot SHALL use the `--color-success` semantic token (green) with no animation
- **WHEN** a RunCard is rendered for a run with phase "Failed"
- **THEN** the status dot SHALL use the `--color-error` semantic token (red) with no animation
- **WHEN** a RunCard is rendered for a run with phase "Pending"
- **THEN** the status dot SHALL use the `--color-warning` semantic token (amber) with no animation
- **WHEN** a RunCard is rendered for a run with phase "Cancelled"
- **THEN** the status dot SHALL use the `--color-neutral` semantic token (gray) with no animation

#### Scenario: Status dot has pulse animation for active runs
- **WHEN** a run has phase "Running"
- **THEN** the status dot SHALL have a CSS `@keyframes pulse` animation that cycles opacity between 1.0 and 0.4

### Requirement: RunFeed component renders a vertical stack of RunCards
The RunFeed component SHALL replace the existing AgentRunTable. It SHALL render RunCard components in a vertical stack, ordered by most recent first. It SHALL support empty state, loading state, and populated state.

#### Scenario: RunFeed renders cards in reverse-chronological order
- **WHEN** the RunFeed receives a list of runs
- **THEN** it SHALL render one RunCard per run, ordered by creation time descending (newest first)

#### Scenario: RunFeed displays empty state
- **WHEN** the RunFeed receives an empty list of runs and is not loading
- **THEN** it SHALL display a centered message: "No runs yet" with a prompt to create one

#### Scenario: RunFeed displays loading state
- **WHEN** the RunFeed is loading data
- **THEN** it SHALL display skeleton placeholder cards (at least 3) using the Skeleton component

#### Scenario: RunFeed is scrollable
- **WHEN** the number of RunCards exceeds the visible area
- **THEN** the RunFeed container SHALL be vertically scrollable with overflow-y auto

### Requirement: Clicking a RunCard opens the detail view
When a user clicks a RunCard, the application SHALL navigate to the detail view for that run. The card click SHALL be the sole entry point to run detail from the feed.

#### Scenario: Card click navigates to detail
- **WHEN** the user clicks anywhere on a RunCard
- **THEN** the application SHALL open the detail view for that run, replacing the feed content

#### Scenario: Card is a clickable region
- **WHEN** a RunCard is rendered
- **THEN** the entire card surface SHALL be clickable (not just the name)
- **AND** the card SHALL have `cursor: pointer`
- **AND** the card SHALL have a hover state (subtle background change)

### Requirement: Selected card has a visual indicator
When a RunCard is selected (its detail view is open or it is keyboard-focused), it SHALL display a distinct visual state so the user knows which run is active.

#### Scenario: Selected card visual state
- **WHEN** a RunCard is the currently selected run
- **THEN** it SHALL have a left border or outline using the `--color-accent` token
- **AND** its background SHALL be slightly elevated compared to unselected cards

#### Scenario: Only one card is selected at a time
- **WHEN** the user clicks a different RunCard
- **THEN** the previously selected card SHALL lose its selected visual state
- **AND** the newly clicked card SHALL gain the selected visual state
