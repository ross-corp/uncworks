## ADDED Requirements

### Requirement: Soft-Serve git server lifecycle
The test harness SHALL manage a Soft-Serve git server process that starts before tests and stops after, providing cloneable repositories via git:// protocol.

#### Scenario: Server starts and accepts connections
- **WHEN** `task test:e2e:setup` is executed
- **THEN** a Soft-Serve process starts listening on the configured git daemon port (default 9418)
- **AND** the process is reachable from both the host and k0s pods

#### Scenario: Server stops cleanly
- **WHEN** `task test:e2e:teardown` is executed
- **THEN** the Soft-Serve process is stopped and its data directory is cleaned up

### Requirement: Fixture repo provisioning
The test harness SHALL push fixture repositories to Soft-Serve during setup, making them available for agent runs to clone.

#### Scenario: Standard fixture repo available
- **WHEN** test setup completes
- **THEN** a repository `e2e-repo` is cloneable at `git://{SOFT_SERVE_ADDR}/e2e-repo`
- **AND** it contains `devbox.json`, `main.go`, and `README.md`

#### Scenario: Multi-repo fixture repos available
- **WHEN** test setup completes
- **THEN** repositories `e2e-repo` and `e2e-repo-frontend` are both cloneable
- **AND** each contains appropriate fixture content for testing multi-repo workspaces

### Requirement: Test environment configuration
The test harness SHALL use environment variables for all external service addresses, with sensible defaults for the aot-local cluster.

#### Scenario: Default configuration
- **WHEN** no environment variables are set
- **THEN** tests connect to API at `localhost:50055`, Temporal at `localhost:7233`, Soft-Serve at `localhost:9418`, and web UI at `localhost:3000`

#### Scenario: Custom configuration
- **WHEN** `AOT_API_URL`, `TEMPORAL_HOST`, `SOFT_SERVE_ADDR`, or `VITE_API_URL` are set
- **THEN** tests use the specified addresses

### Requirement: Taskfile test orchestration
The test harness SHALL provide Taskfile commands for running E2E tests with proper setup and teardown.

#### Scenario: Full E2E run
- **WHEN** `task test:e2e:full` is executed
- **THEN** Soft-Serve starts, fixtures are pushed, Go E2E tests run, Playwright tests run, and Soft-Serve stops
- **AND** the exit code reflects whether all tests passed

#### Scenario: Iterative development
- **WHEN** developer runs `task test:e2e:setup` followed by `task test:e2e:go` multiple times
- **THEN** each Go E2E run uses the already-running Soft-Serve instance without re-setup
