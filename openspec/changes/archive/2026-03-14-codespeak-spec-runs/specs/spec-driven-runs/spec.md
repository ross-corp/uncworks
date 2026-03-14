## ADDED Requirements

### Requirement: Spec content on AgentRunSpec
The proto `AgentRunSpec` SHALL include `string spec_content` and `string spec_source` fields. The CRD `AgentRunSpec` SHALL include corresponding `SpecContent` and `SpecSource` string fields.

#### Scenario: Spec-driven run creation
- **WHEN** a run is created with `spec_content: "# MyConverter\n..."` and `spec_source: "editor"`
- **THEN** the AgentRun CRD stores both fields and they are retrievable via GetAgentRun

#### Scenario: Prompt-driven run (backward compatible)
- **WHEN** a run is created with empty `spec_content`
- **THEN** the run behaves exactly as before with no spec-related behavior

### Requirement: Spec file written to workspace
When an agent run has non-empty `spec_content`, the hydrator SHALL write the spec to `/workspace/spec/main.cs.md` and generate a `/workspace/codespeak.json` referencing it.

#### Scenario: Spec placed in workspace
- **WHEN** hydration runs for a spec-driven agent run with `spec_content: "# EmlConverter\n..."`
- **THEN** `/workspace/spec/main.cs.md` contains the spec content
- **AND** `/workspace/codespeak.json` contains `{"specs": ["spec/main.cs.md"]}`

#### Scenario: Spec with existing repos
- **WHEN** a spec-driven run also has repos cloned to `/workspace/src/`
- **THEN** the spec file and `codespeak.json` are placed at `/workspace/` alongside the `src/` directory
- **AND** `codespeak.json` includes whitelisted file paths if the repos contain files that the spec should modify

### Requirement: Auto-generated prompt for spec runs
When a run has `spec_content`, the system SHALL auto-generate a prompt instructing the agent to run `codespeak build` if the prompt field is empty.

#### Scenario: No explicit prompt with spec
- **WHEN** a run is created with `spec_content` set and `prompt` empty
- **THEN** the system sets the prompt to an instruction to run `codespeak build` in the workspace

#### Scenario: Explicit prompt with spec
- **WHEN** a run is created with both `spec_content` and `prompt` set
- **THEN** the explicit prompt is used as-is (the spec file is still written to the workspace)

### Requirement: Spec content passed through API pipeline
The `spec_content` and `spec_source` fields SHALL flow through the full pipeline: proto → gRPC handler → CRD → controller → workflow → hydrator.

#### Scenario: Round-trip through API
- **WHEN** `CreateAgentRun` is called with `spec_content: "# Test"`
- **THEN** `GetAgentRun` returns the same `spec_content: "# Test"` and `ListAgentRuns` includes it

### Requirement: CodeSpeak available in agent environment
The agent's devbox environment SHALL include the `codespeak` CLI so that `codespeak build` can be executed.

#### Scenario: codespeak CLI available
- **WHEN** an agent starts in a spec-driven run
- **THEN** `codespeak --version` executes successfully in the agent's shell
