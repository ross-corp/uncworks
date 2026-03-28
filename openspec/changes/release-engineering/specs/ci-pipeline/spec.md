## MODIFIED Requirements

### Requirement: Main branch CI workflow includes edge publishing
The `ci.yml` workflow SHALL include two additional jobs that run after the `ci` job passes on pushes to `main`: `pre-release-tag` (creates a traceability tag) and `edge` (publishes `:edge` Docker images). Both jobs SHALL be skipped if the push is a tag push.

#### Scenario: Edge jobs run after CI on main push
- **WHEN** a commit is pushed to `main` and the `ci` job passes
- **THEN** both `pre-release-tag` and `edge` jobs run
- **THEN** both jobs are listed under `needs: [ci]` in the workflow

#### Scenario: Edge jobs require write permissions
- **WHEN** the `edge` job runs
- **THEN** the job has `packages: write` permission to push to ghcr.io
- **THEN** the job has `contents: write` permission to push git tags

#### Scenario: Edge jobs are conditional on branch (not tag)
- **WHEN** a `v*` tag is pushed to the repository
- **THEN** the `pre-release-tag` and `edge` jobs do NOT run
- **THEN** `release-images.yaml` and `release-chart.yaml` handle the stable tag instead
