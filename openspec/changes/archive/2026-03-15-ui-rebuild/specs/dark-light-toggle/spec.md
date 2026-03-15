## ADDED Requirements

### Requirement: Theme toggle button in the header with sun/moon icon
The application header SHALL contain a toggle button that switches between dark and light mode. The button SHALL display a moon icon when in light mode (indicating "switch to dark") and a sun icon when in dark mode (indicating "switch to light").

#### Scenario: Toggle button is visible in header
- **WHEN** the application renders
- **THEN** a theme toggle button SHALL be visible in the header area
- **AND** the button SHALL be accessible with an aria-label of "Toggle theme"

#### Scenario: Icon reflects current theme
- **WHEN** dark mode is active
- **THEN** the toggle button SHALL display a sun icon
- **WHEN** light mode is active
- **THEN** the toggle button SHALL display a moon icon

#### Scenario: Clicking toggle switches theme
- **WHEN** the user clicks the theme toggle while in dark mode
- **THEN** the application SHALL switch to light mode immediately
- **WHEN** the user clicks the theme toggle while in light mode
- **THEN** the application SHALL switch to dark mode immediately

### Requirement: Dark mode uses black background with light foreground and MU-TH-UR effects
When dark mode is active, the application SHALL render with a black/near-black background, light foreground text, and MU-TH-UR visual effects (scanlines, glow). The IoskeleyMono font SHALL be used.

#### Scenario: Dark mode background and text
- **WHEN** dark mode is active
- **THEN** the root background color SHALL be black or near-black (oklch lightness <= 0.05)
- **AND** the primary text color SHALL be light (oklch lightness >= 0.75)

#### Scenario: MU-TH-UR effects active in dark mode
- **WHEN** dark mode is active
- **THEN** the CRT scanline overlay SHALL be visible
- **AND** glow effects on interactive/status elements SHALL be rendered
- **AND** elements using the `.fx-glow` or `.fx-box-glow` classes SHALL display their glow

### Requirement: Light mode uses white background with dark foreground and MU-TH-UR effects disabled
When light mode is active, the application SHALL render with a white/near-white background, dark foreground text, and all MU-TH-UR visual effects (scanlines, glow) disabled. The IoskeleyMono font SHALL remain active.

#### Scenario: Light mode background and text
- **WHEN** light mode is active
- **THEN** the root background color SHALL be white or near-white (oklch lightness >= 0.95)
- **AND** the primary text color SHALL be dark (oklch lightness <= 0.25)

#### Scenario: MU-TH-UR effects disabled in light mode
- **WHEN** light mode is active
- **THEN** the CRT scanline overlay SHALL NOT be visible (display: none or opacity: 0)
- **AND** glow effects SHALL NOT be rendered
- **AND** elements using glow classes SHALL have no visible box-shadow glow

#### Scenario: IoskeleyMono font in both modes
- **WHEN** light mode is active
- **THEN** the computed font-family SHALL be 'IoskeleyMono', monospace
- **WHEN** dark mode is active
- **THEN** the computed font-family SHALL be 'IoskeleyMono', monospace

### Requirement: Theme preference persists to localStorage
The user's theme choice SHALL be saved to localStorage and restored on subsequent page loads. The key SHALL be consistent and the value SHALL be either "dark" or "light".

#### Scenario: Theme is saved on toggle
- **WHEN** the user toggles the theme
- **THEN** the selected theme ("dark" or "light") SHALL be written to localStorage

#### Scenario: Theme is restored on load
- **WHEN** the application loads and localStorage contains a saved theme preference
- **THEN** the application SHALL apply that theme immediately (before first paint if possible)

### Requirement: System preference detection on first load
When no theme preference exists in localStorage (first visit), the application SHALL detect the user's operating system color scheme preference via `prefers-color-scheme` media query and apply the matching theme.

#### Scenario: System prefers dark
- **WHEN** the application loads for the first time (no localStorage value)
- **AND** the system `prefers-color-scheme` is `dark`
- **THEN** the application SHALL start in dark mode

#### Scenario: System prefers light
- **WHEN** the application loads for the first time (no localStorage value)
- **AND** the system `prefers-color-scheme` is `light`
- **THEN** the application SHALL start in light mode

#### Scenario: Explicit choice overrides system preference
- **WHEN** the user has previously toggled the theme (localStorage has a value)
- **THEN** the stored preference SHALL take priority over the system preference
