## ADDED Requirements

### Requirement: j/k keys navigate between runs in the feed
When the feed is visible and no input element is focused, pressing `j` SHALL move the selection to the next run card and pressing `k` SHALL move the selection to the previous run card.

#### Scenario: j moves selection down
- **WHEN** the feed is visible and no input is focused
- **AND** the second run card is selected
- **AND** the user presses `j`
- **THEN** the third run card SHALL become selected
- **AND** the second run card SHALL lose its selected state

#### Scenario: k moves selection up
- **WHEN** the feed is visible and no input is focused
- **AND** the third run card is selected
- **AND** the user presses `k`
- **THEN** the second run card SHALL become selected

#### Scenario: j at the bottom of the list does nothing
- **WHEN** the last run card is selected
- **AND** the user presses `j`
- **THEN** the selection SHALL remain on the last card

#### Scenario: k at the top of the list does nothing
- **WHEN** the first run card is selected
- **AND** the user presses `k`
- **THEN** the selection SHALL remain on the first card

#### Scenario: Selection scrolls into view
- **WHEN** j or k moves selection to a card outside the visible scroll area
- **THEN** the feed SHALL scroll to bring the newly selected card into view

### Requirement: Enter opens the detail view for the selected run
When a run card is selected in the feed and the user presses `Enter`, the detail view SHALL open for that run.

#### Scenario: Enter opens detail
- **WHEN** a run card is selected in the feed
- **AND** the user presses `Enter`
- **THEN** the detail view SHALL open for the selected run

#### Scenario: Enter with no selection does nothing
- **WHEN** no run card is selected
- **AND** the user presses `Enter`
- **THEN** nothing SHALL happen

### Requirement: Escape closes the detail view
When the detail view is open and no input element is focused, pressing `Escape` SHALL close the detail view and return to the feed.

#### Scenario: Escape closes detail and returns to feed
- **WHEN** the detail view is open and no input is focused
- **AND** the user presses `Escape`
- **THEN** the detail view SHALL close
- **AND** the feed SHALL be displayed
- **AND** the previously selected run card SHALL retain its selected state

### Requirement: / focuses the search input
When no input element is focused, pressing `/` SHALL focus the search/filter input, allowing the user to begin typing a search query immediately.

#### Scenario: / focuses search
- **WHEN** no input element is focused
- **AND** the user presses `/`
- **THEN** the search input SHALL receive focus
- **AND** the `/` character SHALL NOT be typed into the input

### Requirement: Keyboard shortcuts are disabled when an input is focused
All keyboard shortcuts (j, k, Enter for navigation, / for search, Escape except from detail view) SHALL be disabled when the active element is an input, textarea, select, or contenteditable element. This prevents shortcuts from interfering with text entry.

#### Scenario: j/k disabled in input
- **WHEN** a text input or textarea is focused
- **AND** the user presses `j`
- **THEN** the `j` character SHALL be typed into the input
- **AND** the feed selection SHALL NOT change

#### Scenario: / disabled in input
- **WHEN** a text input or textarea is focused
- **AND** the user presses `/`
- **THEN** the `/` character SHALL be typed into the input
- **AND** the search input SHALL NOT steal focus

### Requirement: Visual indicator of the keyboard-selected item
The currently keyboard-selected run card SHALL have a distinct visual indicator so the user can see which item will be acted upon by Enter. This visual indicator SHALL be the same as the click-selected state defined in the card-feed-layout spec.

#### Scenario: Selected card is visually distinct
- **WHEN** the user navigates with j/k
- **THEN** the selected card SHALL display the selected visual state (accent border/outline, elevated background)

#### Scenario: First card is selected by default when using keyboard
- **WHEN** the feed loads and the user presses `j` or `k` for the first time
- **AND** no card was previously selected
- **THEN** the first card in the feed SHALL become selected
