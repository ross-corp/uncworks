## ADDED Requirements

### Requirement: TUI enters raw mode and renders dashboard
The TUI application SHALL enter raw terminal mode on startup, render the dashboard view, and restore terminal state on exit.

#### Scenario: Clean startup and shutdown
- **WHEN** the user runs `aot dashboard`
- **THEN** the terminal enters raw mode, the dashboard renders, and pressing `q` restores the terminal and exits cleanly

### Requirement: Keyboard navigation in agent list
The TUI SHALL support arrow key navigation (Up/Down) to select agent runs and Enter to view details.

#### Scenario: Navigate and select
- **WHEN** the user presses Down arrow twice then Enter
- **THEN** the third agent run is selected AND the detail panel shows its information

### Requirement: Screen re-render on state change
The TUI SHALL clear the screen and re-render when the agent run list or selected run state changes.

#### Scenario: Phase change triggers re-render
- **WHEN** a watched agent run transitions from Running to Succeeded
- **THEN** the dashboard re-renders within one frame (16ms) showing the updated phase

### Requirement: HITL text input mode
The TUI SHALL provide an inline text input when the user presses Enter on a WaitingForInput agent.

#### Scenario: Submit human input
- **WHEN** the selected agent is in WaitingForInput phase AND the user presses Enter, types a response, and presses Enter again
- **THEN** the response is sent via `SendHumanInput` gRPC call AND the input prompt closes

#### Scenario: Cancel input
- **WHEN** the user is in input mode AND presses Escape
- **THEN** the input is cancelled AND the TUI returns to navigation mode
