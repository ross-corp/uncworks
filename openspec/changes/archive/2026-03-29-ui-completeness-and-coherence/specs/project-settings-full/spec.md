## ADDED Requirements

### Requirement: Devbox packages editor
The Project Settings tab SHALL include a devbox packages section that lists current `devbox.packages` strings and allows adding/removing them.

#### Scenario: View packages
- **WHEN** user opens Settings tab on a project with devbox packages
- **THEN** each package string is shown with a remove button

#### Scenario: Add package
- **WHEN** user types in the "add package" input and presses Enter or clicks "+"
- **THEN** package is added to the local list; settingsDirty becomes true

#### Scenario: Remove package
- **WHEN** user clicks remove on a package
- **THEN** package is removed from local list; settingsDirty becomes true

#### Scenario: Save persists packages
- **WHEN** user saves settings
- **THEN** PUT /api/v1/projects/:name is called with updated devbox.packages array

### Requirement: Project run defaults editor
The Project Settings tab SHALL include a "Run Defaults" section covering all ProjectDefaults fields: modelTier, manageModelTier, implementModelTier, ttlSeconds, orchestrationMode, autoPush, autoPR, prBaseBranch.

#### Scenario: Model tier fields
- **WHEN** user opens Settings tab
- **THEN** three model tier selects are shown (main, manage, implement) with options: "", "economy", "standard", "performance"

#### Scenario: TTL field
- **WHEN** user changes ttlSeconds
- **THEN** input accepts integer seconds; settingsDirty becomes true

#### Scenario: Boolean toggles
- **WHEN** user toggles autoPush or autoPR checkboxes
- **THEN** settingsDirty becomes true; value reflected in PUT body on save

#### Scenario: prBaseBranch field
- **WHEN** autoPR is enabled
- **THEN** prBaseBranch text input is enabled; otherwise disabled/greyed

#### Scenario: Save persists defaults
- **WHEN** user saves settings
- **THEN** PUT /api/v1/projects/:name called with updated defaults object
