# model-selection-ui Specification

## Purpose
TBD - created by archiving change newrun-model-selection. Update Purpose after archive.
## Requirements
### Requirement: Model tier selection in new run form

The NewRunView form SHALL include a model tier dropdown that allows users to select from all available model tiers before creating an agent run.

#### Scenario: Default tier is pre-selected
- **WHEN** the new run form loads
- **THEN** the model tier dropdown shows "default" as the selected value

#### Scenario: User selects a different tier
- **WHEN** the user opens the model tier dropdown and selects "budget"
- **THEN** the dropdown displays "budget" as the selected value
- **AND** the CreateAgentRun request includes `modelTier: "budget"`

#### Scenario: All tiers are available
- **WHEN** the user opens the model tier dropdown
- **THEN** options include: budget, default, balanced, performance, max, custom, router

#### Scenario: Cost hints displayed
- **WHEN** the user views the model tier options
- **THEN** each option shows a brief cost/quality hint (e.g., "Budget - lowest cost")

### Requirement: Selected tier sent in create request

The CreateAgentRun request payload SHALL include the user-selected model tier value from the dropdown.

#### Scenario: Tier included in request
- **GIVEN** the user selected "performance" as the model tier
- **WHEN** the user submits the form
- **THEN** the API call includes `modelTier: "performance"` in the request body

