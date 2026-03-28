## ADDED Requirements

### Requirement: GitHub authentication uses device flow OAuth
The system SHALL authenticate with GitHub using the OAuth device flow (RFC 8628). Raw personal access tokens SHALL NOT be accepted via the wizard. The `gh` CLI client_id SHALL be used.

#### Scenario: Device flow initiation
- **WHEN** the user clicks "Connect GitHub" in wizard step 2
- **THEN** the app requests a device code from GitHub and displays the user code and verification URL prominently

#### Scenario: Successful authentication
- **WHEN** the user approves the device code in their browser
- **THEN** the app exchanges the device code for a token, stores it in the macOS Keychain, and marks step 2 complete

#### Scenario: Authentication timeout
- **WHEN** the device code expires (15 minutes) before the user approves
- **THEN** the app shows an error and offers a "Try again" button to restart device flow

### Requirement: GitHub token stored in macOS Keychain
The system SHALL store the GitHub OAuth token in the macOS Keychain (service: `uncworks`, account: `github-token`). The token SHALL NOT be stored in plaintext in `config.json`.

#### Scenario: Token written to Keychain
- **WHEN** the OAuth flow completes successfully
- **THEN** the token is written to Keychain and `config.json` stores only a flag `githubAuthed: true`

#### Scenario: Token retrieval on launch
- **WHEN** the app needs to make a GitHub API call
- **THEN** it reads the token from Keychain, not from `config.json`

### Requirement: GitHub auth status shown in Settings
The system SHALL display the connected GitHub account name (from `GET /user`) and a "Disconnect" button in the Settings page when authenticated.

#### Scenario: Authenticated state display
- **WHEN** a GitHub token is present in Keychain
- **THEN** Settings shows "Connected as @username" with a Disconnect button

#### Scenario: Disconnect
- **WHEN** the user clicks "Disconnect"
- **THEN** the token is removed from Keychain and the UI shows "Not connected"

### Requirement: ANTHROPIC_API_KEY field is removed from settings
The system SHALL NOT expose an `llmKey` or `ANTHROPIC_API_KEY` field in the settings UI. Provider API keys are managed in LiteLLM, not in UNCWORKS.

#### Scenario: Legacy llmKey config migration
- **WHEN** a `config.json` contains a `llmKey` field from a previous version
- **THEN** the field is silently ignored on load and omitted on next save
