## ADDED Requirements

### Requirement: GET /api/v1/counts returns lightweight badge counts
The system SHALL expose `GET /api/v1/counts` in the Go backend (`internal/server/counts.go`) that returns a JSON object with total and active counts for all major entity types. The response body SHALL conform to:
```json
{
  "runs": 42,
  "activeRuns": 3,
  "projects": 8,
  "templates": 5,
  "chains": 2,
  "chainruns": 12,
  "schedules": 4
}
```
where `activeRuns` is the count of runs with phase `running`, `pending`, or `waiting_for_input`.

#### Scenario: Successful counts response
- **WHEN** an authenticated client sends `GET /api/v1/counts`
- **THEN** the server responds with HTTP 200 and a JSON body containing integer counts for all fields

#### Scenario: Empty system
- **WHEN** no entities exist
- **THEN** all count fields SHALL be `0` (not null or omitted)

### Requirement: GlobalNav uses /api/v1/counts instead of full list fetches
The system SHALL update `GlobalNav.tsx` to call only `GET /api/v1/counts` for badge data, replacing the current 6 parallel full-list fetches.

#### Scenario: Badge counts displayed after counts fetch
- **WHEN** GlobalNav polls and receives a counts response
- **THEN** all nav item badges reflect the values from the counts response

#### Scenario: Reduced network payload
- **WHEN** GlobalNav polls every 10 seconds
- **THEN** the total payload per poll cycle SHALL be the counts response only (not 6 full list responses)
