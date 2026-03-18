# persistent-run-storage Specification

## Purpose
TBD - created by archiving change persistent-knowledge-system. Update Purpose after archive.
## Requirements
### Requirement: Run logs are persisted to PostgreSQL
All agent run output (stdout, stderr, system messages) SHALL be stored in the `run_logs` table with a foreign key to `agent_states`. Logs SHALL survive PVC cleanup and be queryable by run ID.

#### Scenario: Logs persisted after run completion
- **WHEN** an agent run completes (any terminal phase: Succeeded, Failed, Cancelled)
- **THEN** all captured log output is written to the `run_logs` table
- **AND** each log entry references the run's `agent_run_id`
- **AND** each log entry has a `log_type` (stdout, stderr, system) and timestamp

#### Scenario: Logs queryable by run ID
- **WHEN** a caller queries `run_logs` by `agent_run_id`
- **THEN** all log entries for that run are returned in chronological order

### Requirement: Diffs are persisted per tool call
All file changes produced by an agent run SHALL be stored in the `run_diffs` table with file path, language, diff content, and optional tool call ID.

#### Scenario: Diffs persisted after run completion
- **WHEN** an agent run completes with file changes
- **THEN** each changed file has a corresponding row in `run_diffs`
- **AND** the `file_path`, `diff_content`, and `language` fields are populated
- **AND** if the change was made by a specific tool call, the `tool_call_id` is recorded

#### Scenario: Run with no file changes
- **WHEN** an agent run completes without modifying any files
- **THEN** no rows are inserted into `run_diffs` for that run

### Requirement: Trace spans are persisted
All trace spans from an agent run SHALL be stored in the `run_spans` table with span name, timing, attributes, and parent-child relationships.

#### Scenario: Spans persisted after run completion
- **WHEN** an agent run completes
- **THEN** all trace spans are written to the `run_spans` table
- **AND** each span has `span_name`, `start_time`, and `end_time`
- **AND** parent-child relationships are preserved via `parent_span_id`

#### Scenario: Spans include activity metadata
- **WHEN** a trace span represents a Temporal activity execution
- **THEN** the span's `attributes` JSONB field contains the activity name and input parameters

### Requirement: Schema migration extends existing brain store
The new tables SHALL be created by extending the existing `Store.Migrate()` method in `internal/brain/store.go`. The migration SHALL be idempotent (CREATE TABLE IF NOT EXISTS).

#### Scenario: Migration adds new tables without affecting existing data
- **WHEN** `Store.Migrate()` is called on a database with existing `agent_states` data
- **THEN** the new tables (`run_logs`, `run_diffs`, `run_spans`) are created
- **AND** existing `agent_states` data is unchanged
- **AND** foreign key constraints reference the existing `agent_states` table

#### Scenario: Migration is idempotent
- **WHEN** `Store.Migrate()` is called multiple times
- **THEN** no errors occur and no data is lost

### Requirement: Data retention is permanent
Run artifacts SHALL NOT have a TTL. Unlike PVCs (7-day TTL), PostgreSQL data SHALL be retained indefinitely unless explicitly deleted by an administrator.

#### Scenario: Old data remains accessible
- **WHEN** a run completed more than 7 days ago (beyond PVC TTL)
- **THEN** its logs, diffs, and spans are still queryable from PostgreSQL

