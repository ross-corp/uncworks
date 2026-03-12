## ADDED Requirements

### Requirement: Helm chart installs all AOT components
The Helm chart SHALL deploy the CRD, controller, temporal-worker, API server, web dashboard, ServiceAccounts, and RBAC when installed.

#### Scenario: Fresh install
- **WHEN** `helm install aot oci://ghcr.io/uncworks/charts/aot --set temporal.host=temporal:7233` is run
- **THEN** all AOT components are deployed and running in the target namespace

### Requirement: Temporal host is required
The chart SHALL require `temporal.host` to be set and fail validation if it is missing.

#### Scenario: Missing temporal host
- **WHEN** the chart is installed without `temporal.host` set
- **THEN** Helm renders an error explaining the required value

### Requirement: LLM configuration is optional
The chart SHALL allow optional configuration of LLM endpoint via `llm.baseUrl` and `llm.apiKey`.

#### Scenario: LLM configured
- **WHEN** `llm.baseUrl` and `llm.apiKey` are set in values
- **THEN** the worker deployment injects `OPENAI_BASE_URL` and `OPENAI_API_KEY` environment variables into agent pods

#### Scenario: LLM not configured
- **WHEN** `llm.baseUrl` is not set
- **THEN** agent pods start without LLM env vars (agents use their own default)

### Requirement: Image references are configurable
The chart SHALL allow overriding all image references via `images.*` values with `repository`, `tag`, and `pullPolicy` fields.

#### Scenario: Custom image tag
- **WHEN** `images.controlplane.tag` is set to `v0.2.0`
- **THEN** all control plane deployments use `ghcr.io/uncworks/aot-controlplane:v0.2.0`

### Requirement: Web dashboard is optional
The chart SHALL support disabling the web dashboard via `web.enabled: false`.

#### Scenario: Dashboard disabled
- **WHEN** `web.enabled` is set to `false`
- **THEN** no web Deployment or Service is created

### Requirement: Service type is configurable
The web dashboard service type SHALL be configurable via `web.service.type`.

#### Scenario: NodePort access
- **WHEN** `web.service.type` is set to `NodePort`
- **THEN** the web Service is created with type NodePort

### Requirement: Chart is published as OCI artifact
The Helm chart SHALL be published to `ghcr.io/uncworks/charts/aot` as an OCI artifact on each tagged release.

#### Scenario: Install from registry
- **WHEN** `helm install aot oci://ghcr.io/uncworks/charts/aot --version 0.1.0` is run
- **THEN** Helm pulls the chart from ghcr.io and installs it

### Requirement: No bundled dependencies
The chart SHALL NOT include Temporal, Ollama, PostgreSQL, or any other external dependency as a sub-chart or bundled deployment.

#### Scenario: Chart contains only AOT
- **WHEN** the chart is rendered with default values
- **THEN** only AOT components (controller, worker, apiserver, web, CRD, RBAC) are produced
