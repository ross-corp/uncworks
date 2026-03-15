## ADDED Requirements

### Requirement: Right pane opens when a run is selected
When a run is selected and the user opens detail (Enter key or double-click), the layout SHALL transition from single-column (`grid-template-columns: 1fr 0`) to split-pane (`grid-template-columns: 1fr 1fr`). The list remains visible on the left. The transition SHALL be animated via CSS.

#### Scenario: Open detail pane
- **WHEN** user selects a run and presses Enter
- **THEN** the right pane slides open showing run detail
- **AND** the left list pane remains visible and interactive

#### Scenario: Switch selected run while detail is open
- **WHEN** detail pane is open for run A
- **AND** user clicks run B in the list
- **THEN** the detail pane updates to show run B's information

### Requirement: Detail pane has tabbed content
The detail pane SHALL display tabs: Info, Logs, Files, Shell, Traces. Tab content SHALL reuse existing components: LogViewer, FileExplorer, ShellTerminal, TraceTimeline. The Info tab shows run metadata (ID, prompt, repo, phase, timestamps).

#### Scenario: Switch tabs
- **WHEN** detail pane is open
- **AND** user clicks the "Logs" tab
- **THEN** the LogViewer component renders in the detail pane

#### Scenario: Default tab
- **WHEN** detail pane opens for a run
- **THEN** the Info tab is selected by default

### Requirement: Resizable via drag handle
A vertical drag handle SHALL appear between the list and detail panes. The user SHALL be able to drag it to resize the panes. Minimum pane width is 200px. The drag handle SHALL show a visible grip indicator on hover.

#### Scenario: Resize panes
- **WHEN** user mousedowns on the drag handle and moves right
- **THEN** the list pane grows and the detail pane shrinks
- **AND** neither pane goes below 200px width

#### Scenario: Drag handle cursor
- **WHEN** user hovers over the drag handle
- **THEN** the cursor changes to `col-resize`

### Requirement: Close detail pane
The detail pane SHALL close when the user presses Escape (and no nested element like command palette is open) or re-clicks the currently selected run. Closing SHALL animate back to single-column layout.

#### Scenario: Close with Escape
- **WHEN** detail pane is open and command palette is closed
- **AND** user presses Escape
- **THEN** the detail pane closes with a slide animation
- **AND** the layout returns to single-column

#### Scenario: Close by re-clicking selected run
- **WHEN** run A is selected and detail is open
- **AND** user clicks run A again
- **THEN** the detail pane closes

### Requirement: Pane width persisted in localStorage
The last drag-handle position SHALL be stored in `localStorage` under key `clean-ui-pane-width`. On next open, the detail pane SHALL restore this width. If the stored value is outside [200px, 80vw], the default 50% split SHALL be used.

#### Scenario: Width restored on reopen
- **WHEN** user resizes detail pane to 600px and closes it
- **AND** user opens detail pane again
- **THEN** the detail pane opens at 600px width

#### Scenario: Invalid stored width is ignored
- **WHEN** localStorage contains a pane width of 50px (below minimum)
- **AND** user opens detail pane
- **THEN** the detail pane opens at default 50% width
