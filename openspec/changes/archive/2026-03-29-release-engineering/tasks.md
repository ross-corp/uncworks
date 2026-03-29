## 1. App Icon

- [x] 1.1 Create `cmd/uncworks-app/build/` directory
- [x] 1.2 Download ROSS CORP GitHub org avatar (`https://avatars.githubusercontent.com/ross-corp`) and save as `cmd/uncworks-app/build/appicon.png` at 1024×1024

## 2. Dagger: PushEdge function

- [x] 2.1 Add `PushEdge(ctx, source, registryUser, registryPass)` function to `ci/main.go` that builds all 6 images and pushes `:edge` and `:sha-{sha7}` tags
- [x] 2.2 Derive the short SHA inside `PushEdge` using `git rev-parse --short HEAD` via a Dagger exec
- [x] 2.3 Verify `PushEdge` reuses `buildImage()` and the existing registry auth pattern from `PushImages()`

## 3. CI workflow: pre-release tag + edge jobs

- [x] 3.1 Add `pre-release-tag` job to `.github/workflows/ci.yml` — runs after `ci`, main push only, creates tag `v{next}-pre.{YYYYMMDD}.{sha7}` using a shell script that reads `.release-please-manifest.json`
- [x] 3.2 Add `edge` job to `.github/workflows/ci.yml` — runs after `ci`, main push only, calls `dagger call push-edge --source . --registry-user ... --registry-pass ...`
- [x] 3.3 Add `contents: write` and `packages: write` permissions to the new jobs
- [x] 3.4 Add `if: github.event_name == 'push' && github.ref == 'refs/heads/main'` condition to both jobs

## 4. Release workflow: CLI binaries

- [x] 4.1 Create `.github/workflows/release-binaries.yaml` triggered on `v*` tag push
- [x] 4.2 Add step to extract version from tag (`${GITHUB_REF_NAME#v}`)
- [x] 4.3 Add Dagger step calling `release-binaries --source . --version {version} --output ./dist`
- [x] 4.4 Add step uploading all files in `./dist/` to the GitHub Release via `gh release upload $GITHUB_REF_NAME dist/*`

## 5. Release workflow: macOS DMG

- [x] 5.1 Create `.github/workflows/release-macos.yaml` triggered on `v*` tag push, running on `macos-latest`
- [x] 5.2 Add steps to install Go (1.22+), Node (22), and wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- [x] 5.3 Add step to build the web frontend: `cd web && npm ci && npm run build`
- [x] 5.4 Add step to copy `web/dist/` → `cmd/uncworks-app/frontend/dist/`
- [x] 5.5 Add step to run `wails build -platform darwin/universal` from `cmd/uncworks-app/`
- [x] 5.6 Add step to create DMG: `hdiutil create -volname UNCWORKS -srcfolder "build/bin/UNCWORKS.app" -ov -format UDZO "UNCWORKS-${VERSION}-darwin-universal.dmg"`
- [x] 5.7 Add step uploading the DMG to the GitHub Release via `gh release upload`

## 6. Verification

- [ ] 6.1 Trigger a test push to main and confirm pre-release tag is created
- [ ] 6.2 Confirm `:edge` images appear in ghcr.io after main push
- [ ] 6.3 On next release-please stable tag, confirm CLI binaries appear as GitHub Release assets
- [ ] 6.4 On next release-please stable tag, confirm DMG appears as GitHub Release asset
- [ ] 6.5 Confirm `wails build` produces a working `.app` with the ROSS CORP icon
