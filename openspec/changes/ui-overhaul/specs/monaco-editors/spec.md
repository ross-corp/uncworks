## ADDED Requirements

### Requirement: Prompt editor uses Monaco
The prompt textarea in the New Run view SHALL be replaced with a Monaco editor instance with markdown syntax highlighting.

#### Scenario: Monaco renders for prompt input
- **WHEN** a user opens the New Run view in prompt mode
- **THEN** the prompt input SHALL be a Monaco editor with markdown language mode

#### Scenario: Monaco follows site theme
- **WHEN** the site theme is light
- **THEN** the Monaco editor SHALL use a light theme (e.g., "vs") AND when dark, SHALL use "vs-dark"

### Requirement: Spec editor uses Monaco
The spec textarea in the New Run view SHALL be replaced with a Monaco editor instance with markdown syntax highlighting.

#### Scenario: Monaco renders for spec input
- **WHEN** a user switches to Spec mode in the New Run view
- **THEN** the spec input SHALL be a Monaco editor with markdown language mode and monospace font

### Requirement: File preview Monaco is theme-aware
The existing Monaco editor in the file preview component SHALL follow the site dark/light theme.

#### Scenario: File preview respects light theme
- **WHEN** the site is in light mode
- **THEN** the file preview Monaco editor SHALL use the "vs" theme instead of hardcoded "vs-dark"
