## ADDED Requirements

### Requirement: Handle check_run completed events
The webhook handler SHALL accept `check_run` events in addition to the existing `push` events. When the `X-GitHub-Event` header is `check_run`, the handler SHALL parse the payload as a `checkRunPayload` and route it to the CI autofix flow.

#### Scenario: check_run event with conclusion failure on aot branch
- **WHEN** a `check_run` event arrives with `action` = `completed`, `conclusion` = `failure`, and the head branch starts with `aot/`
- **THEN** the handler triggers the CI autofix flow for that branch and repository
- **AND** returns HTTP 200 with `{"ok": true, "autofix": true}`

#### Scenario: check_run event with conclusion success
- **WHEN** a `check_run` event arrives with `action` = `completed` and `conclusion` = `success`
- **THEN** the handler returns HTTP 200 with `{"ok": true, "message": "check passed, no action needed"}`
- **AND** no autofix flow is triggered

#### Scenario: check_run event on non-aot branch
- **WHEN** a `check_run` event arrives for a branch that does not start with `aot/`
- **THEN** the handler returns HTTP 200 with `{"ok": true, "message": "branch not managed by aot"}`
- **AND** no autofix flow is triggered

### Requirement: Reuse existing webhook infrastructure
The `check_run` handler SHALL reuse the existing signature validation, repo allowlist check, and GitHub token provider from `WebhookHandler`. No new HTTP endpoints SHALL be created; the existing `/api/v1/webhooks/github` endpoint SHALL handle both `push` and `check_run` events based on the `X-GitHub-Event` header.

#### Scenario: check_run event with invalid signature
- **WHEN** a `check_run` event arrives with an invalid `X-Hub-Signature-256` header and a webhook secret is configured
- **THEN** the handler returns HTTP 401 with "invalid signature"

#### Scenario: check_run event from disallowed repo
- **WHEN** a `check_run` event arrives from a repository not in the allowlist
- **THEN** the handler returns HTTP 200 with `{"ok": true, "message": "repo not in allowlist"}`

### Requirement: Extract check run metadata from payload
The handler SHALL extract the `check_run.id`, `check_run.head_sha`, `check_run.name`, `repository.full_name`, and the head branch name from the `check_run` event payload. These fields SHALL be passed to the CI log extraction step.

#### Scenario: Payload contains all required fields
- **WHEN** a valid `check_run` payload is received
- **THEN** the handler extracts the check run ID, head SHA, check name, repo full name, and head branch
- **AND** passes them to the log extraction function

#### Scenario: Payload is malformed
- **WHEN** the `check_run` payload cannot be unmarshalled
- **THEN** the handler returns HTTP 400 with "invalid payload"
