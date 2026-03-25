## ADDED Requirements

### Requirement: Broken clone directory is detected and recovered
The hydration system SHALL detect when a `.bare` clone directory exists but is not a valid git repository. When detected, it MUST remove the broken directory and reattempt the clone rather than proceeding with corrupt state.

#### Scenario: Broken bare directory is replaced
- **WHEN** a `.bare` clone directory exists from a previous failed clone
- **WHEN** `git rev-parse --git-dir` fails on that directory
- **THEN** the directory is removed
- **THEN** a fresh clone is performed

#### Scenario: Valid existing clone is reused
- **WHEN** a `.bare` clone directory exists and is a valid git repository
- **THEN** the clone step is skipped and the existing repo is used

### Requirement: Clone failure cleans up partial state
The hydration system SHALL remove any partially-created `.bare` directory when a `git clone` fails, so that subsequent retries do not encounter corrupt state.

#### Scenario: Clone failure removes partial directory
- **WHEN** `git clone` fails (network error, auth failure, etc.)
- **THEN** the partially-created `.bare` directory is removed
- **THEN** the error is returned to the caller

### Requirement: Malformed AOT_REPOS logs a warning
When the `AOT_REPOS` environment variable contains invalid JSON, the hydration system SHALL log a warning indicating the parse failure and fall back to single-repo mode. It MUST NOT silently ignore the malformed input.

#### Scenario: Invalid JSON triggers warning
- **WHEN** `AOT_REPOS` is set to `{invalid json}`
- **THEN** a warning is logged containing "failed to parse AOT_REPOS"
- **THEN** hydration falls back to single-repo mode using `AOT_REPO_URL`
