## Why

The project ships Docker images and a Helm chart on stable tags but has no mechanism to distribute CLI binaries or the macOS desktop app, and every main push leaves no traceable artifact. Contributors and users have no way to grab a pre-built binary or test an unreleased build without cloning and building locally.

## What Changes

- Every push to `main` creates a pre-release git tag (`v{next}-pre.{YYYYMMDD}.{sha7}`) for traceability
- Every push to `main` republishes all 6 Docker images with an `:edge` tag in ghcr.io
- Stable `v*` tags trigger upload of 4 CLI binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) as GitHub Release assets
- Stable `v*` tags trigger a macOS DMG build (`darwin/universal`) and upload to GitHub Release
- A new Dagger function `PushEdge()` handles the edge container workflow
- App icon set to ROSS CORP GitHub org avatar for the Wails `.app` bundle

## Capabilities

### New Capabilities

- `pre-release-tagging`: Git tag every main push with a structured pre-release version string derived from the release-please manifest
- `edge-images`: Publish `:edge` Docker images to ghcr.io on every main push
- `cli-release-assets`: Build and upload cross-platform CLI binaries to GitHub Releases on stable tags
- `macos-dmg-release`: Build a universal macOS DMG via Wails and upload to GitHub Releases on stable tags

### Modified Capabilities

- `ci-pipeline`: The main CI workflow gains an edge-publish job that runs after CI passes on main

## Impact

- `.github/workflows/ci.yml`: new `edge` job after `ci` job (main push only)
- `.github/workflows/release-binaries.yaml`: new workflow (v* tags)
- `.github/workflows/release-macos.yaml`: new workflow (v* tags, macos-latest runner)
- `ci/main.go`: new `PushEdge()` Dagger function
- `cmd/uncworks-app/build/appicon.png`: new file (app icon)
- No changes to existing `release-images.yaml` or `release-chart.yaml`
- No changes to release-please config
