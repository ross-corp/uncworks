# ui-theming Specification

## Purpose
TBD - created by archiving change ui-rewrite. Update Purpose after archive.
## Requirements
### Requirement: Support all shadcn built-in themes
The UI SHALL support all shadcn built-in color themes (zinc, slate, stone, gray, neutral, red, rose, orange, green, blue, yellow, violet) selectable via the command palette or settings.

#### Scenario: Theme selection via command palette
- **WHEN** the user opens the command palette and types "theme"
- **THEN** a list of available themes is shown
- **AND** selecting one applies the theme immediately

### Requirement: Light/dark mode toggle
The UI SHALL support light and dark color modes, toggleable via keyboard shortcut or command palette.

#### Scenario: Toggle dark mode
- **WHEN** the user presses the dark mode toggle shortcut
- **THEN** the UI switches between light and dark mode

#### Scenario: Default to system preference
- **WHEN** no theme preference is saved
- **THEN** the UI uses the system color scheme preference (prefers-color-scheme)
- **AND** defaults to dark if no system preference is detected

### Requirement: Theme preference persisted to localStorage
The selected theme and color mode SHALL be saved to localStorage and restored on page load.

#### Scenario: Preference persists across reloads
- **WHEN** the user selects the "blue" theme in dark mode and reloads the page
- **THEN** the "blue" theme in dark mode is applied on load without flash

#### Scenario: Anti-flash on load
- **WHEN** the page loads with a saved dark mode preference
- **THEN** the dark mode class is applied before first paint (no white flash)

