## ADDED Requirements

### Requirement: BFF serves static files and proxies API
The system SHALL have a Go BFF server that serves the web dashboard static files and proxies all API requests to the apiserver.

#### Scenario: Static file serving
- **WHEN** a browser requests `/` or any SPA route (e.g., `/run/ar-abc123`)
- **THEN** the BFF SHALL serve the embedded `index.html` from the static file bundle

#### Scenario: API proxy
- **WHEN** the frontend sends a request to `/api/v1/runs`
- **THEN** the BFF SHALL forward it to the apiserver and return the response

#### Scenario: WebSocket proxy
- **WHEN** the frontend sends a WebSocket upgrade to `/api/v1/runs/{id}/exec`
- **THEN** the BFF SHALL proxy the WebSocket connection to the apiserver natively (no nginx map hack)

### Requirement: BFF replaces nginx deployment
The system SHALL deploy the BFF as a single container replacing the current nginx + static files deployment.

#### Scenario: Single container
- **WHEN** the Helm chart deploys aot-bff
- **THEN** it SHALL serve both static files and API proxy on port 3000 from a single Go binary with embedded assets
