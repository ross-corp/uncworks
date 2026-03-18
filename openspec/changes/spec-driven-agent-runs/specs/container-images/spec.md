## MODIFIED Requirements

### Requirement: Sidecar image includes agent tooling
The `aot-sidecar` image SHALL contain the pi-coding-agent npm package, Node.js runtime, and the OpenSpec CLI, enabling the sidecar to run planning, execution, and verification agents with OpenSpec skill support.

#### Scenario: OpenSpec CLI available in sidecar
- **WHEN** a container runs `aot-sidecar`
- **THEN** the `openspec` command is available in the container's PATH
- **AND** `openspec --version` exits with code 0
