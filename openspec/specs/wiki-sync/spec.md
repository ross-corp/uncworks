# wiki-sync Specification

## Purpose
TBD - created by archiving change docs-rebrand-uncworks. Update Purpose after archive.
## Requirements
### Requirement: automatic wiki synchronization
The system SHALL automatically sync docs/ to the GitHub wiki when changes are pushed to main.

#### Scenario: docs change triggers sync
- **WHEN** a push to main modifies files under `docs/`
- **THEN** the GitHub Actions workflow SHALL copy those files to the wiki repo and push

#### Scenario: wiki sidebar generation
- **WHEN** the wiki sync runs
- **THEN** it SHALL generate a `_Sidebar.md` from the docs directory structure with proper navigation links

#### Scenario: wiki home page
- **WHEN** a user visits the GitHub wiki
- **THEN** the Home page SHALL display the content from `docs/README.md`

