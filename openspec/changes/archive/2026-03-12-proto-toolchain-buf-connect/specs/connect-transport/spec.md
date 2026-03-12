## ADDED Requirements

### Requirement: single-port multi-protocol server

The API server SHALL serve gRPC, gRPC-Web, and Connect protocols on a single port (`:50051`).

#### Scenario: gRPC client connects

- **WHEN** a gRPC client (TUI, CLI) connects to `:50051` using the standard gRPC protocol over h2c
- **THEN** the server handles the request and returns a valid gRPC response

#### Scenario: Connect client connects

- **WHEN** a browser client sends a Connect-protocol HTTP request to `:50051`
- **THEN** the server handles the request and returns a valid Connect response

#### Scenario: gRPC-Web client connects

- **WHEN** a gRPC-Web client sends a request to `:50051`
- **THEN** the server handles the request and returns a valid gRPC-Web response

### Requirement: ConnectRPC Go module

The server SHALL use the `connectrpc.com/connect` Go module with Go's standard `net/http` server. The server MUST NOT use `google.golang.org/grpc` as the server implementation.

#### Scenario: server uses net/http

- **WHEN** the API server starts
- **THEN** it creates an `http.Server` and registers Connect handlers on an `http.ServeMux`

### Requirement: browser streaming via Connect server-streaming

Browser clients SHALL consume the `WatchAgentRun` RPC via Connect server-streaming, which uses Server-Sent Events (SSE) over HTTP/1.1 for the Connect protocol.

#### Scenario: browser watches an agent run

- **WHEN** a browser client calls `WatchAgentRun` using the Connect protocol
- **THEN** the server responds with a server-streaming response delivered as SSE events over HTTP/1.1
- **THEN** each status update for the agent run is delivered as a stream message

### Requirement: WebSocket endpoint removal

The WebSocket endpoint SHALL be removed. There SHALL be no `/ws` path served by the API server. The `:8080` HTTP server SHALL be deleted.

#### Scenario: WebSocket path returns not found

- **WHEN** a client sends a WebSocket upgrade request to the API server
- **THEN** the server does not upgrade the connection and no WebSocket handler exists

#### Scenario: port 8080 is not bound

- **WHEN** the API server is running
- **THEN** no listener is bound to port `:8080`

### Requirement: all RPCs served via Connect handlers

All existing gRPC RPCs SHALL be served via Connect handlers. This includes `CreateAgentRun`, `GetAgentRun`, `ListAgentRuns`, `WatchAgentRun`, `CancelAgentRun`, and `SendHumanInput`.

#### Scenario: CreateAgentRun via Connect

- **WHEN** a client sends a `CreateAgentRun` request using any supported protocol (gRPC, Connect, gRPC-Web)
- **THEN** the server processes the request through its Connect handler and returns a response

#### Scenario: all RPCs are registered

- **WHEN** the API server starts
- **THEN** Connect handlers are registered for all six RPCs: `CreateAgentRun`, `GetAgentRun`, `ListAgentRuns`, `WatchAgentRun`, `CancelAgentRun`, `SendHumanInput`

### Requirement: sidecar services use Connect handlers

The sidecar API services (`AgentSidecarService`, `AgentNotificationService`) SHALL also use Connect handlers.

#### Scenario: sidecar reports status via Connect

- **WHEN** an agent sidecar calls an `AgentSidecarService` RPC
- **THEN** the server processes the request through its Connect handler

#### Scenario: sidecar notifications via Connect

- **WHEN** an agent sidecar calls an `AgentNotificationService` RPC
- **THEN** the server processes the request through its Connect handler

### Requirement: generated TypeScript client replaces hand-written client

The generated TypeScript Connect client from `gen/ts/` SHALL replace the hand-written `@aot/shared/grpc/client.ts`. The hand-written client file SHALL be deleted.

#### Scenario: hand-written client is removed

- **WHEN** a developer inspects the `packages/shared/grpc/` directory
- **THEN** `client.ts` does not exist

#### Scenario: web dashboard imports generated client

- **WHEN** the web dashboard imports a service client
- **THEN** the import resolves to a generated Connect client from `gen/ts/`
