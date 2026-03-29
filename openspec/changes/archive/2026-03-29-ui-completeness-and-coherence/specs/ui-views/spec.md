## MODIFIED Requirements

### Requirement: GlobalNav items
GlobalNav SHALL include nav items in this order: Runs, Projects, Templates, Chains, Chain Runs, Schedules. Templates SHALL use icon `◻`. Chain Runs SHALL use a distinct icon from Chains (e.g. `↩` or `⛓↩`). Each item shows a count badge. Templates count = total template count.

#### Scenario: Templates nav item visible
- **WHEN** GlobalNav renders
- **THEN** "Templates" link to /templates is visible between Projects and Chains

#### Scenario: Distinct icons
- **WHEN** GlobalNav renders
- **THEN** Chains and Chain Runs have visually distinct icons

### Requirement: ScheduleListView header
ScheduleListView header SHALL contain only the title "Schedules" and a "+ new schedule" button. The "Runs" and "Chains" navigation buttons SHALL be removed.

#### Scenario: No redundant nav buttons
- **WHEN** /schedules renders
- **THEN** header has no "Runs" or "Chains" buttons

#### Scenario: New schedule button
- **WHEN** user clicks "+ new schedule"
- **THEN** navigates to /schedules/new

### Requirement: ChainListView header
ChainListView header SHALL contain the title "Chains" and a "+ new chain" button linking to /chains/new.

#### Scenario: New chain button
- **WHEN** user clicks "+ new chain"
- **THEN** navigates to /chains/new
