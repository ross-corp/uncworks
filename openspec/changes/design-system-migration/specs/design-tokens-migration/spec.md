## ADDED Requirements

### Requirement: MU-TH-UR 6000 color tokens replace all AOT custom color tokens
The system SHALL use the MU-TH-UR 6000 color token system defined via CSS custom properties with oklch color values. All references to the legacy AOT color tokens (--surface-0, --surface-1, --surface-2, --surface-3, --text-primary, --text-secondary, --text-tertiary, --border, --border-strong, --accent, --accent-hover, --danger, --warning, --success, --info) SHALL be removed and replaced with MU-TH-UR equivalents (--background, --foreground, --primary, --primary-foreground, --secondary, --secondary-foreground, --card, --card-foreground, --muted, --muted-foreground, --accent, --accent-foreground, --destructive, --destructive-foreground, --border, --input, --ring).

#### Scenario: Background color uses pure black
- **WHEN** the application renders the root body element
- **THEN** the background color SHALL be oklch(0 0 0) (pure black), not #09090b

#### Scenario: Primary text uses amber phosphor color
- **WHEN** any primary text content is rendered
- **THEN** the text color SHALL be oklch(0.85 0.15 75) (amber phosphor), not #fafafa

#### Scenario: Secondary accent uses Nostromo green
- **WHEN** secondary-colored elements are rendered (success states, secondary buttons, secondary data)
- **THEN** the color SHALL be oklch(0.80 0.18 145) (Nostromo green), not the legacy success/info colors

#### Scenario: No legacy AOT color tokens remain in CSS
- **WHEN** the built CSS output is inspected
- **THEN** there SHALL be zero references to --surface-0, --surface-1, --surface-2, --surface-3, --text-primary, --text-secondary, --text-tertiary, or --accent-hover

### Requirement: IoskeleyMono is the sole typeface
The system SHALL use IoskeleyMono as the only font family for both the `font-sans` and `font-mono` Tailwind utilities. The Inter and JetBrains Mono font imports SHALL be removed. The Google Fonts import for Inter and JetBrains Mono SHALL be removed from index.css.

#### Scenario: Body text renders in IoskeleyMono
- **WHEN** the application body text is rendered
- **THEN** the computed font-family SHALL be 'IoskeleyMono', monospace

#### Scenario: Code/mono text renders in IoskeleyMono
- **WHEN** monospace text is rendered (code blocks, terminal output, editor content)
- **THEN** the computed font-family SHALL be 'IoskeleyMono', monospace

#### Scenario: No Inter or JetBrains Mono references remain
- **WHEN** the CSS source files are inspected
- **THEN** there SHALL be zero references to 'Inter', 'JetBrains Mono', or 'Fira Code'

### Requirement: Font files are self-hosted
The system SHALL load IoskeleyMono font files from the local `public/fonts/` directory using `@font-face` declarations. The system SHALL NOT depend on any external CDN or Google Fonts for font delivery.

#### Scenario: Font files exist in public directory
- **WHEN** the project file structure is inspected
- **THEN** IoskeleyMono font files (.woff2 and/or .woff) SHALL exist in `web/public/fonts/`

#### Scenario: Font loads without network dependency
- **WHEN** the application is loaded with no external network access
- **THEN** IoskeleyMono SHALL render correctly

### Requirement: All border-radius values are zero
The system SHALL set all border-radius tokens (--radius, --radius-sm, --radius-md, --radius-lg, --radius-xl) to 0px. No element in the UI SHALL have visible border-radius. The `rounded` utility class references in components SHALL be removed or overridden to 0px.

#### Scenario: Buttons have no border-radius
- **WHEN** a Button component is rendered
- **THEN** the computed border-radius SHALL be 0px

#### Scenario: Cards have no border-radius
- **WHEN** a Card component is rendered
- **THEN** the computed border-radius SHALL be 0px

#### Scenario: Input fields have no border-radius
- **WHEN** an Input component is rendered
- **THEN** the computed border-radius SHALL be 0px

### Requirement: Body text is uppercase with wide letter-spacing
The system SHALL apply `uppercase` text-transform and `tracking-widest` letter-spacing to the body element, matching the MU-TH-UR 6000 terminal aesthetic.

#### Scenario: Body text is uppercase
- **WHEN** the body element styles are inspected
- **THEN** text-transform SHALL be `uppercase`

#### Scenario: Body text has wide tracking
- **WHEN** the body element styles are inspected
- **THEN** letter-spacing SHALL match Tailwind's `tracking-widest` value

### Requirement: Spacing tokens use 8px base unit
The system SHALL use the MU-TH-UR spacing scale with an 8px base unit: xs(4px), sm(8px), md(12px), base(16px), lg(24px), xl(32px), 2xl(48px), 3xl(64px).

#### Scenario: Spacing tokens are available as CSS variables
- **WHEN** the design tokens CSS is loaded
- **THEN** spacing custom properties (--spacing-xs through --spacing-3xl) SHALL be defined with the correct values

### Requirement: Glow effect tokens are defined
The system SHALL define CSS custom properties for glow effects: --glow-primary (amber at 50% opacity) and --glow-secondary (green at 50% opacity). These tokens SHALL be consumed by the fx-glow and fx-box-glow utility classes.

#### Scenario: Glow tokens are set in :root
- **WHEN** the root CSS custom properties are inspected
- **THEN** --glow-primary SHALL be oklch(0.85 0.15 75 / 0.5) and --glow-secondary SHALL be oklch(0.80 0.18 145 / 0.5)
