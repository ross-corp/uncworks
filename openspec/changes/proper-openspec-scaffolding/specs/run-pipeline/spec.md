## MODIFIED Requirements

### Requirement: Plan stage scaffolds OpenSpec change via CLI before agent starts
The PlanRun activity SHALL create the OpenSpec change directory and retrieve artifact templates via CLI commands before starting the planning agent.

#### Scenario: Change scaffolded before agent
- **WHEN** PlanRun starts
- **THEN** `openspec new change "<run-id>"` is executed
- **AND** `openspec instructions` is called for each artifact to get templates
- **AND** the agent prompt includes the exact file paths and format templates

#### Scenario: Agent writes to scaffolded paths
- **WHEN** the planning agent receives its prompt
- **THEN** the prompt includes exact file paths like `openspec/changes/<id>/proposal.md`
- **AND** the prompt includes the template format for each artifact
- **AND** the agent writes content to those exact paths

#### Scenario: Scaffolding failure prevents agent start
- **WHEN** `openspec new change` fails
- **THEN** the planning stage fails with a clear error
- **AND** the agent is NOT started
