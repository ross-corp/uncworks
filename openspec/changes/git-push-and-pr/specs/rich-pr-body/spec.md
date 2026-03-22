## ADDED Requirements

### Requirement: PR body includes OpenSpec proposal
The system SHALL read the OpenSpec proposal.md from the workspace and include its content in the GitHub PR body.

#### Scenario: Proposal included in PR body
- **WHEN** a spec-driven run succeeds and creates a PR
- **THEN** the PR body SHALL contain the content of `openspec/changes/{changeName}/proposal.md`

### Requirement: PR body includes diff stats
The system SHALL include the number of files changed, additions, and deletions in the PR body.

#### Scenario: Diff stats in PR body
- **WHEN** a successful run pushes changes and creates a PR
- **THEN** the PR body SHALL include lines like "N files changed, +M additions, -K deletions"

### Requirement: PR body includes run link
The system SHALL include a link to the UNCWORKS run in the PR body.

#### Scenario: Run link in PR body
- **WHEN** a PR is created for run ar-abc123
- **THEN** the PR body SHALL contain the run ID and a note that it was created by the spec-driven pipeline

### Requirement: Push handles existing branch
The system SHALL handle the case where the target branch already exists by using force push or a unique branch name.

#### Scenario: Branch already exists
- **WHEN** PushChanges is called and the branch `aot/{runId}` already exists on the remote
- **THEN** the push SHALL succeed (force push to overwrite the stale branch)
