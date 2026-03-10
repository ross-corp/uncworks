## ADDED Requirements

### Requirement: Temporal Workflow Unit Tests
Temporal workflow logic SHALL be tested using the Temporal SDK test suite with mocked activities.

#### Scenario: Test framework
- **GIVEN** the Temporal workflow test suite
- **THEN** it SHALL use `go.temporal.io/sdk/testsuite` for workflow unit tests
- **AND** all activities SHALL be mocked

#### Scenario: Happy path
- **GIVEN** a workflow unit test for the agent run workflow
- **WHEN** the workflow executes the happy path
- **THEN** it SHALL verify the sequence: create pod -> hydrate -> start -> complete

#### Scenario: TTL expiry
- **GIVEN** a workflow unit test with a TTL configured
- **WHEN** the timer is fast-forwarded past the TTL
- **THEN** the workflow SHALL expire and clean up resources

#### Scenario: HITL signal flow
- **GIVEN** a workflow unit test for human-in-the-loop
- **WHEN** a HITL signal is received by the workflow
- **THEN** the corresponding activity SHALL be called to deliver the input to the agent

#### Scenario: Cancel signal flow
- **GIVEN** a workflow unit test for cancellation
- **WHEN** a cancel signal is received by the workflow
- **THEN** the agent SHALL be stopped
- **AND** the pod SHALL be cleaned up

#### Scenario: Spawn junior as child workflow
- **GIVEN** a workflow unit test for multi-agent orchestration
- **WHEN** a spawn_junior request is processed
- **THEN** a child workflow SHALL be started

#### Scenario: Compensation on activity failure
- **GIVEN** a workflow unit test where an activity fails
- **WHEN** the failure is detected
- **THEN** compensation logic SHALL execute (pod cleanup)

### Requirement: Temporal Integration Tests
Temporal workflows SHALL be tested end-to-end against a real Temporal server.

#### Scenario: Dev server usage
- **GIVEN** the Temporal integration test suite
- **THEN** it SHALL use `temporal-cli` dev server (started automatically)

#### Scenario: Real workflow execution
- **GIVEN** a Temporal integration test
- **WHEN** the test executes
- **THEN** it SHALL run real workflows with real activities against test infrastructure

### Requirement: Temporal Test Organization
Temporal tests SHALL be organized and runnable via Taskfile.

#### Scenario: Task target
- **GIVEN** the Taskfile
- **WHEN** `task test:temporal` is executed
- **THEN** it SHALL run all Temporal tests (both unit and integration)

#### Scenario: File location
- **GIVEN** the Temporal test files
- **THEN** they SHALL be located at `test/temporal/`
