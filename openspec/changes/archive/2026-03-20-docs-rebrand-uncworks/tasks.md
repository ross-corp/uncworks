## 1. Branding Rename

- [x] 1.1 Rewrite README.md: title "UNCWORKS", subtitle "An agentic development environment", accurate feature list and architecture overview
- [x] 1.2 Update web UI HTML title tag from "AOT — UNCWERKS" to "UNCWORKS"
- [x] 1.3 Update Helm chart NOTES.txt from "AOT" to "UNCWORKS"
- [x] 1.4 Update ROADMAP.md header and content
- [x] 1.5 Update AGENTS.md terminology

## 2. Documentation Rewrite

- [x] 2.1 Create docs/README.md (wiki Home page) — product overview
- [x] 2.2 Create docs/getting-started.md — k0s local dev quickstart
- [x] 2.3 Create docs/architecture/overview.md — system diagram with all components
- [x] 2.4 Create docs/architecture/control-plane.md — API server, controller, Temporal worker
- [x] 2.5 Create docs/architecture/agent-pods.md — sidecar, pi, determinism extension, workspace
- [x] 2.6 Create docs/architecture/pipeline.md — Plan/Execute/Verify with manage/implement roles
- [x] 2.7 Create docs/architecture/workspace.md — hydration, worktree layout, OpenSpec
- [x] 2.8 Create docs/guides/creating-runs.md — UI walkthrough
- [x] 2.9 Create docs/guides/spec-driven.md — spec-driven pipeline guide
- [x] 2.10 Create docs/guides/models.md — LiteLLM, Ollama, OpenRouter config
- [x] 2.11 Create docs/reference/api.md — ConnectRPC endpoints
- [x] 2.12 Create docs/reference/crd.md — AgentRun CRD spec
- [x] 2.13 Create docs/reference/extension.md — determinism extension reference
- [x] 2.14 Create docs/reference/helm-values.md — Helm values reference
- [x] 2.15 Create docs/contributing/development.md — local dev setup
- [x] 2.16 Create docs/contributing/testing.md — test infrastructure
- [x] 2.17 Delete stale docs (mockups/, outdated files)

## 3. Wiki Sync

- [x] 3.1 Create .github/workflows/wiki-sync.yml — sync docs/ to wiki on push
- [x] 3.2 Generate _Sidebar.md from directory structure
- [x] 3.3 Map docs/README.md → Home.md, flatten paths for wiki page names

## 4. Doc Staleness Detection

- [x] 4.1 Create scripts/check-doc-staleness.sh — scan docs for stale code references
- [x] 4.2 Create .github/workflows/doc-staleness.yml — run on PRs touching source
