## ADDED Requirements

### Requirement: Trace detail panel shows tool output
The SpanDetail panel SHALL display the tool's output/result alongside the tool input for tool spans.

#### Scenario: Tool output shown after tool input
- **WHEN** the user opens a span detail for a tool span that has `toolOutput` in its metadata
- **THEN** a "Tool Output" collapsible section is shown immediately after the "Tool Input" section
- **AND** the section is collapsed by default

#### Scenario: Long tool output is truncated
- **WHEN** the tool output exceeds 256 lines
- **THEN** only the first 256 lines are displayed
- **AND** a notice "(truncated — view full output in logs)" is shown below the truncated content

#### Scenario: No tool output for non-tool spans
- **WHEN** the user opens a span detail for a span that has no `toolOutput` metadata field
- **THEN** no "Tool Output" section is rendered
