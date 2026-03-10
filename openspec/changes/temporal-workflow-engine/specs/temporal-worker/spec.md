## ADDED Requirements

### Requirement: Standalone Worker Binary
A standalone binary at `cmd/temporal-worker/main.go` SHALL connect to Temporal Frontend and register all workflows and activities.

#### Scenario: Worker startup
- **WHEN** the temporal-worker binary is started
- **THEN** it SHALL connect to the Temporal Frontend at the address specified by `TEMPORAL_HOST`
- **AND** it SHALL register `AgentRunWorkflow` and all associated activities on the configured task queue
- **AND** it SHALL begin polling for workflow tasks

#### Scenario: Worker connection failure
- **WHEN** the temporal-worker binary cannot connect to Temporal Frontend
- **THEN** it SHALL log the connection error and exit with a non-zero status code

### Requirement: Task Queue Configuration
The worker SHALL use the task queue name from the `TEMPORAL_TASK_QUEUE` environment variable with a default of `aot-agent-runs`.

#### Scenario: Default task queue
- **WHEN** `TEMPORAL_TASK_QUEUE` is not set
- **THEN** the worker SHALL listen on the `aot-agent-runs` task queue

#### Scenario: Custom task queue
- **WHEN** `TEMPORAL_TASK_QUEUE` is set to a custom value
- **THEN** the worker SHALL listen on the specified task queue

### Requirement: Graceful Shutdown
The worker SHALL gracefully shut down on SIGINT/SIGTERM.

#### Scenario: Receiving SIGTERM
- **WHEN** the worker process receives a SIGTERM signal
- **THEN** the worker SHALL stop accepting new workflow and activity tasks
- **AND** the worker SHALL wait for in-progress activities to complete (up to a configurable timeout)
- **AND** the worker SHALL exit cleanly

#### Scenario: Receiving SIGINT
- **WHEN** the worker process receives a SIGINT signal
- **THEN** the worker SHALL perform the same graceful shutdown as SIGTERM

### Requirement: Kubernetes Client Access
The worker SHALL have access to a controller-runtime K8s client for pod management activities.

#### Scenario: Creating pods from activities
- **WHEN** the `CreateAgentPod` activity executes within the worker
- **THEN** the activity SHALL use the controller-runtime K8s client to create the pod
- **AND** the client SHALL be configured via in-cluster config (when running in K8s) or kubeconfig (when running locally)

#### Scenario: Cleaning up pods from activities
- **WHEN** the `CleanupPod` activity executes within the worker
- **THEN** the activity SHALL use the same K8s client to delete the pod

### Requirement: Brain Store Access
The worker SHALL have access to the brain store for metadata persistence.

#### Scenario: Persisting agent metadata
- **WHEN** an activity needs to store agent output or trace IDs
- **THEN** the activity SHALL use the brain store client to persist the metadata
- **AND** the brain store connection SHALL be configured via the same environment variables as the main AOT server

### Requirement: Build Target Inclusion
The worker binary SHALL be included in `task build` targets.

#### Scenario: Building all binaries
- **WHEN** a developer runs `task build`
- **THEN** the build SHALL compile `cmd/temporal-worker/main.go` into a binary
- **AND** the binary SHALL be placed in the standard output directory alongside other AOT binaries

### Requirement: Execution Logging
The worker SHALL log workflow execution start/completion events.

#### Scenario: Workflow started
- **WHEN** a workflow task begins execution on the worker
- **THEN** the worker SHALL log the workflow type, workflow ID, and run ID at info level

#### Scenario: Workflow completed
- **WHEN** a workflow completes (success, failure, or cancellation)
- **THEN** the worker SHALL log the workflow ID, run ID, and terminal status at info level

#### Scenario: Activity execution
- **WHEN** an activity begins execution on the worker
- **THEN** the worker SHALL log the activity type, workflow ID, and activity ID at debug level
