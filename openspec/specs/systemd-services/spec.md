## ADDED Requirements

### Requirement: Temporal dev server unit
The system SHALL have a systemd user unit `aot-temporal.service` that runs the Temporal dev server with SQLite persistence.

#### Scenario: Temporal starts and listens
- **WHEN** `systemctl --user start aot-temporal.service` is run
- **THEN** the Temporal dev server listens on port 7233 and uses `.temporal.db` for persistence

### Requirement: AOT controller unit
The system SHALL have a systemd user unit `aot-controller.service` that runs the AOT controller with the correct KUBECONFIG and TEMPORAL_HOST.

#### Scenario: Controller starts after Temporal
- **WHEN** `aot-temporal.service` is active
- **THEN** `aot-controller.service` starts and connects to both k0s (via kubeconfig) and Temporal

#### Scenario: Controller uses non-conflicting metrics port
- **WHEN** the controller starts
- **THEN** it SHALL bind metrics to port 8095 (via METRICS_ADDR env var)

### Requirement: Temporal worker unit
The system SHALL have a systemd user unit `aot-worker.service` that runs the temporal-worker with local image configuration.

#### Scenario: Worker starts with local images
- **WHEN** `aot-worker.service` starts
- **THEN** it SHALL have `AOT_AGENT_IMAGE=aot-agent:local`, `AOT_SIDECAR_IMAGE=aot-sidecar:local`, and `AOT_INIT_IMAGE=aot-init:local` set

### Requirement: API server unit
The system SHALL have a systemd user unit `aot-apiserver.service` that runs the API server on port 50055.

#### Scenario: API server starts
- **WHEN** `aot-apiserver.service` starts
- **THEN** it SHALL listen on port 50055 and connect to Temporal at localhost:7233

### Requirement: Web UI unit
The system SHALL have a systemd user unit `aot-web.service` that runs the vite dev server for the web dashboard.

#### Scenario: Web UI starts and proxies API
- **WHEN** `aot-web.service` starts
- **THEN** the vite dev server listens on port 3000 and proxies `/aot.api.v1.AOTService` requests to the API server on port 50055

### Requirement: Cluster target group
The system SHALL have a systemd target `aot-cluster.target` that groups all AOT services.

#### Scenario: Start all services
- **WHEN** `systemctl --user start aot-cluster.target` is run
- **THEN** all AOT services start in dependency order

#### Scenario: Stop all services
- **WHEN** `systemctl --user stop aot-cluster.target` is run
- **THEN** all AOT services stop

### Requirement: Auto-restart on failure
All AOT service units SHALL restart automatically on failure with a 5-second delay.

#### Scenario: Service crashes and recovers
- **WHEN** any AOT service process exits unexpectedly
- **THEN** systemd restarts it within 5 seconds

### Requirement: Environment file configuration
Each service unit SHALL source its environment from a `.env` file in `deploy/systemd/env/`.

#### Scenario: Environment files used
- **WHEN** a service starts
- **THEN** it reads environment variables from `deploy/systemd/env/<service>.env`
