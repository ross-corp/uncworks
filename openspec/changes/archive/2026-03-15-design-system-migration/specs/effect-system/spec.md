## ADDED Requirements

### Requirement: CRT scanline overlay available as fx-scanlines utility
The system SHALL provide an `fx-scanlines` CSS utility class that renders a repeating linear-gradient pseudo-element overlay simulating CRT scanlines. The overlay SHALL be non-interactive (pointer-events: none), absolutely positioned over the element, use mix-blend-mode: multiply, and render at 50% opacity with a 3px scanline pitch.

#### Scenario: Panel with fx-scanlines displays scanline overlay
- **WHEN** a Card or panel element has the `fx-scanlines` class applied
- **THEN** a ::after pseudo-element SHALL render over the element with repeating horizontal lines simulating CRT scanlines

#### Scenario: Scanline overlay does not block interaction
- **WHEN** a user clicks on interactive content within an fx-scanlines element
- **THEN** the click SHALL pass through the scanline overlay to the underlying element (pointer-events: none)

### Requirement: Phosphor glow effect available as fx-glow utility
The system SHALL provide an `fx-glow` CSS utility class that applies a text-shadow simulating phosphor bloom. The glow SHALL use the --glow-primary token (amber) with two shadow layers: a tight 4px blur and a wider 12px blur.

#### Scenario: Heading with fx-glow displays amber text shadow
- **WHEN** a text element has the `fx-glow` class applied
- **THEN** the text SHALL have a visible amber glow effect via text-shadow using the --glow-primary token

#### Scenario: fx-glow-amber and fx-glow-danger variants exist
- **WHEN** the CSS utilities are inspected
- **THEN** `fx-glow-amber` and `fx-glow-danger` utility classes SHALL be defined with their respective color tokens

### Requirement: Box glow effects available as fx-box-glow utilities
The system SHALL provide `fx-box-glow`, `fx-box-glow-warning`, and `fx-box-glow-danger` CSS utility classes that apply box-shadow effects simulating terminal panel illumination. Each SHALL use a 1px border ring plus a 16px color glow.

#### Scenario: Focused input with fx-box-glow has amber box shadow
- **WHEN** an element has the `fx-box-glow` class applied
- **THEN** the element SHALL have a box-shadow with a 16px amber glow from --glow-primary

#### Scenario: Error state with fx-box-glow-danger has red box shadow
- **WHEN** an element has the `fx-box-glow-danger` class applied
- **THEN** the element SHALL have a box-shadow with a 16px red glow from --glow-danger

### Requirement: CRT flicker animation available as fx-flicker utility
The system SHALL provide an `fx-flicker` CSS utility class that applies a subtle opacity-varying animation on a 6-second infinite loop. The animation SHALL create a realistic CRT phosphor flicker with brief dips to 93-96% opacity at irregular intervals. The animation SHALL only run when `prefers-reduced-motion: no-preference`.

#### Scenario: App wrapper with fx-flicker has subtle opacity animation
- **WHEN** an element has the `fx-flicker` class applied and the user has not enabled reduced motion
- **THEN** the element SHALL animate opacity between 0.93 and 1.0 on a 6-second cycle

#### Scenario: fx-flicker respects reduced motion preference
- **WHEN** the user has enabled `prefers-reduced-motion: reduce`
- **THEN** the fx-flicker animation SHALL NOT run

### Requirement: Glitch effect available as fx-glitch utility
The system SHALL provide an `fx-glitch` CSS utility class that applies a transform-based glitch animation on a 4-second infinite loop. The animation SHALL create brief horizontal displacement and skew effects simulating digital signal corruption. The animation SHALL only run when `prefers-reduced-motion: no-preference`.

#### Scenario: Error element with fx-glitch displays transform glitch
- **WHEN** an element has the `fx-glitch` class applied and the user has not enabled reduced motion
- **THEN** the element SHALL periodically translate horizontally by 1-2px and skew by up to 0.5deg

#### Scenario: fx-glitch respects reduced motion preference
- **WHEN** the user has enabled `prefers-reduced-motion: reduce`
- **THEN** the fx-glitch animation SHALL NOT run

### Requirement: Pulse glow animation available as fx-pulse utility
The system SHALL provide an `fx-pulse` CSS utility class that applies a pulsing box-shadow animation on a 3-second ease-in-out infinite loop. The animation SHALL oscillate box-shadow from 8px to 40px blur using the --glow-primary token. The animation SHALL only run when `prefers-reduced-motion: no-preference`.

#### Scenario: Active status indicator with fx-pulse has pulsing glow
- **WHEN** an element has the `fx-pulse` class applied
- **THEN** the element SHALL have a box-shadow that pulses between 8px and 40px blur in amber

#### Scenario: fx-pulse respects reduced motion preference
- **WHEN** the user has enabled `prefers-reduced-motion: reduce`
- **THEN** the fx-pulse animation SHALL NOT run

### Requirement: Noise texture available as fx-noise utility
The system SHALL provide an `fx-noise` CSS utility class that renders a radial-gradient dot pattern pseudo-element overlay simulating phosphor noise. The overlay SHALL use a 3px x 3px dot grid at 20% opacity with mix-blend-mode: overlay.

#### Scenario: Background panel with fx-noise displays dot texture
- **WHEN** an element has the `fx-noise` class applied
- **THEN** a ::before pseudo-element SHALL render with a subtle dot grid pattern

#### Scenario: Noise overlay does not block interaction
- **WHEN** a user interacts with content inside an fx-noise element
- **THEN** the interaction SHALL pass through the noise overlay (pointer-events: none)

### Requirement: Border flash animation available as fx-border-flash utility
The system SHALL provide an `fx-border-flash` CSS utility class that alternates border-color between the default border color and destructive red with a danger glow on a 1-second stepped infinite loop.

#### Scenario: Emergency state element flashes red border
- **WHEN** an element has the `fx-border-flash` class applied
- **THEN** the element border SHALL alternate between the default border color and destructive red at 1-second intervals

### Requirement: Text flicker animation available as fx-text-flicker utility
The system SHALL provide an `fx-text-flicker` CSS utility class that applies a rapid opacity variation (0.8-1.0) on a 0.1-second infinite loop simulating vintage terminal text rendering.

#### Scenario: Terminal text with fx-text-flicker has rapid opacity changes
- **WHEN** an element has the `fx-text-flicker` class applied
- **THEN** the element opacity SHALL vary rapidly between 0.8 and 1.0

### Requirement: Panel depth effect available as fx-panel utility
The system SHALL provide an `fx-panel` CSS utility class that applies a subtle inset highlight and 1px border ring via box-shadow, creating a Linear-style panel depth effect without using visible borders.

#### Scenario: Card with fx-panel has subtle depth
- **WHEN** a Card element has the `fx-panel` class applied
- **THEN** the element SHALL have an inset top highlight and a 1px border ring via box-shadow

### Requirement: Effects are applied to appropriate UI elements
The system SHALL apply CRT effects to specific element categories: fx-scanlines on content panels (Cards, log viewers, terminal containers); fx-glow on primary headings and active navigation items; fx-flicker on the application root wrapper; fx-glitch only on error/destructive states; fx-pulse on active/running status indicators; fx-box-glow on focused inputs and selected interactive elements.

#### Scenario: Log viewer has scanline effect
- **WHEN** the LogViewer component renders
- **THEN** its container Card SHALL have the `fx-scanlines` class

#### Scenario: Shell terminal has scanline and flicker effects
- **WHEN** the ShellTerminal component renders
- **THEN** its container SHALL have both `fx-scanlines` and `fx-flicker` classes

#### Scenario: Running status badge has pulse glow
- **WHEN** a "running" status badge is displayed
- **THEN** it SHALL have the `fx-pulse` class applied

#### Scenario: Section headings have glow effect
- **WHEN** a primary section heading (h1 or h2) is rendered in the layout
- **THEN** it SHALL have the `fx-glow` class applied

#### Scenario: Error boundary has glitch effect
- **WHEN** the ErrorBoundary component displays an error state
- **THEN** the error content SHALL have the `fx-glitch` class applied

### Requirement: MU-TH-UR scrollbar styling
The system SHALL apply custom scrollbar styling matching the MU-TH-UR aesthetic: 6px width/height, black track with 1px left border, muted thumb with 0px border-radius, and amber thumb on hover. This SHALL apply globally via ::-webkit-scrollbar pseudo-elements.

#### Scenario: Scrollbar matches MU-TH-UR aesthetic
- **WHEN** a scrollable area is rendered with visible scrollbar
- **THEN** the scrollbar SHALL be 6px wide with a black track, muted gray thumb, and amber thumb on hover
