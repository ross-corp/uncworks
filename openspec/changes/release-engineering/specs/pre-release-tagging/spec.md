## ADDED Requirements

### Requirement: Pre-release git tag on every main push
After CI passes on a push to `main`, the pipeline SHALL create an annotated git tag of the form `v{next}-pre.{YYYYMMDD}.{sha7}` where `next` is the next patch version derived from `.release-please-manifest.json` and `sha7` is the 7-character short SHA of the HEAD commit. No GitHub Release SHALL be created for pre-release tags.

#### Scenario: Tag created after successful CI on main
- **WHEN** a commit is pushed to `main` and CI passes
- **THEN** an annotated git tag matching `v*-pre.*.*` is created pointing at HEAD

#### Scenario: Tag skipped when push is already a stable tag
- **WHEN** the triggering ref is a `v*` tag (stable release)
- **THEN** the pre-release tagging job is skipped

#### Scenario: Tag format is valid semver pre-release
- **WHEN** a pre-release tag is created
- **THEN** the tag string satisfies semver pre-release format: `v{major}.{minor}.{patch}-pre.{YYYYMMDD}.{sha7}`
- **THEN** the tag does NOT contain a `+` character (avoids OCI and URL encoding issues)
