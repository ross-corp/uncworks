## Purpose

Add unit tests for the most critical untested code paths in server, sidecar, and temporal packages.

## ADDED Requirements

### Requirement: Server JSONL parsing is tested
Tests SHALL verify that `parseAgentJSONL` correctly deduplicates entries and handles malformed lines.

#### Scenario: Duplicate message IDs
- **WHEN** the JSONL file contains two entries with the same message ID
- **THEN** only the latest entry is returned

#### Scenario: Malformed JSONL line
- **WHEN** a line is not valid JSON
- **THEN** the line is skipped without crashing

### Requirement: Server file listing is tested
Tests SHALL verify `isHiddenDir` and `parseLsOutput` behavior.

#### Scenario: Hidden directory filtered
- **WHEN** the path starts with `.` (e.g., `.git`)
- **THEN** `isHiddenDir` returns true

### Requirement: Sidecar exec workdir is tested
Tests SHALL verify that `ExecCommand` uses the correct working directory.

#### Scenario: Workdir set correctly
- **WHEN** ExecCommand is called with a specific workdir
- **THEN** the underlying exec.Cmd.Dir matches that workdir

### Requirement: Sidecar loop detection is tested
Tests SHALL verify `extractToolCallSignature` identifies repeated tool call patterns.

#### Scenario: Identical consecutive tool calls
- **WHEN** the same tool call signature appears 3+ times consecutively
- **THEN** loop detection triggers

### Requirement: Temporal plan/verify is tested
Tests SHALL verify PlanRun scaffolding and VerifyRun gate logic without LLM calls.

#### Scenario: VerifyRun gates
- **WHEN** a spec has lint errors
- **THEN** the lint gate fails and the run is marked as needing fixes
