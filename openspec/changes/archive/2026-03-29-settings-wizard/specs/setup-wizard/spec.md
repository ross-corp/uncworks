## ADDED Requirements

### Requirement: App bootstraps config directory on first launch
On first launch the system SHALL create `~/.config/uncworks/` if it does not exist, and write a default `config.json` with zero-value fields.

#### Scenario: First launch with no config
- **WHEN** the app launches and `~/.config/uncworks/config.json` does not exist
- **THEN** the directory and default config file are created and the setup wizard modal is shown

#### Scenario: Subsequent launch with config present
- **WHEN** the app launches and `~/.config/uncworks/config.json` exists with at least one completed wizard step
- **THEN** the setup wizard is NOT shown automatically

### Requirement: Setup wizard runs as a modal overlay
The system SHALL present the setup wizard as a full-screen modal overlay. The wizard SHALL be triggerable from the Settings page at any time.

#### Scenario: Re-run wizard from settings
- **WHEN** the user clicks "Re-run setup wizard" in Settings
- **THEN** the wizard modal opens with current values pre-filled

#### Scenario: Wizard is dismissible mid-flow
- **WHEN** the user clicks the close/skip button during any wizard step
- **THEN** the modal is dismissed and completed steps are saved; incomplete steps remain in default state

### Requirement: Wizard has three sequential steps
The wizard SHALL have exactly three steps in order: (1) Cluster, (2) GitHub, (3) LiteLLM. Each step SHALL show progress (e.g., "Step 2 of 3"). Completed steps SHALL be visually marked.

#### Scenario: Step progression
- **WHEN** the user completes a step and clicks "Continue"
- **THEN** the next step is shown and the previous step is marked complete

#### Scenario: Step navigation backwards
- **WHEN** the user clicks "Back" on any step after the first
- **THEN** the previous step is shown with its previously entered values preserved
