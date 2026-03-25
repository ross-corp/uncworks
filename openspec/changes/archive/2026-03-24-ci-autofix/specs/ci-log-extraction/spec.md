## ADDED Requirements

### Requirement: Fetch CI run logs from GitHub Actions API
The system SHALL fetch the logs for a failed GitHub Actions run using `GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs`. The request SHALL use the existing GitHub token provider for authentication. The response is a zip archive containing per-job log files.

#### Scenario: Logs are available for a failed run
- **WHEN** a failed check run has an associated Actions run ID
- **THEN** the system fetches the log zip from the GitHub Actions API
- **AND** extracts the text content from the zip archive

#### Scenario: GitHub API returns 404 for logs
- **WHEN** the GitHub Actions API returns 404 (logs expired or unavailable)
- **THEN** the system logs a warning and creates the autofix run with a generic "CI failed, logs unavailable" message in the prompt
- **AND** the autofix flow is not blocked

#### Scenario: GitHub API rate limit exceeded
- **WHEN** the GitHub Actions API returns 403 with a rate limit error
- **THEN** the system logs the error and does not trigger an autofix run for this failure

### Requirement: Parse and condense error messages
The system SHALL parse the raw CI log output to extract actionable error lines: compiler errors, test failures, lint violations, and build errors. The condensed output SHALL be no longer than 8000 characters to fit within LLM context limits. If the raw output exceeds this limit, the system SHALL prioritize error lines (lines containing "error", "Error", "FAIL", "failed") and truncate from the middle.

#### Scenario: Log contains compiler errors
- **WHEN** the CI log contains lines matching Go compiler error patterns (e.g., `file.go:42:5: undefined: Foo`)
- **THEN** the condensed output includes those lines with their file paths and line numbers

#### Scenario: Log contains test failures
- **WHEN** the CI log contains test failure output (e.g., `--- FAIL: TestFoo`, `FAIL github.com/...`)
- **THEN** the condensed output includes the failing test names and assertion messages

#### Scenario: Log exceeds size limit
- **WHEN** the raw CI log exceeds 8000 characters after error filtering
- **THEN** the system truncates from the middle, preserving the first and last error sections
- **AND** inserts a `[... truncated N lines ...]` marker at the truncation point

### Requirement: Map check_run to Actions run ID
The system SHALL use the GitHub API to resolve a `check_run` ID to the corresponding Actions workflow run ID. This requires calling `GET /repos/{owner}/{repo}/check-runs/{check_run_id}` to get the `details_url` or using the `check_suite.id` from the payload to find the associated workflow run via `GET /repos/{owner}/{repo}/actions/runs?check_suite_id={id}`.

#### Scenario: check_run maps to a workflow run
- **WHEN** the check_run payload includes a check_suite ID
- **THEN** the system resolves it to an Actions workflow run ID
- **AND** uses that run ID to fetch logs

#### Scenario: check_run does not map to an Actions run
- **WHEN** the check_run is from a third-party CI provider (not GitHub Actions)
- **THEN** the system logs an info message "check_run is not a GitHub Actions run, skipping"
- **AND** no autofix flow is triggered
