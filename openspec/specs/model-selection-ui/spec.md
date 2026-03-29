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

### Requirement: Unified project field in NewRunView
NewRunView SHALL present a single project field that unifies the projectRef CRD dropdown and classification project label.

#### Scenario: CRD projects offered as primary options
- **WHEN** CRD projects exist
- **THEN** the project field shows them as dropdown options
- **AND** selecting a CRD project sets projectRef and auto-fills repos/model/orchestration mode

#### Scenario: Custom label option for non-CRD projects
- **WHEN** the user selects "Custom label..." from the project dropdown
- **THEN** a text input appears for entering a free-form project label
- **AND** this sets only the classification project field, not projectRef

#### Scenario: Ctrl+Enter submits the form
- **WHEN** the user presses Ctrl+Enter (or Cmd+Enter on Mac) while NewRunView is focused
- **THEN** the form is submitted if canRun is true
- **AND** a keyboard shortcut hint shows near the Run button: "⌘↵"

### Requirement: Visible "Improve with AI" button
The "Improve with AI" button SHALL be visually prominent and actionable.

#### Scenario: Button is visible at default zoom
- **WHEN** the prompt editor is shown
- **THEN** the "Improve with AI" button is at minimum 28px tall with an icon (✨ or sparkle)
- **AND** uses variant="outline" or "secondary", not ghost

#### Scenario: Error toast on improvement failure
- **WHEN** the improve API call fails
- **THEN** a toast shows "Couldn't improve prompt — try again"
- **AND** the button returns to its default state

### Requirement: Model tier descriptions communicate trade-offs
Model tier options SHALL describe decision criteria, not implementation details.

#### Scenario: Model tier shows trade-off description
- **WHEN** the user opens the model tier selector
- **THEN** each option shows a human-readable trade-off label (e.g., "Fast & cheap", "Best quality", "Local / offline")
- **AND** the raw model identifier is shown as secondary text only

