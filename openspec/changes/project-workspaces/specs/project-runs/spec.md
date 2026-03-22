## ADDED Requirements

### Requirement: AgentRun Project Reference
The AgentRun CRD SHALL include an optional `projectRef` field that references a Project resource by name. The AgentRun CRD SHALL also include an optional `specRef` field that identifies a specification file path within the project's config repository.

#### Scenario: AgentRun created with project and spec references
- **WHEN** an AgentRun is created with `projectRef: my-project` and `specRef: openspec/feature-x/spec.md`
- **THEN** the AgentRun resource SHALL store both references and they SHALL be queryable via the API

### Requirement: Project-Inherited Configuration
When an AgentRun has a `projectRef` set, the run SHALL inherit the project's `repos`, default `model`, and `devboxPackages` unless explicitly overridden in the AgentRun spec. Project defaults SHALL be resolved at run creation time.

#### Scenario: Run inherits project repos and model
- **WHEN** an AgentRun is created with `projectRef` pointing to a project that has `repos: ["https://github.com/org/app"]` and `defaults.model: "claude-sonnet"`
- **THEN** the run SHALL clone `https://github.com/org/app` and use `claude-sonnet` as its model

#### Scenario: Run-level override takes precedence
- **WHEN** an AgentRun is created with `projectRef` pointing to a project with `defaults.model: "claude-sonnet"` but the AgentRun itself specifies `model: "claude-opus"`
- **THEN** the run SHALL use `claude-opus`

### Requirement: Spec Content Fetching
When `specRef` is set on an AgentRun, the run controller SHALL fetch the spec file content from the project's soft-serve config repository and provide it to the agent as the task prompt.

#### Scenario: Spec content loaded from config repo
- **WHEN** an AgentRun starts with `specRef: openspec/auth/spec.md`
- **THEN** the controller SHALL clone the project's config repo, read the file at `openspec/auth/spec.md`, and supply its content as the agent's task input

### Requirement: Standalone Run Compatibility
AgentRuns without a `projectRef` SHALL continue to function exactly as before this change. The `projectRef` and `specRef` fields SHALL be optional with no default values.

#### Scenario: Run without projectRef works unchanged
- **WHEN** an AgentRun is created without a `projectRef` or `specRef`
- **THEN** the run SHALL execute using only its own inline configuration with no project lookup performed
