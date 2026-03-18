## ADDED Requirements

### Requirement: display_name field on AgentRunSpec proto
The `AgentRunSpec` proto message SHALL include a `display_name` string field. This field holds a human-readable name for the run. It is optional — runs may have an empty display_name.

#### Scenario: Field present in proto
- **WHEN** a client reads the `AgentRunSpec` proto definition
- **THEN** a `string display_name` field exists
- **AND** it is serialized/deserialized correctly in both Go and TypeScript

#### Scenario: Field is optional
- **WHEN** a run is created without a display_name (e.g., LLM fallback)
- **THEN** the field defaults to an empty string
- **AND** no error is raised

### Requirement: display_name field on AgentRun CRD
The AgentRun CRD spec SHALL include a `displayName` field (camelCase per K8s convention). The field is optional and stored as a string.

#### Scenario: CRD includes displayName
- **WHEN** an AgentRun CRD is created with a `displayName` in the spec
- **THEN** the field is persisted in etcd and readable via kubectl or the K8s API

#### Scenario: CRD without displayName
- **WHEN** an AgentRun CRD is created without a `displayName`
- **THEN** the CRD is valid and the field defaults to empty

### Requirement: Shared TypeScript types include displayName
The shared TypeScript types package SHALL include `displayName` as an optional string field on the AgentRunSpec type. The web types SHALL also be updated if separately maintained.

#### Scenario: TypeScript types are in sync
- **WHEN** the web UI imports agent run types
- **THEN** the `displayName` field is available on the spec type
- **AND** TypeScript compilation succeeds

### Requirement: Web UI shows display_name as primary run identifier
The web UI SHALL show `display_name` as the primary name for runs wherever run names appear. When `display_name` is empty, the UI SHALL fall back to showing the K8s resource name.

#### Scenario: Run list shows display name
- **WHEN** a user views the run list
- **AND** a run has `display_name` set to `fix-auth-token-expiry`
- **THEN** the list shows `fix-auth-token-expiry` as the primary text for that run
- **AND** the K8s name (e.g., `ar-a3gfp3`) is shown as secondary/muted text

#### Scenario: Run list falls back to K8s name
- **WHEN** a user views the run list
- **AND** a run has an empty `display_name`
- **THEN** the list shows the K8s name as the primary text
- **AND** no secondary name is shown

#### Scenario: Run detail header shows display name
- **WHEN** a user views a run's detail page
- **AND** the run has a `display_name`
- **THEN** the page title/header shows the display name
- **AND** the K8s name is shown as a subtitle or secondary identifier

#### Scenario: Command palette searches display name
- **WHEN** a user searches in the command palette
- **THEN** the search matches against both `display_name` and K8s name
- **AND** results show `display_name` as the primary label
