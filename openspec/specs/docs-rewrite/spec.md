# docs-rewrite Specification

## Purpose
TBD - created by archiving change docs-rebrand-uncworks. Update Purpose after archive.
## Requirements
### Requirement: product branding
The system SHALL use "UNCWORKS" as the product name and "agentic development environment" as the tagline in all user-facing documentation.

#### Scenario: README title
- **WHEN** a user views the repository README
- **THEN** it SHALL display "UNCWORKS" as the title and "An agentic development environment" as the subtitle

#### Scenario: web UI title
- **WHEN** a user opens the web dashboard
- **THEN** the browser tab SHALL display "UNCWORKS" as the page title

### Requirement: accurate architecture documentation
The system SHALL provide architecture documentation that accurately reflects the current system components and data flow.

#### Scenario: architecture overview
- **WHEN** a user reads docs/architecture/overview.md
- **THEN** it SHALL describe the control plane (API server, controller, Temporal worker), agent pods (pi + sidecar), and infrastructure (LiteLLM, Ollama, Temporal)

#### Scenario: pipeline documentation
- **WHEN** a user reads docs/architecture/pipeline.md
- **THEN** it SHALL describe the Plan → Execute → Verify flow with agent-manage and agent-implement roles

#### Scenario: workspace documentation
- **WHEN** a user reads docs/architecture/workspace.md
- **THEN** it SHALL describe the workspace layout with repos as worktrees at `/workspace/<repo>/` and OpenSpec at `/workspace/openspec/`

### Requirement: structured documentation
The system SHALL organize docs into architecture/, guides/, reference/, and contributing/ sections.

#### Scenario: doc structure
- **WHEN** a user browses the docs/ directory
- **THEN** they SHALL find subdirectories for architecture, guides, reference, and contributing

