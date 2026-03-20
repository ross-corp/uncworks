## Why

CI has been failing silently for a long time. The GitHub Actions workflow has several issues: fragile envtest setup that swallows errors, missing npm installs for TypeScript checks, web UI not checked at all, and contract tests that depend on envtest but don't set it up correctly. Every commit to main should pass CI reliably.

## What Changes

- Fix envtest setup in CI (proper KUBEBUILDER_ASSETS path)
- Add npm install for web UI before TypeScript check
- Add web UI TypeScript check to CI (currently only shared + extension are checked)
- Fix contract test envtest setup
- Add Go build verification step (catch compile errors early)
- Ensure buf commands handle missing buf gracefully on push (not just PR)
- Remove fragile patterns (piping envtest output to GITHUB_ENV)
- Add caching for Go modules and npm packages

## Capabilities

### New Capabilities
- `reliable-ci`: CI pipeline that passes on every commit without false failures

### Modified Capabilities

(none)

## Impact

- `.github/workflows/ci.yml`: Rewrite with proper setup steps
- No code changes needed — this is infrastructure only
