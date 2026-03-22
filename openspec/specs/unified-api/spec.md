# unified-api Specification

## Purpose
TBD - created by archiving change backend-for-frontend. Update Purpose after archive.
## Requirements
### Requirement: Unified REST API surface
The system SHALL expose a single REST API surface for the frontend, eliminating the dual ConnectRPC + REST protocol split.

#### Scenario: List runs via REST
- **WHEN** the frontend sends `GET /api/v1/runs`
- **THEN** the BFF SHALL return a JSON array of runs (internally calling the ConnectRPC ListAgentRuns endpoint)

#### Scenario: Create run via REST
- **WHEN** the frontend sends `POST /api/v1/runs` with a JSON body
- **THEN** the BFF SHALL create the run via ConnectRPC and return the created run as JSON

#### Scenario: Cancel run via REST
- **WHEN** the frontend sends `POST /api/v1/runs/{id}/cancel`
- **THEN** the BFF SHALL cancel via ConnectRPC and return the updated run

