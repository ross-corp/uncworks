## ADDED Requirements

### Requirement: Wails v2 macOS application bundle
The system SHALL provide a native macOS application (`UNCWORKS.app`) built with Wails v2 that embeds the existing React web frontend in a WKWebView.

#### Scenario: App launches and shows web UI
- **WHEN** `UNCWORKS.app` is opened on macOS
- **THEN** a native window opens showing the UNCWORKS web UI served from the embedded frontend assets

### Requirement: Minimum macOS version
The app SHALL target macOS 13 (Ventura) as the minimum supported version.

#### Scenario: Runs on macOS 13
- **WHEN** the app is launched on macOS 13.x
- **THEN** the app launches without errors and the web UI renders correctly

### Requirement: Menu bar status indicator
The app SHALL display a menu bar icon that shows the current cluster status (running / stopped / unknown).

#### Scenario: Cluster running
- **WHEN** the UNCWORKS stack is healthy in the local cluster
- **THEN** the menu bar icon shows a filled indicator and the tooltip displays "UNCWORKS — Running"

#### Scenario: Cluster stopped or unreachable
- **WHEN** the UNCWORKS stack cannot be reached
- **THEN** the menu bar icon shows an outlined indicator and the tooltip displays "UNCWORKS — Stopped"

### Requirement: Start and stop cluster from menu bar
The app SHALL provide "Start" and "Stop" menu items that invoke `uncworks setup` and `uncworks teardown` respectively as subprocesses, with progress shown in the app window.

#### Scenario: Start cluster
- **WHEN** the user clicks "Start" in the menu bar menu
- **THEN** the app window shows setup wizard progress output and switches to the web UI on completion

#### Scenario: Stop cluster
- **WHEN** the user clicks "Stop" in the menu bar menu
- **THEN** `uncworks teardown` is invoked and the menu bar icon updates to stopped state

### Requirement: App is distributed via GitHub Releases
A signed and notarized `.dmg` containing `UNCWORKS.app` SHALL be published to GitHub Releases on each tagged version.

#### Scenario: DMG published on release
- **WHEN** a new version tag is pushed to GitHub
- **THEN** CI builds `UNCWORKS.app`, packages it in a `.dmg`, and uploads it as a release asset

### Requirement: Homebrew cask distribution
A Homebrew cask formula SHALL be maintained so users can install the app with `brew install --cask uncworks`.

#### Scenario: Install via Homebrew cask
- **WHEN** `brew install --cask uncworks/tap/uncworks` is run
- **THEN** the `.dmg` is downloaded, `UNCWORKS.app` is installed to `/Applications`, and it launches successfully

### Requirement: App build is part of CI
The Wails app SHALL be buildable via the existing Dagger CI pipeline on a darwin runner.

#### Scenario: CI builds app artifact
- **WHEN** the CI `build-app` target is triggered
- **THEN** a Wails build produces `UNCWORKS.app` and the build exits successfully
