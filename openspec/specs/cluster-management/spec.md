## ADDED Requirements

### Requirement: Cluster setup command
The system SHALL have a `task cluster:setup` target that installs all systemd units, enables lingering, and starts the cluster.

#### Scenario: First-time setup
- **WHEN** `task cluster:setup` is run
- **THEN** it SHALL enable loginctl linger, install all unit files to `~/.config/systemd/user/`, reload the systemd daemon, and start `aot-cluster.target`

### Requirement: Cluster status command
The system SHALL have a `task cluster:status` target that shows the health of all services.

#### Scenario: All services healthy
- **WHEN** `task cluster:status` is run and all services are running
- **THEN** it SHALL display the status of each service (active/inactive/failed) and listening ports

### Requirement: Cluster teardown command
The system SHALL have a `task cluster:teardown` target that stops all services and removes unit files.

#### Scenario: Full teardown
- **WHEN** `task cluster:teardown` is run
- **THEN** it SHALL stop `aot-cluster.target`, disable all units, and remove unit files from `~/.config/systemd/user/`

### Requirement: Cluster logs command
The system SHALL have a `task cluster:logs` target that shows combined logs from all services.

#### Scenario: View logs
- **WHEN** `task cluster:logs` is run
- **THEN** it SHALL display interleaved journalctl output from all `aot-*` units

### Requirement: Ollama model pre-pull
The `task cluster:setup` target SHALL ensure Ollama is deployed in k0s and the qwen2.5:0.5b model is pulled.

#### Scenario: Model available after setup
- **WHEN** `task cluster:setup` completes
- **THEN** Ollama is running in k0s and `qwen2.5:0.5b` is available for inference

### Requirement: Image build and import
The `task cluster:setup` target SHALL build and import local Docker images into k0s.

#### Scenario: Images available after setup
- **WHEN** `task cluster:setup` completes
- **THEN** `aot-agent:local`, `aot-sidecar:local`, and `aot-init:local` are available in k0s
