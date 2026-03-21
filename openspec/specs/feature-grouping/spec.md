# feature-grouping Specification

## Purpose
TBD - created by archiving change run-organization. Update Purpose after archive.
## Requirements
### Requirement: Feature as a run grouping
The system SHALL group runs by feature label, where a feature represents a unit of deliverable value with one or more run attempts.

#### Scenario: Multiple runs in same feature
- **WHEN** two runs have label `aot.uncworks.io/feature=factory-droid`
- **THEN** they are grouped together as attempts of the same feature
- **AND** the feature shows aggregate status (best result, total attempts)

#### Scenario: Feature links to OpenSpec change
- **WHEN** a spec-driven run creates an OpenSpec change named "ar-xyz123"
- **AND** the run has feature label "factory-droid"
- **THEN** the feature detail view shows the OpenSpec change artifacts (proposal, specs, tasks)

### Requirement: Feature lifecycle tracking
The system SHALL track feature status as an aggregate of its runs and PR state.

#### Scenario: Feature status derived from runs
- **WHEN** a feature has runs with phases [FAILED, FAILED, SUCCEEDED]
- **THEN** the feature status SHALL be "DONE" (latest successful attempt)

#### Scenario: Feature with open PR
- **WHEN** a feature's latest successful run has a non-empty prUrl
- **THEN** the feature status SHALL be "IN REVIEW"

### Requirement: Retry creates linked run
The system SHALL support retrying a feature by creating a new run with the same prompt and the same feature label.

#### Scenario: Retry from feature view
- **WHEN** the user triggers retry on feature "factory-droid"
- **THEN** a new run is created with the same prompt, repos, and feature label
- **AND** the previous failure context is included in the prompt

