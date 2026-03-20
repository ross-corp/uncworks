## MODIFIED Requirements

### Requirement: Workspace path references use correct layout

All code comments, proto comments, and spec documents that reference the workspace directory layout SHALL use `/workspace/<repo>/` instead of the deprecated `/workspace/src/` path.

#### Scenario: Go type comment references correct path
- **GIVEN** `api/v1alpha1/types.go` line 66 contains a workspace path reference
- **WHEN** a developer reads the comment
- **THEN** the path shown is `/workspace/<repo>/`, not `/workspace/src/`

#### Scenario: Proto comment references correct path
- **GIVEN** `proto/aot/api/v1/api.proto` line 81 contains a workspace path reference
- **WHEN** a developer reads the comment
- **THEN** the path shown is `/workspace/<repo>/`, not `/workspace/src/`

#### Scenario: Sidecar-exec spec uses correct path
- **GIVEN** the sidecar-exec spec contains a workspace path example
- **WHEN** a developer reads the spec
- **THEN** the path shown is `/workspace/<repo>/`, not `/workspace/src/`

### Requirement: Doc staleness script excludes Helm value patterns

The doc staleness script SHALL NOT flag dotted-notation strings (e.g., `web.port`, `worker.image`) as stale references.

#### Scenario: Helm value not flagged as stale
- **GIVEN** a document contains the string `web.port`
- **WHEN** the staleness script runs
- **THEN** `web.port` is not reported as a stale reference

#### Scenario: Actual stale path still flagged
- **GIVEN** a document references a file path that no longer exists
- **WHEN** the staleness script runs
- **THEN** the stale path is still reported
