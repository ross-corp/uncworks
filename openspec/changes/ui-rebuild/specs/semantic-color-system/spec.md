## ADDED Requirements

### Requirement: Semantic color tokens are defined for five status intents
The system SHALL define CSS custom property color tokens for five semantic intents: success, active, warning, error, and neutral. Each token SHALL have a foreground color and a muted/background variant. These tokens SHALL be the only colors used for status communication anywhere in the UI.

#### Scenario: Success token is green
- **WHEN** the semantic color tokens are loaded
- **THEN** `--color-success` SHALL be a green value suitable for text/icons on both dark and light backgrounds
- **AND** `--color-success-muted` SHALL be a low-opacity green suitable for backgrounds

#### Scenario: Active token is blue
- **WHEN** the semantic color tokens are loaded
- **THEN** `--color-active` SHALL be a blue value indicating in-progress/running state
- **AND** `--color-active-muted` SHALL be a low-opacity blue suitable for backgrounds

#### Scenario: Warning token is amber
- **WHEN** the semantic color tokens are loaded
- **THEN** `--color-warning` SHALL be an amber value indicating pending/attention state
- **AND** `--color-warning-muted` SHALL be a low-opacity amber suitable for backgrounds

#### Scenario: Error token is red
- **WHEN** the semantic color tokens are loaded
- **THEN** `--color-error` SHALL be a red value indicating failure/broken state
- **AND** `--color-error-muted` SHALL be a low-opacity red suitable for backgrounds

#### Scenario: Neutral token is gray
- **WHEN** the semantic color tokens are loaded
- **THEN** `--color-neutral` SHALL be a gray value indicating cancelled/idle state
- **AND** `--color-neutral-muted` SHALL be a low-opacity gray suitable for backgrounds

### Requirement: Colors carry meaning, never decoration
Every use of a semantic color token in the UI SHALL communicate state or intent. No semantic color token SHALL be used purely for visual decoration or aesthetics. Accent colors (for interactive elements like buttons and links) are separate from semantic status colors.

#### Scenario: No decorative use of status colors
- **WHEN** the codebase is inspected
- **THEN** every reference to `--color-success`, `--color-active`, `--color-warning`, `--color-error`, or `--color-neutral` SHALL be associated with a status-communicating element (status dot, status badge, status text, status border)

### Requirement: Status dots and badges use semantic color tokens exclusively
All status indicators in the UI — dots, badges, text labels, borders — SHALL derive their color from the semantic color tokens. No status indicator SHALL use a hardcoded color value or a non-semantic token.

#### Scenario: StatusBadge uses semantic tokens
- **WHEN** a StatusBadge component renders a status
- **THEN** the badge background SHALL use the muted variant of the corresponding semantic token
- **AND** the badge text color SHALL use the foreground variant of the corresponding semantic token

#### Scenario: Status dots use semantic tokens
- **WHEN** a status dot renders in a RunCard
- **THEN** the dot fill/background color SHALL use the corresponding semantic token directly (not a hardcoded hex value)

### Requirement: Semantic intent is preserved across dark and light mode
The semantic color tokens SHALL convey the same meaning in both dark and light mode. The actual color values MAY differ between modes to maintain contrast and readability, but the mapping of intent to hue SHALL remain constant (success is always green, error is always red, etc.).

#### Scenario: Token values differ by theme but hue matches
- **WHEN** dark mode is active
- **THEN** `--color-success` SHALL be a green value with sufficient contrast against dark backgrounds
- **WHEN** light mode is active
- **THEN** `--color-success` SHALL be a green value with sufficient contrast against light backgrounds
- **AND** both values SHALL be perceptibly green

#### Scenario: All five tokens are defined in both themes
- **WHEN** the `:root` (light) and `.dark` (dark) CSS scopes are inspected
- **THEN** both scopes SHALL define all ten semantic tokens (five foreground, five muted)

### Requirement: All existing status badges are migrated to semantic colors
Every existing StatusBadge instance and any other status-communicating element in the codebase SHALL be updated to use the semantic color tokens instead of the legacy mono-amber color system.

#### Scenario: No legacy status colors remain
- **WHEN** the built CSS and component source are inspected
- **THEN** there SHALL be zero status-communicating elements using the old amber-only color for status differentiation
- **AND** every status element SHALL use one of the five semantic tokens based on its status value

#### Scenario: StatusBadge maps phases to semantic tokens
- **WHEN** StatusBadge renders phase "Succeeded" **THEN** it SHALL use `--color-success`
- **WHEN** StatusBadge renders phase "Running" **THEN** it SHALL use `--color-active`
- **WHEN** StatusBadge renders phase "Pending" **THEN** it SHALL use `--color-warning`
- **WHEN** StatusBadge renders phase "Failed" **THEN** it SHALL use `--color-error`
- **WHEN** StatusBadge renders phase "Cancelled" **THEN** it SHALL use `--color-neutral`
