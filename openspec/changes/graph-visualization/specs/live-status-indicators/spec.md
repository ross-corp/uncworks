## ADDED Requirements

### Requirement: Active nodes have phosphor pulse animation
Graph nodes in the Running phase SHALL display a phosphor green pulse animation indicating active processing.

#### Scenario: Agent starts running
- **WHEN** an agent's phase transitions to Running
- **THEN** the node border glows phosphor green (#00ff41) with a pulsing animation (1.5s cycle)
- **AND** the glow intensity alternates between 0.4 and 0.8 opacity

#### Scenario: Agent finishes running
- **WHEN** a running agent transitions to Succeeded or Failed
- **THEN** the pulse animation stops
- **AND** the node transitions to its final visual state (dim green for success, static amber for failure)

### Requirement: Root spec node shows radar sweep while orchestration is active
The root spec node SHALL display a Radar component (from the homelab design system) while any child agents are still running.

#### Scenario: Orchestration in progress
- **WHEN** at least one agent in the spec run is in a non-terminal phase (Pending or Running)
- **THEN** the spec root node displays a radar sweep animation behind the node content

#### Scenario: Orchestration complete
- **WHEN** all agents in the spec run have reached terminal phases (Succeeded, Failed, Cancelled)
- **THEN** the radar sweep stops and fades out over 500ms

### Requirement: Active nodes show current activity text
Graph nodes in the Running phase SHALL display a brief text indicator of the agent's current activity (e.g., "THINKING", "EDITING file.ts", "RUNNING TESTS").

#### Scenario: Activity text updates
- **WHEN** the SSE stream delivers a node_progress event with current_activity text
- **THEN** the corresponding graph node updates its activity label
- **AND** the label is truncated to 30 characters with ellipsis if needed

### Requirement: DataStream effect on nodes receiving output
Graph nodes that are actively producing output SHALL display a subtle DataStream hex waterfall effect in the node background.

#### Scenario: Agent producing output
- **WHEN** an agent is actively streaming output (events arriving)
- **THEN** the node background shows a subtle hex character waterfall (DataStream component, low opacity)

#### Scenario: Agent idle but running
- **WHEN** an agent is in Running phase but no events have arrived for 3+ seconds
- **THEN** the DataStream effect pauses (no animation, just static)

### Requirement: Animations respect prefers-reduced-motion
All animated status indicators SHALL be disabled or reduced when the user's system preference is prefers-reduced-motion.

#### Scenario: Reduced motion enabled
- **WHEN** the user has prefers-reduced-motion: reduce set
- **THEN** phosphor pulse is replaced with a static green border
- **AND** radar sweep is replaced with a static radar icon
- **AND** DataStream waterfall is replaced with a static hex pattern
- **AND** all transition animations are instant (0ms duration)
