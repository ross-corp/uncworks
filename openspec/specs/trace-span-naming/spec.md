# trace-span-naming Specification

## Purpose
TBD - created by archiving change observability-ux-overhaul. Update Purpose after archive.
## Requirements
### Requirement: Span names include tool kind
The system SHALL include the specific tool name in span names for tool execution spans, using the format `{role}.{toolName}`.

#### Scenario: Write tool span named correctly
- **WHEN** the implementer agent executes a `write` tool
- **THEN** the span name SHALL be `implement.write` (not `implement.tool`)

#### Scenario: Bash tool span named correctly
- **WHEN** the manager agent executes a `bash` tool
- **THEN** the span name SHALL be `manage.bash`

#### Scenario: Thought spans unchanged
- **WHEN** the implementer agent produces an LLM response
- **THEN** the span name SHALL be `implement.thought`

### Requirement: Span metadata includes tool input summary
The system SHALL include a truncated summary of the tool input in span metadata for tool execution spans.

#### Scenario: Write tool metadata shows path
- **WHEN** an `implement.write` span is created for writing to `/workspace/neph.nvim/HELLO.md`
- **THEN** the span metadata SHALL include `toolInput` with at minimum the file path

#### Scenario: Bash tool metadata shows command
- **WHEN** a `manage.bash` span is created for running `ls -la /workspace`
- **THEN** the span metadata SHALL include `toolInput` with the command string (truncated to 200 chars)

