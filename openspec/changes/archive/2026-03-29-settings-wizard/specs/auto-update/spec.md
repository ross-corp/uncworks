## ADDED Requirements

### Requirement: Auto-update is opt-in
The system SHALL NOT check for updates unless the user has explicitly enabled auto-update in Settings. The default SHALL be disabled.

#### Scenario: Auto-update disabled (default)
- **WHEN** auto-update is not enabled
- **THEN** no version check requests are made and no update UI is shown

#### Scenario: User enables auto-update
- **WHEN** the user enables auto-update in Settings
- **THEN** the app performs a version check at next launch and the result is shown in the Settings page

### Requirement: Update channels: stable and nightly
The system SHALL support two update channels: `stable` (GitHub Releases without pre-release flag) and `nightly` (GitHub Releases with pre-release flag matching `v*-pre.*`). The user SHALL be able to select their channel in Settings.

#### Scenario: Stable channel update available
- **WHEN** auto-update is enabled on stable channel and a newer stable version exists
- **THEN** Settings shows "Update available: v{new}" with a button that opens the GitHub Releases page in the browser

#### Scenario: Nightly channel update available
- **WHEN** auto-update is enabled on nightly channel and a newer nightly tag exists
- **THEN** Settings shows the nightly tag name and offers to open the releases page

#### Scenario: Already on latest
- **WHEN** the installed version matches the latest in the selected channel
- **THEN** Settings shows "Up to date (v{version})"

### Requirement: Local builds show a no-update state
The system SHALL detect when it is running a local (untagged) build. For local builds, auto-update SHALL be non-functional and Settings SHALL display "Local build — updates not available".

#### Scenario: Local build detection
- **WHEN** the embedded build version is empty or equal to `dev`
- **THEN** the update UI shows "Local build — updates not available" regardless of channel selection

### Requirement: Version check is cached per session
The system SHALL perform at most one GitHub API version check per app session to avoid rate limiting.

#### Scenario: Multiple settings page visits
- **WHEN** the user navigates to Settings multiple times in one session
- **THEN** the version check result from the first visit is reused; no additional API calls are made
