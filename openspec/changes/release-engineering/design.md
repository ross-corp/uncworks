## Context

Current release flow: release-please manages stable semver tags; existing workflows push Docker images and Helm chart on `v*` tags via Dagger. The Dagger module in `ci/main.go` already has `BuildBinaries()` and `ReleaseBinaries()` but no workflow invokes them. The macOS Wails app (`cmd/uncworks-app/`) has no CI path at all. There is no per-commit traceability artifact.

## Goals / Non-Goals

**Goals:**
- Traceable pre-release tag on every main commit (no GH Release, just a tag)
- `:edge` Docker images refreshed on every main push
- CLI binaries attached to stable GitHub Releases
- Universal macOS DMG attached to stable GitHub Releases
- App icon set from ROSS CORP org avatar

**Non-Goals:**
- Code signing / notarization (requires Apple Developer cert; deferred)
- Windows / Linux desktop packaging
- Pre-release GitHub Releases (too noisy; tags are enough)
- Homebrew tap (separate change once signing is sorted)
- Edge DMG builds (macOS runner cost; stable-tag only)

## Decisions

**Pre-release tag format: `v{next}-pre.{YYYYMMDD}.{sha7}`**
Read `.release-please-manifest.json` → `"."` field → current version (e.g. `0.3.0`) → bump patch → `0.3.1` → tag: `v0.3.1-pre.20260328.abc1234`. This is purely informational — no workflow reacts to these tags, no GH Release is created. The `-pre.` separator makes it a valid semver pre-release field, sortable, no `+` (avoids URL encoding and OCI tag issues).

Alternative considered: moving `edge` git tag — rejected because it's non-immutable and confusing alongside release-please's stable tags.

**Edge images: new `PushEdge()` Dagger function**
Reuses `buildImage()` and the existing registry push logic from `PushImages()` but pins the tag to `:edge` (and `:sha-{sha7}` as an immutable alias). Called from a new `edge` job in `ci.yml` that runs after `ci` on main push only. This keeps edge publishing inside the Dagger module for consistency.

Alternative considered: shell `docker buildx` in the workflow directly — rejected because it bypasses the Dagger build cache and the existing image build logic.

**CLI binaries: Dagger export + `gh release upload`**
`ReleaseBinaries()` already returns a `*dagger.Directory`. The new workflow exports it with `--output ./dist` then calls `gh release upload $TAG dist/*`. This avoids duplicating build logic.

**macOS DMG: native `macos-latest` GHA runner, no Dagger**
Wails requires CGO + macOS SDK + Xcode. Dagger containers can't run macOS binaries. The DMG workflow runs natively:
1. Install Go, Node, wails CLI
2. `cd web && npm ci && npm run build`
3. `cp -r web/dist cmd/uncworks-app/frontend/dist`
4. `wails build -platform darwin/universal -o UNCWORKS.app`
5. `hdiutil create` to produce the DMG
6. `gh release upload` to attach it

`darwin/universal` produces a single fat binary (arm64 + amd64 lipo'd) — the right default for public distribution.

**App icon: fetched at CI time, committed as static asset**
Download `https://avatars.githubusercontent.com/ross-corp` (GitHub org avatar) → resize to 1024×1024 → commit to `cmd/uncworks-app/build/appicon.png`. Wails converts this to `.icns` at build time. Committing it avoids a runtime network fetch in CI.

## Risks / Trade-offs

[Pre-release tag race] CI creates the pre-release tag after release-please runs — if release-please creates a stable tag in the same push, both workflows run and the pre-release tag is redundant → Mitigation: pre-release job is skipped if `GITHUB_REF` is already a `v*` tag.

[macos-latest runner cost] macOS GHA runners are ~10× more expensive than Linux → Mitigation: DMG workflow only triggers on stable `v*` tags, not edge/main.

[Unsigned DMG] Users see "unidentified developer" Gatekeeper warning on first launch → Mitigation: document right-click → Open workaround; revisit with Apple cert later.

[Edge image churn] Pushing `:edge` on every main commit means 6 image pushes per merge — adds ~5min to main CI → Acceptable: keeps edge consumers up to date, and is the expected behaviour for a rolling edge channel.

## Migration Plan

1. Commit app icon to `cmd/uncworks-app/build/appicon.png`
2. Add `PushEdge()` to `ci/main.go`
3. Update `ci.yml` with `pre-release-tag` and `edge` jobs
4. Add `release-binaries.yaml`
5. Add `release-macos.yaml`
6. Verify on next main push (edge jobs) and next release-please tag (binary + DMG jobs)
7. No rollback needed — all new jobs; removing them reverts to current state
