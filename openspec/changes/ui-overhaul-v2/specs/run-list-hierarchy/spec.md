## ADDED Requirements

### Requirement: Feature group header expand/collapse affordance
Feature group headers in RunListView SHALL have a clearly visible chevron and full-row hover state.

#### Scenario: Chevron is visually prominent
- **WHEN** a feature group header is rendered
- **THEN** the chevron is text-sm or larger, foreground color (not muted)
- **AND** rotates 90° when expanded

#### Scenario: Full row is hoverable
- **WHEN** the user hovers over a feature group header
- **THEN** the entire row shows a background highlight indicating interactivity
