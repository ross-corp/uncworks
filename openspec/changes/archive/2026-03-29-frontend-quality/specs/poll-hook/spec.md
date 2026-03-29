## ADDED Requirements

### Requirement: usePoll hook provides interval polling with cancellation
The system SHALL provide a `usePoll(fn, intervalMs, deps?)` React hook in `web/src/hooks/usePoll.ts` that invokes `fn` immediately on mount, then repeatedly at `intervalMs` intervals, and cancels any in-flight invocation and the interval timer on unmount or when `deps` change.

#### Scenario: Immediate invocation on mount
- **WHEN** a component mounts with `usePoll(fetchData, 5000)`
- **THEN** `fetchData` is called once immediately before the first interval fires

#### Scenario: Repeated invocation at interval
- **WHEN** the component remains mounted
- **THEN** `fetchData` is called again every `intervalMs` milliseconds

#### Scenario: Cancellation on unmount
- **WHEN** the component unmounts while a `fetchData` call is in progress
- **THEN** any state updates triggered by that call SHALL be suppressed and the interval SHALL be cleared

#### Scenario: Deps change resets polling
- **WHEN** a value in the `deps` array changes
- **THEN** the current interval is cleared and a new polling cycle begins immediately

### Requirement: All six polling sites use usePoll
The system SHALL migrate RunListView, ProjectListView, ActivityFeed, TraceTimeline, RunDetailView, and GlobalNav to use `usePoll` in place of inline `setInterval` + `cancelled` flag patterns.

#### Scenario: Equivalent polling behavior after migration
- **WHEN** any of the six components is mounted
- **THEN** its data-fetch function is invoked at the same interval as before migration, with correct cleanup on unmount
