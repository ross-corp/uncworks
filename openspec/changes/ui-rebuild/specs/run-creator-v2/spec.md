## ADDED Requirements

### Requirement: Radix Form-based run creation
The run creator SHALL be built using Radix Form components styled with the MU-TH-UR theme. It SHALL open as a modal overlay triggered by a "New Run" button in the navigator header. The form SHALL include: spec selector (dropdown of existing specs or "new spec" option), repos section (URL required, branch and path optional, add/remove multiple), prompt textarea (required), backend selector (Pod/KubeVirt/External), and a collapsible "Advanced" section with devboxConfig, ttlSeconds, envVars (key-value pairs), and image fields.

#### Scenario: Open run creator
- **WHEN** user clicks the "New Run" button in the navigator header
- **THEN** a modal overlay appears with the run creation form
- **AND** the form is styled with MU-TH-UR theme (dark background, phosphor green inputs, amber accent buttons)

#### Scenario: Create run with spec association
- **WHEN** user selects an existing spec, fills in repos and prompt, and submits
- **THEN** the run is created via the API with the selected spec association
- **AND** the modal closes and the new run appears in the navigator under that spec

#### Scenario: Create run with new spec
- **WHEN** user selects "New Spec" and provides a spec name
- **THEN** a new spec is created and the run is associated with it

#### Scenario: Multi-repo configuration
- **WHEN** user clicks "Add Repository" multiple times
- **THEN** additional repo fields appear, each with URL, branch, and path inputs
- **AND** each repo can be individually removed with a delete button

#### Scenario: Validation prevents empty submission
- **WHEN** user submits without required fields (at least one repo URL, prompt)
- **THEN** validation errors appear on the empty fields with red glow styling
- **AND** the form does not submit

#### Scenario: Advanced fields are collapsible
- **WHEN** user clicks the "Advanced" section header
- **THEN** the section expands to show devboxConfig, ttlSeconds, envVars, and image fields
- **AND** collapsing hides them without losing entered values

### Requirement: Template presets
The run creator SHALL support template presets that pre-fill form fields. A "Load Preset" dropdown SHALL list saved presets. Users SHALL be able to save the current form state as a new preset.

#### Scenario: Load a preset
- **WHEN** user selects a preset from the dropdown
- **THEN** all form fields are populated with the preset's values
- **AND** the user can modify any field before submitting

#### Scenario: Save current form as preset
- **WHEN** user clicks "Save as Preset" and provides a name
- **THEN** the current form values are saved to localStorage as a named preset
- **AND** the preset appears in the Load Preset dropdown
