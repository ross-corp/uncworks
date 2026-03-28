## ADDED Requirements

### Requirement: Edge Docker images published on every main push
After CI passes on a push to `main`, the pipeline SHALL build and push all 6 component images (controlplane, init, sidecar, agent, web, bff) to ghcr.io with two tags: `:edge` (mutable, always latest main) and `:sha-{sha7}` (immutable, content-addressable).

#### Scenario: Edge images pushed after successful main CI
- **WHEN** a commit is pushed to `main` and CI passes
- **THEN** all 6 images are pushed to `ghcr.io/uncworks/{component}:edge`
- **THEN** all 6 images are pushed to `ghcr.io/uncworks/{component}:sha-{sha7}`

#### Scenario: Edge publish uses Dagger PushEdge function
- **WHEN** the edge job runs
- **THEN** it calls the Dagger `push-edge` function from `ci/main.go`
- **THEN** build logic is identical to the stable release image builds

#### Scenario: Edge images are not created for stable tag pushes
- **WHEN** the triggering ref is a `v*` tag
- **THEN** the edge job does not run (stable release workflows handle tagging)
