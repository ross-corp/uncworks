## ADDED Requirements

### Requirement: Monaco-based side-by-side diff viewer
The diff viewer SHALL use Monaco Editor's diff mode to display before/after file content side-by-side with syntax highlighting, aligned line numbers, and change highlighting.

#### Scenario: View diff for a single file change
- **WHEN** user clicks a tool-call span that modified one file
- **THEN** the diff viewer opens showing the before (left) and after (right) versions
- **AND** syntax highlighting is applied based on file extension
- **AND** changed lines are highlighted with background color
- **AND** added lines are marked green, removed lines marked red

#### Scenario: View diff for multiple file changes
- **WHEN** user clicks a tool-call span that modified multiple files
- **THEN** the diff viewer opens with a file list sidebar on the left
- **AND** clicking a file in the sidebar shows that file's diff in the main area
- **AND** the file list shows change summary (lines added/removed) per file

#### Scenario: Toggle inline vs side-by-side
- **WHEN** user clicks the "inline" toggle in the diff viewer header
- **THEN** the diff switches from side-by-side to inline (unified) mode
- **AND** the toggle state persists for the session

### Requirement: Diff viewer loads lazily
Monaco Editor SHALL be loaded via dynamic import when the diff viewer is first opened. A loading state SHALL be shown during the load.

#### Scenario: First diff open
- **WHEN** user opens the diff viewer for the first time in a session
- **THEN** a loading skeleton appears with "LOADING DIFF..." text in MU-TH-UR style
- **AND** Monaco loads in the background via dynamic import
- **AND** once loaded, the diff renders and the skeleton is replaced

#### Scenario: Subsequent diff opens
- **WHEN** user opens the diff viewer after Monaco has already been loaded
- **THEN** the diff renders immediately without a loading state

### Requirement: Diff viewer accessible from timeline and completion summary
The diff viewer SHALL be openable from two entry points: clicking a file-modifying span in the trace timeline, or clicking a file in the completion summary's aggregated diff list.

#### Scenario: Open from trace timeline
- **WHEN** user clicks a tool-call span with file modifications in the trace timeline
- **THEN** the diff viewer opens below the timeline within the detail panel

#### Scenario: Open from completion summary
- **WHEN** user clicks a modified file in the completion summary
- **THEN** the diff viewer opens in a modal overlay on top of the summary

### Requirement: Diff viewer shows MU-TH-UR styling
The diff viewer SHALL apply MU-TH-UR visual treatment to the Monaco editor container.

#### Scenario: Styled diff viewer
- **WHEN** the diff viewer is open
- **THEN** the editor uses a dark theme matching the MU-TH-UR palette (#0a0a0a background, #00ff41 accent)
- **AND** the file list sidebar has monospace labels
- **AND** the viewer header shows the file path in phosphor green
