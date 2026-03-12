## ADDED Requirements

### Requirement: API server has RBAC for AgentRun CRDs
The Helm chart SHALL grant the API server deployment permissions to create, get, list, and watch AgentRun CRDs via the shared ServiceAccount.

#### Scenario: API server creates CRDs in-cluster
- **WHEN** the API server is deployed via the Helm chart
- **AND** a client calls `CreateAgentRun`
- **THEN** the API server successfully creates an AgentRun CRD using its ServiceAccount

### Requirement: API server deployment references ServiceAccount
The API server Deployment template SHALL set `serviceAccountName` to the shared ServiceAccount used by the controller.

#### Scenario: API server pod has ServiceAccount
- **WHEN** the Helm chart is rendered
- **THEN** the apiserver Deployment template includes `serviceAccountName` matching the controller's ServiceAccount
