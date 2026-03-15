## ADDED Requirements

### Requirement: Graph renders spec → senior → junior tree hierarchy
The orchestration graph SHALL render a vertical tree layout showing the spec run as the root node, senior agents as children, and junior agents as grandchildren. Each node SHALL display the run ID, agent type, and current phase.

#### Scenario: Simple spec with one senior and two juniors
- **WHEN** a spec run has spawned one senior agent which has spawned two junior agents
- **THEN** the graph renders a tree with the spec node at the top
- **AND** the senior node is centered below the spec node
- **AND** the two junior nodes are side-by-side below the senior node
- **AND** edges connect parent to child nodes

#### Scenario: Empty spec run (no agents yet)
- **WHEN** a spec run has just started and no agents have been spawned
- **THEN** the graph renders only the root spec node with status "Pending"

#### Scenario: Multiple senior agents
- **WHEN** a spec run spawns three senior agents
- **THEN** all three senior nodes appear at the same level below the spec node
- **AND** each senior's juniors appear below their respective parent

### Requirement: Graph nodes show real-time status
Each graph node SHALL reflect the current phase of its corresponding run. Phase transitions SHALL update the node's visual state within one frame of receiving the SSE event.

#### Scenario: Agent transitions from Pending to Running
- **WHEN** an agent's phase changes from Pending to Running
- **THEN** the graph node updates to show the Running visual state (phosphor glow)
- **AND** the node label updates to show "RUNNING"

#### Scenario: Agent fails
- **WHEN** an agent's phase changes to Failed
- **THEN** the graph node updates to show the Failed visual state (amber pulse)
- **AND** the node label updates to show "FAILED"

#### Scenario: Agent succeeds
- **WHEN** an agent's phase changes to Succeeded
- **THEN** the graph node updates to show the Succeeded visual state (dim green, no glow)

### Requirement: Click node to show detail panel
Clicking a graph node SHALL open a detail panel on the right side showing that run's trace timeline, output, and metadata. The graph SHALL shrink to accommodate the panel.

#### Scenario: Click a running agent node
- **WHEN** user clicks a node in the orchestration graph
- **THEN** a detail panel slides in from the right (40% width)
- **AND** the graph area shrinks to 60% width
- **AND** the detail panel shows the selected run's trace timeline and metadata
- **AND** the clicked node gets a highlighted border

#### Scenario: Click a different node while panel is open
- **WHEN** user clicks a different node while the detail panel is open
- **THEN** the panel content updates to show the newly selected run
- **AND** the previously selected node loses its highlight
- **AND** the newly selected node gets the highlight

#### Scenario: Close detail panel
- **WHEN** user clicks the close button on the detail panel (or clicks the same node again)
- **THEN** the panel slides out
- **AND** the graph expands back to full width

### Requirement: Graph receives real-time updates via SSE
The graph SHALL subscribe to a WatchSpecRunGraph SSE stream that delivers node_added, node_status_changed, and node_progress events. The graph SHALL update reactively without full refetches.

#### Scenario: New agent spawned during execution
- **WHEN** a senior agent spawns a new junior agent
- **THEN** the graph receives a node_added event
- **AND** a new node appears in the graph with an entrance animation
- **AND** the tree layout recalculates to accommodate the new node

#### Scenario: SSE stream disconnects
- **WHEN** the SSE connection drops
- **THEN** the system automatically reconnects
- **AND** a "RECONNECTING..." indicator appears on the graph
- **AND** on reconnection, the full graph state is refetched to sync

### Requirement: Graph API endpoints exist
The API server SHALL expose GetSpecRunGraph and WatchSpecRunGraph RPCs. GetSpecRunGraph returns the full graph structure (all nodes and edges). WatchSpecRunGraph streams incremental updates.

#### Scenario: Fetch graph for a spec run
- **WHEN** a client calls GetSpecRunGraph with a spec run ID
- **THEN** the response contains all agent run nodes with their parent relationships and current phases

#### Scenario: Watch graph for a spec run
- **WHEN** a client calls WatchSpecRunGraph with a spec run ID
- **THEN** the server streams events as agents are spawned and phases change
- **AND** the stream remains open until the client disconnects or the spec run completes

### Requirement: Graph layout handles edge cases
The tree layout SHALL handle degenerate cases gracefully.

#### Scenario: Deep tree (4+ levels)
- **WHEN** the orchestration tree has more than 3 levels of nesting
- **THEN** the graph renders all levels with vertical scrolling enabled

#### Scenario: Wide tree (10+ siblings)
- **WHEN** a single parent has more than 10 children
- **THEN** the graph renders all children horizontally with horizontal scrolling enabled
- **AND** a "fit to view" button appears to zoom out
