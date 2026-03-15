## ADDED Requirements

### Requirement: Timeline renders spans with MU-TH-UR aesthetic
The trace timeline SHALL display agent activity spans as horizontal bars on a time axis. The visual treatment SHALL follow the MU-TH-UR design language: dark background (#0a0a0a), phosphor green (#00ff41) for active elements, CRT scanline overlay, and monospace typography.

#### Scenario: Timeline displays completed spans
- **WHEN** the timeline loads for a run with completed spans
- **THEN** each span is rendered as a horizontal bar proportional to its duration
- **AND** span labels are in monospace font (JetBrains Mono), 11px, uppercase
- **AND** completed spans are dim green (#1a3a1a) without glow
- **AND** a faint CRT scanline overlay is visible on the background

#### Scenario: Timeline displays active spans
- **WHEN** a span is currently in progress (no end time)
- **THEN** the span bar has a phosphor green glow (box-shadow: 0 0 8px #00ff41)
- **AND** the span's right edge animates to grow with elapsed time
- **AND** the duration label updates in real-time

#### Scenario: Timeline displays failed spans
- **WHEN** a span completed with an error
- **THEN** the span bar is amber (#ff6600) with a pulse animation

### Requirement: Spans are interactive
Users SHALL be able to hover and click spans for details. Tool-call spans that modified files SHALL be clickable to open the diff viewer.

#### Scenario: Hover shows tooltip
- **WHEN** user hovers over a span
- **THEN** a tooltip appears showing span name, start time, duration, and file list (if tool-call)
- **AND** the tooltip appears within 100ms of hover

#### Scenario: Click tool-call span opens diff viewer
- **WHEN** user clicks a tool-call span that has file modifications
- **THEN** the diff viewer opens below the timeline showing the before/after of modified files
- **AND** if multiple files were modified, a file list sidebar allows switching between them

#### Scenario: Click non-file span shows detail
- **WHEN** user clicks a think or test span (no file modifications)
- **THEN** a detail panel below the timeline shows the span's full content (thought text, test output)

### Requirement: Timeline integrates with orchestration graph
When a node is selected in the orchestration graph, the timeline SHALL show that run's spans. The timeline header SHALL indicate which run is being displayed.

#### Scenario: Select node shows that run's timeline
- **WHEN** user clicks a node in the orchestration graph
- **THEN** the trace timeline in the detail panel loads the selected run's spans
- **AND** the timeline header shows the run ID and agent type

#### Scenario: Switch between nodes
- **WHEN** user clicks a different node in the graph
- **THEN** the timeline updates to show the newly selected run's spans
- **AND** the previous run's spans are replaced

### Requirement: Timeline handles large span counts
The timeline SHALL remain performant with large numbers of spans by virtualizing the rendering.

#### Scenario: Run with 500 spans
- **WHEN** a run has 500 spans
- **THEN** only the visible spans (in the current scroll viewport) are rendered as DOM elements
- **AND** scrolling is smooth (60fps)

#### Scenario: Run with active streaming
- **WHEN** new spans arrive via the event stream
- **THEN** spans are appended to the timeline without re-rendering existing spans
- **AND** if the user has not scrolled, the timeline auto-scrolls to show the latest span

### Requirement: Respects accessibility preferences
The timeline SHALL respect the user's motion preferences.

#### Scenario: Reduced motion preference
- **WHEN** the user has prefers-reduced-motion enabled
- **THEN** glow animations are disabled
- **AND** pulse animations are replaced with static color changes
- **AND** span growth animations are instant rather than smooth
