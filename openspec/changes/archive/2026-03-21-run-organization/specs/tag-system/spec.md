## ADDED Requirements

### Requirement: Freeform tags on runs
The system SHALL support multiple freeform tags per run, stored as a comma-separated label value.

#### Scenario: Tags set at creation
- **WHEN** a run is created with tags ["feature", "backend", "lua"]
- **THEN** label `aot.uncworks.io/tags` SHALL be set to "backend,feature,lua" (sorted)

#### Scenario: Filter by tag
- **WHEN** the user filters the run list with `/tag:backend`
- **THEN** only runs whose tags label contains "backend" are shown

### Requirement: Post-run tag enrichment
The system SHALL asynchronously enrich run tags after completion based on git diff analysis.

#### Scenario: File type tags from diff
- **WHEN** a run completes and its git diff touches .lua and .md files
- **THEN** tags "lua" and "docs" SHALL be appended to the run's tags

#### Scenario: Scope tag from diff size
- **WHEN** a run's git diff changes fewer than 5 files
- **THEN** tag "small-change" SHALL be appended
