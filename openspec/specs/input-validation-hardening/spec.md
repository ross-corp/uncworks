# input-validation-hardening Specification

## Purpose
TBD - created by archiving change code-quality-hardening. Update Purpose after archive.
## Requirements
### Requirement: Repository path is validated before use
The hydration system SHALL reject any `Repository.Path` value that could escape the workspace directory. Specifically: absolute paths (starting with `/`), paths containing `..` components, and paths that when cleaned resolve to a string starting with `..` MUST be rejected with an error before any filesystem operation.

#### Scenario: Path traversal attempt is rejected
- **WHEN** `Repository.Path` is set to `../../etc/passwd`
- **THEN** hydration returns an error containing "path traversal not allowed"
- **THEN** no filesystem operations are performed

#### Scenario: Absolute path is rejected
- **WHEN** `Repository.Path` is set to `/workspace/malicious`
- **THEN** hydration returns an error containing "path must be relative"

#### Scenario: Valid nested path is accepted
- **WHEN** `Repository.Path` is set to `services/api`
- **THEN** hydration proceeds normally using that path as the workspace subdirectory

#### Scenario: Empty path uses default derivation
- **WHEN** `Repository.Path` is empty string
- **THEN** hydration derives the path from the repository URL as before

### Requirement: Git URLs with tokens are validated before injection
The hydration system SHALL use proper URL parsing when injecting GitHub tokens. It MUST validate that the URL scheme is `https` and the host matches an allowlist (defaulting to `github.com`) before injecting credentials. URLs that fail validation MUST be returned unchanged without token injection.

#### Scenario: Token injected into valid GitHub HTTPS URL
- **WHEN** `Repository.URL` is `https://github.com/org/repo.git`
- **THEN** the injected URL is `https://x-access-token:TOKEN@github.com/org/repo.git`

#### Scenario: Crafted URL with attacker host is rejected
- **WHEN** `Repository.URL` is `https://github.com@attacker.com/org/repo.git`
- **THEN** the URL is returned unchanged (host parses as `attacker.com`, not in allowlist)
- **THEN** no token is injected

#### Scenario: SSH URL is passed through unchanged
- **WHEN** `Repository.URL` is `git@github.com:org/repo.git`
- **THEN** the URL is returned unchanged (not https scheme)

