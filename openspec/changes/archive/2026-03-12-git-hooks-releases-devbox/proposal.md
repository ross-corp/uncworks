## Why

The project has no automated quality gates before code lands -- no pre-commit linting, no commit message enforcement, no breaking-change detection at push time. Commit messages are inconsistent, making it impossible to auto-generate changelogs or compute semver bumps. The `devbox.json` is incomplete -- it's missing tools that are actually used in development (golangci-lint, buf, helm, setup-envtest, grpcurl, lefthook) and tools needed for upcoming changes (temporal-cli). There is no release workflow -- no versioning, no changelog, no GitHub Releases.

## What Changes

- **Lefthook for git hooks**: Install lefthook via devbox, configure `lefthook.yml` with pre-commit (go fmt, golangci-lint, buf lint, TypeScript type checks), commit-msg (commitlint for conventional commits), and pre-push (go test, buf breaking) hooks.
- **Conventional commits enforcement**: Add commitlint with `@commitlint/config-conventional` as a dev dependency. Enforce via lefthook commit-msg hook. All commits MUST follow `type(scope): description` format.
- **Release Please for automated releases**: Add GitHub Actions workflow using `googleapis/release-please-action@v4`. Conventional commits drive semver bumps (fix→patch, feat→minor, `!`→major), auto-generated CHANGELOG.md, and GitHub Releases with tags.
- **golangci-lint for Go quality**: Replace bare `go vet` in lint task with golangci-lint. Configure `.golangci.yml` with sensible defaults (govet, errcheck, staticcheck, unused, gosimple, ineffassign).
- **Complete devbox.json**: Add all missing development dependencies as Nix packages so `devbox shell` gives a fully functional environment with zero manual setup.

## Capabilities

### New Capabilities
- `git-hooks`: Lefthook-managed git hooks for pre-commit, commit-msg, and pre-push quality gates.
- `conventional-commits`: Commitlint enforcement of conventional commit message format across the project.
- `release-automation`: Release Please GitHub Actions workflow for automated versioning, changelog generation, and GitHub Release publishing.
- `devbox-completeness`: Complete devbox.json with all tools needed for local development, testing, infrastructure, and agent workflows.

### Modified Capabilities
- `testing-infra`: Lint task upgraded from `go vet` to golangci-lint. Pre-push hook runs tests automatically.

## Impact

- **`devbox.json`**: Expanded from 8 to ~15+ packages. Adds: lefthook, golangci-lint, buf, grpcurl, helm (kubernetes-helm), temporal-cli, setup-envtest, go-task.
- **New files**: `lefthook.yml`, `.golangci.yml`, `commitlint.config.js`, `release-please-config.json`, `.release-please-manifest.json`, `.github/workflows/release-please.yml`.
- **`Taskfile.yml`**: `lint` task updated to use golangci-lint. New `hooks:install` task.
- **`package.json` (root)**: New root package.json for commitlint dev dependency (or install in existing package).
- **Developer workflow**: After `devbox shell`, developers run `lefthook install` once (or via init_hook). All subsequent commits are validated automatically.
- **CI**: Release Please workflow runs on push to main. Creates/updates a Release PR. Merging the PR creates a GitHub Release with changelog.
