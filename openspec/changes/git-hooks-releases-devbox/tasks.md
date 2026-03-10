## 1. Devbox Completeness

- [x] 1.1 Add `lefthook@latest` to devbox.json packages
- [x] 1.2 Add `golangci-lint@latest` to devbox.json packages
- [x] 1.3 Add `buf@latest` to devbox.json packages
- [x] 1.4 Add `grpcurl@latest` to devbox.json packages
- [x] 1.5 Add `kubernetes-helm@latest` to devbox.json packages
- [x] 1.6 Add `temporal-cli@latest` to devbox.json packages
- [x] 1.7 Add `go-task@latest` to devbox.json packages
- [x] 1.8 Add `setup-envtest@latest` to devbox.json packages
- [x] 1.9 Add `lefthook install` to devbox.json init_hook array
- [x] 1.10 Run `devbox install` to generate updated devbox.lock
- [x] 1.11 Verify all tools are available on PATH after `devbox shell`

## 2. golangci-lint Configuration

- [x] 2.1 Create `.golangci.yml` at project root with linters: govet, errcheck, staticcheck, unused, gosimple, ineffassign, gocritic, gofmt, misspell
- [x] 2.2 Configure timeout (3 minutes), Go version, and exclude patterns for generated code (`gen/go/`)
- [x] 2.3 Update `Taskfile.yml`: change `lint` task Go command from `go vet ./...` to `golangci-lint run ./...`
- [x] 2.4 Run `golangci-lint run ./...` and fix any violations
- [x] 2.5 Update AGENTS.md to document golangci-lint usage

## 3. Conventional Commits (commitlint)

- [x] 3.1 Create root-level `package.json` with `@commitlint/cli` and `@commitlint/config-conventional` as devDependencies
- [x] 3.2 Create `commitlint.config.js` extending `@commitlint/config-conventional`
- [x] 3.3 Run `npm install` at root to install commitlint
- [x] 3.4 Add root `node_modules/` is already in .gitignore (verify)
- [x] 3.5 Test commitlint: `echo "bad message" | npx commitlint` should fail; `echo "feat: good message" | npx commitlint` should pass

## 4. Lefthook Git Hooks

- [x] 4.1 Create `lefthook.yml` at project root
- [x] 4.2 Configure pre-commit: `go fmt` with `stage_fixed: true` on `*.go` glob
- [x] 4.3 Configure pre-commit: `golangci-lint run --new-from-rev=HEAD` on `*.go` glob
- [x] 4.4 Configure pre-commit: `buf lint` on `*.proto` glob
- [x] 4.5 Configure pre-commit: `tsc --noEmit` for web and packages (parallel)
- [x] 4.6 Configure commit-msg: `npx commitlint --edit {1}`
- [x] 4.7 Configure pre-push: `go test ./api/... ./internal/... -count=1`
- [x] 4.8 Configure pre-push: `buf breaking --against '.git#branch=main'`
- [x] 4.9 Run `lefthook install` and verify hooks are in `.git/hooks/`
- [x] 4.10 Test full flow: make a commit with bad message (should fail), fix message (should pass), push (tests should run)
- [x] 4.11 Add `task hooks:install` target to Taskfile.yml as alias for `lefthook install`

## 5. Release Please

- [ ] 5.1 Create `release-please-config.json` at project root: release-type `go`, package `.`, changelog-sections configuration
- [ ] 5.2 Create `.release-please-manifest.json` at project root with initial version `{"." : "0.1.0"}`
- [ ] 5.3 Create `.github/workflows/release-please.yml`: trigger on push to main, use `googleapis/release-please-action@v4`
- [ ] 5.4 Configure workflow permissions: contents:write, pull-requests:write
- [ ] 5.5 Document RELEASE_PLEASE_TOKEN secret requirement in deploy/ or docs/
- [ ] 5.6 Test by pushing a conventional commit to main and verifying Release PR creation

## 6. Verification

- [ ] 6.1 Verify `devbox shell` provides all declared tools
- [ ] 6.2 Verify `lefthook install` runs automatically on shell entry
- [ ] 6.3 Verify pre-commit hook catches Go lint violations
- [ ] 6.4 Verify commit-msg hook rejects non-conventional messages
- [ ] 6.5 Verify pre-push hook runs tests and buf breaking
- [ ] 6.6 Verify `task lint` uses golangci-lint
- [ ] 6.7 Verify release-please workflow file is valid YAML
