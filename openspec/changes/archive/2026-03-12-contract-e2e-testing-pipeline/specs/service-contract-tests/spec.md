## ADDED Requirements

### Requirement: Server Contract Verification
Go gRPC server implementations SHALL be verified against their proto contracts.

#### Scenario: AOTService contract coverage
- **GIVEN** the AOTService gRPC server implementation
- **WHEN** server contract tests are executed
- **THEN** they SHALL test all 6 RPCs: CreateAgentRun, GetAgentRun, ListAgentRuns, CancelAgentRun, SendHumanInput, StreamEvents

#### Scenario: AgentSidecarService contract coverage
- **GIVEN** the AgentSidecarService gRPC server implementation
- **WHEN** server contract tests are executed
- **THEN** they SHALL test all 5 RPCs: RegisterAgent, Heartbeat, ReportToolCall, AskHuman, SpawnJunior

#### Scenario: AgentNotificationService contract coverage
- **GIVEN** the AgentNotificationService gRPC server implementation
- **WHEN** server contract tests are executed
- **THEN** they SHALL test the NotifyEvent RPC

#### Scenario: Error code verification
- **GIVEN** a server contract test
- **WHEN** an invalid request is sent (e.g., missing required fields, nonexistent ID)
- **THEN** the server SHALL return the correct gRPC error code (NOT_FOUND, INVALID_ARGUMENT, FAILED_PRECONDITION, etc.)

#### Scenario: Protovalidate rule enforcement
- **GIVEN** a proto message with protovalidate rules defined
- **WHEN** a request violating those rules is sent to the server
- **THEN** the server SHALL reject the request with an appropriate error

### Requirement: Client Contract Verification
TypeScript Connect clients SHALL be verified against their proto contracts.

#### Scenario: Request serialization
- **GIVEN** a TypeScript Connect client for an AOT service
- **WHEN** the client sends a request
- **THEN** the serialized request SHALL match the proto schema as verified by a GripMock mock server

#### Scenario: Response deserialization
- **GIVEN** a GripMock mock server returning a valid proto response
- **WHEN** the TypeScript Connect client receives the response
- **THEN** it SHALL correctly deserialize all fields for all RPC response types

#### Scenario: Error handling
- **GIVEN** a GripMock mock server returning a gRPC error response
- **WHEN** the TypeScript Connect client receives the error
- **THEN** it SHALL correctly handle and surface the error code and message

### Requirement: GripMock Service Mocking
GripMock SHALL be used to create mock gRPC servers from .proto files for client contract testing.

#### Scenario: GripMock stub configuration
- **GIVEN** the contract test suite
- **THEN** GripMock stubs SHALL be defined in YAML files at `test/contract/stubs/`

#### Scenario: Dynamic stub management
- **GIVEN** a running GripMock container
- **WHEN** a contract test needs to configure specific responses
- **THEN** it MAY use GripMock's REST API for dynamic stub management during tests

### Requirement: Contract Test Isolation
Contract tests SHALL run without any external infrastructure.

#### Scenario: No infrastructure dependencies
- **GIVEN** the contract test suite
- **WHEN** `task test:contract` is executed
- **THEN** it SHALL run all contract tests
- **AND** it SHALL NOT require a database, Kubernetes cluster, or any other external infrastructure beyond Docker (for GripMock)
