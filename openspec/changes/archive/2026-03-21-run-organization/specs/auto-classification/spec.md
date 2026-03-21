## ADDED Requirements

### Requirement: LLM classification at run creation
The system SHALL call a cheap LLM model to classify runs at creation time, suggesting feature name, project, and tags.

#### Scenario: New feature detected
- **WHEN** a run is created with prompt "Add factory-droid backend support"
- **AND** no existing feature matches
- **THEN** the classifier SHALL suggest feature name "factory-droid-backend" and featureIsNew=true

#### Scenario: Existing feature matched
- **WHEN** a run is created with prompt "Fix the factory-droid test failures"
- **AND** a feature "factory-droid-backend" exists with a prior failed run
- **THEN** the classifier SHALL suggest feature "factory-droid-backend" and featureIsNew=false

#### Scenario: Project suggestion from repo
- **WHEN** a run targets repo "github.com/roshbhatia/neph.nvim"
- **AND** a project "neph-nvim" exists with prior runs targeting the same repo
- **THEN** the classifier SHALL suggest project "neph-nvim"

### Requirement: Classification is non-blocking
The system SHALL pre-fill classification suggestions in the NewRunView without blocking run creation.

#### Scenario: User accepts defaults
- **WHEN** the classifier suggests feature "factory-droid-backend" and project "neph-nvim"
- **AND** the user submits without changing them
- **THEN** the run is created with those labels

#### Scenario: User overrides suggestion
- **WHEN** the classifier suggests feature "factory-droid-backend"
- **AND** the user changes it to "droid-integration"
- **THEN** the run is created with feature "droid-integration"

### Requirement: Deterministic auto-assignment
The system SHALL deterministically assign repo label from the run's first repo URL without an LLM call.

#### Scenario: Repo label extracted
- **WHEN** a run has repos[0].url = "https://github.com/roshbhatia/neph.nvim"
- **THEN** the label `aot.uncworks.io/repo` SHALL be set to "neph.nvim"
