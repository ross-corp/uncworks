## ADDED Requirements

### Requirement: Run list displays total cost
The system SHALL display an estimated total cost for completed runs in the run list.

#### Scenario: Cost column for completed run
- **WHEN** a run has completed (succeeded or failed) and has trace spans with cost metadata
- **THEN** the run list SHALL display the aggregated cost (e.g., "$0.12") in a cost column

#### Scenario: Cost not available for running jobs
- **WHEN** a run is still in progress
- **THEN** the cost column SHALL display "—" or be empty

### Requirement: Run list displays diff stats
The system SHALL display +/- lines changed for runs that have diff metadata.

#### Scenario: Diff stats for run with changes
- **WHEN** a run has trace spans with diff.additions and diff.deletions metadata
- **THEN** the run list SHALL display aggregated diff stats (e.g., "+42/-5") with green/red coloring

### Requirement: Run list displays PR badge
The system SHALL display a clickable PR badge for runs that created a pull request.

#### Scenario: PR badge with link
- **WHEN** a run has a `prUrl` in its status
- **THEN** the run list SHALL display a clickable badge linking to the PR

#### Scenario: PR badge shows target repo
- **WHEN** a user hovers over or clicks the PR badge
- **THEN** the badge SHALL indicate the target repository

### Requirement: Run list displays dual model info
The system SHALL display which models are used for manage and implement roles.

#### Scenario: Single model display
- **WHEN** a run uses the same model for manage and implement
- **THEN** the model column SHALL display the single model name

#### Scenario: Dual model display
- **WHEN** a run uses different models for manage and implement
- **THEN** the model column SHALL display both (e.g., "qwen3:8b / deepseek-v3.1")

### Requirement: Feature group badges are inline
Feature group headers SHALL display status badges inline with the feature name.

#### Scenario: Inline status badge
- **WHEN** a feature group is rendered in the run list
- **THEN** the status badge SHALL appear immediately after the feature name, not in a separate column
