## ADDED Requirements

### Requirement: macOS universal DMG attached to stable GitHub Releases
When a stable `v*` tag is pushed, the pipeline SHALL build the UNCWORKS macOS desktop application as a universal binary (arm64 + amd64), package it as a DMG, and upload it as an asset on the GitHub Release.

#### Scenario: DMG built as universal binary
- **WHEN** a `v*` tag is pushed
- **THEN** `wails build -platform darwin/universal` is run on a `macos-latest` runner
- **THEN** the resulting `.app` bundle contains a fat binary supporting both arm64 and amd64

#### Scenario: DMG asset uploaded to GitHub Release
- **WHEN** the DMG is built successfully
- **THEN** it is uploaded to the GitHub Release as `UNCWORKS-{version}-darwin-universal.dmg`

#### Scenario: Frontend is built and embedded before wails build
- **WHEN** the macOS release workflow runs
- **THEN** `npm ci && npm run build` is run in `web/`
- **THEN** the built `web/dist/` is copied to `cmd/uncworks-app/frontend/dist/` before `wails build`

#### Scenario: DMG is unsigned but distributable
- **WHEN** a user downloads and opens the DMG on macOS
- **THEN** the app launches after right-click → Open (Gatekeeper one-time acknowledgement)
- **THEN** no Apple Developer certificate is required in CI

### Requirement: App icon uses ROSS CORP org avatar
The Wails `.app` bundle SHALL use the ROSS CORP GitHub organisation avatar as its application icon.

#### Scenario: Icon present at wails build time
- **WHEN** `wails build` runs
- **THEN** `cmd/uncworks-app/build/appicon.png` exists as a 1024×1024 PNG
- **THEN** Wails generates `.icns` from it for the bundle
