## ADDED Requirements

### Requirement: comprehensive platform audit
The system SHALL have a documented audit of every component, identifying dead code, broken paths, stale tests, and areas needing improvement.

#### Scenario: audit produces findings
- **WHEN** the audit tasks are completed
- **THEN** a findings report SHALL exist at `openspec/changes/full-platform-audit/findings.md` with categorized issues

#### Scenario: findings generate proposals
- **WHEN** the findings report identifies significant issues
- **THEN** each issue category SHALL have a corresponding `/opsx:propose` change created
