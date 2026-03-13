## ADDED Requirements

### Requirement: Loading skeletons for async data
Components that fetch data SHALL show animated skeleton placeholders while loading, matching the layout of the content they replace.

#### Scenario: Run list loading
- **WHEN** the runs list is fetching data
- **THEN** the table shows skeleton rows matching the column layout

#### Scenario: Run detail loading
- **WHEN** a run detail is being fetched
- **THEN** the detail panel shows skeleton blocks for the header, status, and event log

### Requirement: Empty states with guidance
Components that display lists SHALL show a descriptive empty state with an action prompt when the list is empty.

#### Scenario: No runs exist
- **WHEN** the user has no agent runs
- **THEN** the table shows "No agent runs yet" with a "Create Run" button

#### Scenario: No events for a run
- **WHEN** a run's event log has no events
- **THEN** the events panel shows "No events yet. Events will appear as the agent runs."

### Requirement: Error boundaries with retry
The app SHALL catch rendering errors and display a fallback UI with a retry action instead of a blank screen.

#### Scenario: Component render error
- **WHEN** a component throws during render
- **THEN** an error boundary shows "Something went wrong" with a "Retry" button
- **AND** clicking "Retry" re-renders the component

### Requirement: Toast notifications for actions
User-initiated actions (create, cancel, send input) SHALL show toast notifications for success and failure feedback.

#### Scenario: Run created successfully
- **WHEN** the user creates a new agent run
- **THEN** a success toast appears: "Agent run created"

#### Scenario: Action fails
- **WHEN** a cancel or send-input action fails
- **THEN** an error toast appears with the error message
- **AND** the toast auto-dismisses after 5 seconds

### Requirement: Components wired to real API
All components SHALL use the `AOTClient` from `packages/shared` for data fetching and mutations, replacing the mock data layer from the `ui` branch.

#### Scenario: Run list fetches from API
- **WHEN** the runs page loads
- **THEN** it calls `AOTClient.listAgentRuns()` and displays the results

#### Scenario: Run detail streams events
- **WHEN** a run detail page loads
- **THEN** it calls `AOTClient.watchAgentRun()` and appends events in real-time

#### Scenario: HITL input submission
- **WHEN** the user submits input on a waiting run
- **THEN** it calls `AOTClient.sendHumanInput()` and shows a success toast
