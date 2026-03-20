## ADDED Requirements

### Requirement: doc staleness detection
The system SHALL detect when documentation references code identifiers that no longer exist.

#### Scenario: stale reference detection
- **WHEN** a PR modifies Go or TypeScript source files
- **THEN** the CI check SHALL scan docs for backtick-quoted identifiers and report any that cannot be found in the codebase

#### Scenario: staleness threshold
- **WHEN** more than 5 stale references are found
- **THEN** the CI check SHALL fail with a list of stale references and their locations
