## ADDED Requirements

### Requirement: Chain run detail view with DAG visualization
The web UI SHALL provide a ChainRunDetailView that renders the chain's steps as a vertical DAG graph with live status indicators per node.

#### Scenario: Render a chain run DAG
- **WHEN** a user navigates to /chain-runs/{name}
- **THEN** the view displays a vertical directed graph where each node is a step
- **AND** edges represent dependsOn relationships (arrows flow top to bottom)
- **AND** each node shows the step name, referenced RunTemplate name, and current phase

#### Scenario: Live status updates on DAG nodes
- **WHEN** a step transitions from Pending to Running
- **THEN** the node's status indicator updates in real time (via SSE or polling)
- **AND** the node shows a running animation or spinner
- **AND** when the step succeeds, the node shows a success indicator

#### Scenario: Failed step highlights downstream
- **WHEN** step A fails and steps B and C depend on A
- **THEN** step A's node shows a failure indicator
- **AND** steps B and C show "Skipped" with a dimmed appearance
- **AND** the overall chain run status banner shows "Failed"

#### Scenario: Click a step node to view its AgentRun
- **WHEN** a user clicks on a step node that has an agentRunRef
- **THEN** the view navigates to the AgentRun detail page for that step's run

#### Scenario: Cancel chain run from the detail view
- **WHEN** a user clicks the "Cancel" button on a running ChainRunDetailView
- **THEN** the system calls POST /api/v1/chain-runs/{name}/cancel
- **AND** the DAG nodes update to reflect cancellation

### Requirement: Schedule list view
The web UI SHALL provide a ScheduleListView that shows all Schedules with their status, cron expression, target, and last/next fire times.

#### Scenario: Render schedule list
- **WHEN** a user navigates to /schedules
- **THEN** the view displays a table of all Schedules
- **AND** each row shows: name, cron expression (human-readable), target (RunTemplate or Chain name), status (Active/Suspended), lastFireTime, nextFireTime, executionCount

#### Scenario: Suspend a schedule from the list
- **WHEN** a user clicks the suspend toggle on an active Schedule row
- **THEN** the system calls POST /api/v1/schedules/{name}/suspend
- **AND** the row updates to show "Suspended" status

#### Scenario: Resume a schedule from the list
- **WHEN** a user clicks the suspend toggle on a suspended Schedule row
- **THEN** the system calls POST /api/v1/schedules/{name}/resume
- **AND** the row updates to show "Active" status with a computed nextFireTime

#### Scenario: Trigger a schedule manually from the list
- **WHEN** a user clicks the "Trigger Now" button on a Schedule row
- **THEN** the system calls POST /api/v1/schedules/{name}/trigger
- **AND** navigates to the newly created run's detail page

#### Scenario: View schedule execution history
- **WHEN** a user clicks on a Schedule name in the list
- **THEN** the view expands or navigates to show the schedule's recent runs (up to history limit)
- **AND** each run shows its phase, duration, and a link to the run detail page

### Requirement: Trigger chain button
The web UI SHALL provide a "Trigger Chain" action accessible from the Chain list and Chain detail views.

#### Scenario: Trigger a chain from the chain list
- **WHEN** a user clicks "Trigger" on a Chain row in the chain list view
- **THEN** the system calls POST /api/v1/chains/{name}/trigger
- **AND** navigates to the new ChainRun detail page

#### Scenario: Chain list view
- **WHEN** a user navigates to /chains
- **THEN** the view displays a table of all Chains
- **AND** each row shows: name, step count, projectRef, last triggered time, and a "Trigger" button

### Requirement: Navigation integration
The web UI SHALL add navigation entries for Schedules, Chains, and Chain Runs in the application sidebar or top navigation.

#### Scenario: Navigate to schedules from sidebar
- **WHEN** a user clicks "Schedules" in the navigation
- **THEN** the application routes to /schedules and renders the ScheduleListView

#### Scenario: Navigate to chains from sidebar
- **WHEN** a user clicks "Chains" in the navigation
- **THEN** the application routes to /chains and renders the chain list view

#### Scenario: Run list shows chain context
- **WHEN** a user views the run list and a run was created by a ChainRun
- **THEN** the run row displays a badge or label showing the chain name and step name
- **AND** clicking the badge navigates to the parent ChainRun detail view
