## ADDED Requirements

### Requirement: Quick start guide
The documentation SHALL include a quick start guide that gets a user from zero to a running AOT instance in under 5 minutes (assuming prerequisites are met).

#### Scenario: User follows quick start
- **WHEN** a user with a Kubernetes cluster and Temporal follows the quick start
- **THEN** they have a running AOT instance with the web dashboard accessible

### Requirement: Prerequisites documentation
The documentation SHALL list all prerequisites: Kubernetes cluster, Temporal server, Helm 3, and optional LLM endpoint.

#### Scenario: User checks prerequisites
- **WHEN** a user reads the prerequisites section
- **THEN** they know exactly what they need before installing

### Requirement: Configuration reference
The documentation SHALL include a complete reference of all `values.yaml` parameters with descriptions, types, and defaults.

#### Scenario: User looks up a value
- **WHEN** a user wants to know what `worker.replicas` does
- **THEN** they find it in the configuration reference with description and default value

### Requirement: Architecture overview
The documentation SHALL include an architecture diagram showing how AOT components interact with each other and external dependencies.

#### Scenario: User understands the system
- **WHEN** a user reads the architecture section
- **THEN** they understand the relationship between controller, worker, Temporal, API server, web dashboard, and agent pods

### Requirement: Upgrade instructions
The documentation SHALL explain how to upgrade AOT to a new version, including CRD upgrade caveats.

#### Scenario: User upgrades
- **WHEN** a user runs `helm upgrade` with a new chart version
- **THEN** they know to also apply CRD updates manually if the schema changed
