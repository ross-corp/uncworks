## MODIFIED Requirements

### Requirement: Verification gates fail on real errors
The verification gates SHALL NOT swallow errors. All OpenSpec CLI calls SHALL capture stdout, stderr, and exit code, and fail the gate with a clear error message when commands fail.

#### Scenario: openspec list failure reported
- **WHEN** `openspec list --json` fails or returns invalid JSON
- **THEN** the task completion gate fails with the error message and stderr output

#### Scenario: openspec validate failure reported
- **WHEN** `openspec validate --json` reports `valid: false`
- **THEN** the structural validation gate fails with the specific validation issues

#### Scenario: No error swallowing
- **WHEN** any verification gate runs an OpenSpec CLI command
- **THEN** stderr is NOT redirected to `/dev/null`
- **AND** exit codes are checked (non-zero = failure)

### Requirement: Archive gate reports errors
The archive gate SHALL NOT use `|| true`. If `openspec archive` fails, the error SHALL be included in the verification result.

#### Scenario: Archive failure reported
- **WHEN** `openspec archive --yes` fails (e.g., incomplete tasks, validation error)
- **THEN** the archive gate fails with the error output
- **AND** the verification result includes the archive error

### Requirement: JSON parsing uses Go, not shell pipes
The verification activity SHALL parse OpenSpec CLI JSON output using Go's `json.Unmarshal`, not piped through python3 or shell commands.

#### Scenario: Direct JSON parsing
- **WHEN** `openspec list --json` returns JSON with a leading text prefix (e.g., "- Loading...")
- **THEN** the Go code strips the prefix and parses the JSON directly
- **AND** no python3 or shell text processing is used
