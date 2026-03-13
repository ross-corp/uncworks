## ADDED Requirements

### Requirement: Push spec to GitHub repo
The system SHALL provide an API endpoint to push a spec file to a GitHub repository, creating a commit with the `.cs.md` file at a specified path.

#### Scenario: Push new spec to repo
- **WHEN** a user clicks "Push to GitHub" in the spec editor with repo `org/myproject` and path `spec/converter.cs.md`
- **THEN** the system creates a commit in `org/myproject` containing the spec content at `spec/converter.cs.md`
- **AND** the commit message references the spec name

#### Scenario: Update existing spec in repo
- **WHEN** a user pushes a spec to a path that already exists in the repo
- **THEN** the existing file is updated with the new content in a new commit

### Requirement: Pull spec from GitHub repo
The spec editor SHALL support loading a spec from a GitHub repository by specifying the repo and file path.

#### Scenario: Load spec from repo
- **WHEN** a user clicks "Load from GitHub" and enters repo `org/myproject` and path `spec/converter.cs.md`
- **THEN** the file content is fetched from GitHub and populated into the Monaco editor
- **AND** the `spec_source` is set to `"github:org/myproject/spec/converter.cs.md"`

#### Scenario: File not found
- **WHEN** a user attempts to load a spec from a path that does not exist
- **THEN** an error message is shown and the editor content is not modified

### Requirement: GitHub webhook receiver
The system SHALL expose a `POST /api/v1/webhooks/github` endpoint that receives GitHub push webhooks and creates spec-driven agent runs when `.cs.md` files are modified.

#### Scenario: Push with modified spec file
- **WHEN** a GitHub push webhook arrives with a commit that modified `spec/converter.cs.md`
- **THEN** the system reads the updated file content from GitHub
- **AND** creates an AgentRun with `spec_content` set to the file content and `spec_source` set to `"github:<repo>/<path>"`
- **AND** the repo from the webhook is included in the run's `repos[]`

#### Scenario: Push with no spec file changes
- **WHEN** a GitHub push webhook arrives with commits that only modify `.go` or `.ts` files
- **THEN** no agent run is created

#### Scenario: Push with multiple spec files changed
- **WHEN** a push modifies 3 different `.cs.md` files
- **THEN** one agent run is created per modified spec file

### Requirement: Webhook signature validation
The webhook endpoint SHALL validate the GitHub webhook signature using HMAC-SHA256 with a configured shared secret.

#### Scenario: Valid signature
- **WHEN** a webhook request arrives with a valid `X-Hub-Signature-256` header
- **THEN** the request is processed normally

#### Scenario: Invalid signature
- **WHEN** a webhook request arrives with an invalid or missing signature
- **THEN** the request is rejected with HTTP 401

### Requirement: Webhook configuration
The system SHALL support configuring which repos and file patterns trigger spec runs via webhooks.

#### Scenario: Configured repo receives push
- **WHEN** repo `org/myproject` is configured for webhook triggers and a push arrives
- **THEN** the push is processed for `.cs.md` file changes

#### Scenario: Unconfigured repo receives push
- **WHEN** a push webhook arrives from a repo not configured for triggers
- **THEN** the request is acknowledged (HTTP 200) but no run is created

### Requirement: Spec source tracking
The `spec_source` field SHALL track the origin of a spec for provenance and display purposes.

#### Scenario: Spec authored in editor
- **WHEN** a user writes a spec in the web UI editor and creates a run
- **THEN** `spec_source` is set to `"editor"`

#### Scenario: Spec loaded from GitHub
- **WHEN** a user loads a spec from GitHub and creates a run
- **THEN** `spec_source` is set to `"github:<owner>/<repo>/<path>"`

#### Scenario: Spec triggered by webhook
- **WHEN** a webhook triggers a spec run
- **THEN** `spec_source` is set to `"webhook:github:<owner>/<repo>/<path>"`

#### Scenario: Spec source displayed in detail panel
- **WHEN** a user views a spec-driven run in the detail panel
- **THEN** the spec source is shown as metadata (e.g., "Source: github:org/myproject/spec/converter.cs.md")
