## ADDED Requirements

### Requirement: Client-side routing with shareable URLs
The web UI SHALL use @solidjs/router for client-side navigation. Route `/` SHALL display the run list. Route `/runs/:id` SHALL display the run detail with event streaming. URLs SHALL be shareable — navigating directly to `/runs/:id` SHALL load that run.

#### Scenario: Navigate from list to detail
- **WHEN** user clicks a run in the list
- **THEN** the browser URL changes to `/runs/:id`
- **AND** the detail view loads for that run

#### Scenario: Direct navigation to run detail
- **WHEN** user opens `/runs/abc123` directly
- **THEN** the detail view fetches run `abc123` and displays it
- **AND** the watchAgentRun stream starts for that run

#### Scenario: Navigate back to list
- **WHEN** user is on a detail page and navigates back
- **THEN** the list view loads and the event stream is cleaned up

### Requirement: Store-first architecture
App.tsx SHALL use createAgentStore from packages/shared instead of local signals. The store SHALL be the single source of truth for runs, events, selection, and filter state.

#### Scenario: List page populates store
- **WHEN** the list page loads
- **THEN** it calls listAgentRuns and populates the store via setRuns

#### Scenario: Detail page updates store from stream
- **WHEN** watchAgentRun emits events on the detail page
- **THEN** phase_changed events update the run in the store via addEvent
- **AND** all events are appended to the store's event list
