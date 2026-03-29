## ADDED Requirements

### Requirement: ActivityFeed polling uses usePoll hook
The ActivityFeed component SHALL implement its polling via the `usePoll` hook rather than inline `setInterval` + `cancelled` flag logic. Polling behavior (interval, cancellation on unmount) SHALL remain identical to the previous implementation.

#### Scenario: ActivityFeed polling delegates to usePoll
- **WHEN** ActivityFeed mounts and begins polling for new events
- **THEN** it uses `usePoll(fetchFn, intervalMs)` and does not contain inline `setInterval` calls

#### Scenario: Unmount cancels pending fetch
- **WHEN** ActivityFeed unmounts while a fetch is in progress
- **THEN** the `usePoll` hook ensures no state update is applied from that fetch
