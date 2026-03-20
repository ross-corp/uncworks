## Why

All user-facing documentation references "AOT" (Agent Orchestration Tool) and describes the product as "a Cloud Native OS for AI Engineers." The product is actually called **UNCWORKS** and is an **agentic development environment**. Docs reference components that don't exist (SolidJS TUI, PostgreSQL Brain), use stale architecture diagrams, and have no mechanism to stay current. There is no GitHub wiki, and docs rot silently.

## What Changes

- Rename "AOT" to "UNCWORKS" in all user-facing docs (README, docs/, Helm chart notes, UI title bar)
- Rewrite README.md as a concise product overview for UNCWORKS
- Rewrite docs/architecture.md with accurate diagrams reflecting current system (manage/implement agents, workspace layout, Temporal pipeline)
- Restructure docs/ into architecture/, guides/, reference/ sections
- Delete or rewrite stale docs (mockups, outdated references)
- Set up GitHub wiki auto-sync via CI action (docs/ → wiki)
- Add doc staleness detection: CI check that flags docs referencing dead code paths
- Update AGENTS.md and ROADMAP.md
- Update web UI title from "AOT" to "UNCWORKS"

## Capabilities

### New Capabilities
- `docs-rewrite`: Complete rewrite of all user-facing documentation with accurate architecture diagrams and component descriptions
- `wiki-sync`: GitHub Actions workflow that syncs docs/ to the GitHub wiki on push to main
- `doc-staleness`: CI check that detects when docs reference functions, types, or paths that no longer exist

### Modified Capabilities

(none)

## Impact

- **docs/**: All files rewritten or restructured
- **README.md**: Complete rewrite
- **ROADMAP.md**: Updated to reflect current state
- **AGENTS.md**: Updated terminology
- **web/**: Title bar, HTML title tag
- **.github/workflows/**: New wiki-sync and doc-staleness workflows
- **deploy/helm/aot/**: Chart notes updated
- Internal code stays `aot` — this is branding only
