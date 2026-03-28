## ADDED Requirements

### Requirement: TUI client connects to configured gRPC server
The TUI SHALL connect to the gRPC address stored in `~/.config/uncworks/config.yaml` under `server.address`, or to `localhost:50055` if no address is configured.

#### Scenario: Connects to local server
- **WHEN** `uncworks tui` is run with no stored server address
- **THEN** the TUI connects to `localhost:50055` via `kubectl port-forward` (same as `uncworks open`)

#### Scenario: Connects to remote server
- **WHEN** `uncworks connect grpc.example.com:50055` has been run previously
- **THEN** `uncworks tui` connects directly to `grpc.example.com:50055` without starting a port-forward

### Requirement: Run list view
The TUI SHALL display a paginated list of agent runs with their status, repository, branch, and elapsed time.

#### Scenario: Run list renders
- **WHEN** the TUI launches and connects successfully
- **THEN** a list of recent agent runs is shown, each with: status indicator, repo name, branch, start time, and duration

#### Scenario: Status updates live
- **WHEN** an agent run changes status while the TUI is open
- **THEN** the run list updates in-place without a full re-render

### Requirement: Log streaming view
The TUI SHALL allow navigating into a selected run to stream its live output.

#### Scenario: Enter log view
- **WHEN** the user selects a run from the run list and presses Enter
- **THEN** the TUI switches to a log streaming view showing the run's output in real-time

#### Scenario: Scroll log history
- **WHEN** in log view
- **THEN** the user can scroll up through historical output and the viewport follows new output unless scrolled up

### Requirement: Submit new run
The TUI SHALL provide a form to submit a new agent run.

#### Scenario: Submit run form
- **WHEN** the user presses `n` from the run list view
- **THEN** a form is shown with fields for repository URL, branch, and spec content

#### Scenario: Run submitted
- **WHEN** the form is submitted with valid input
- **THEN** a new agent run is created via gRPC and the TUI returns to the run list with the new run visible

### Requirement: Keyboard navigation
The TUI SHALL be fully keyboard-navigable with no mouse required.

#### Scenario: Navigation keys
- **WHEN** the TUI is open
- **THEN** j/k or arrow keys navigate the list, Enter selects, q or Escape returns to the previous view, ? shows a key binding reference

### Requirement: Graceful connection failure
The TUI SHALL display a clear error when it cannot connect to the gRPC server, with a hint on how to resolve it.

#### Scenario: Connection refused
- **WHEN** the gRPC server is unreachable
- **THEN** the TUI displays "Cannot connect to UNCWORKS server at <address>. Run `uncworks open` to start the local server." and exits

### Requirement: TUI renders correctly on macOS Terminal.app and Linux terminals
The TUI SHALL use only Unicode box-drawing characters and standard 256-color ANSI codes so it renders correctly in Terminal.app (macOS), iTerm2, and common Linux terminal emulators (Gnome Terminal, Konsole, WezTerm).

#### Scenario: No rendering artifacts in Terminal.app
- **WHEN** `uncworks tui` is run inside macOS Terminal.app
- **THEN** borders, lists, and log output render without garbled characters
