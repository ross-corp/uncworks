# UNCWORKS Platform Audit Findings

Audit date: 2026-03-20

## Critical / High

| # | Severity | Component | Issue |
|---|----------|-----------|-------|
| 1 | HIGH | Proto/CRD | Comments in `types.go:66` and `api.proto:81` say `/workspace/src/` but code uses `/workspace/<repo>/` |
| 2 | HIGH | OpenSpec spec | `ui-theming` spec requires 12 shadcn themes but implementation only supports light/dark |
| 3 | HIGH | CI | Doc staleness script fails with 27 stale refs (all Helm values — script is too strict) |

## Medium

| # | Severity | Component | Issue |
|---|----------|-----------|-------|
| 4 | MEDIUM | Web UI | `SpecEditor.tsx` is dead code — not imported anywhere |
| 5 | MEDIUM | Web UI | `NewRunView` hardcodes model tier to "default" — users can't select models |
| 6 | MEDIUM | Extensions | Two separate extension files (`packages/pi-aot-extension` vs `extensions/aot-determinism.ts`) — relationship undocumented |
| 7 | MEDIUM | Extensions | `aot-determinism.ts` can't be type-checked in isolation (deps not in any package.json) |
| 8 | MEDIUM | OpenSpec spec | `sidecar-exec` spec references `/workspace/src/` path (stale) |

## Low

| # | Severity | Component | Issue |
|---|----------|-----------|-------|
| 9 | LOW | Web UI | `use-mobile.tsx` dead hook — not imported anywhere |
| 10 | LOW | Web UI | `use-toast.ts` dead hook — superseded by Toast.tsx provider |
| 11 | LOW | Tests | `internal/server` coverage 21.9% — missing SSE/exec/traces tests |
| 12 | LOW | Tests | `internal/sidecar` coverage 10.2% — missing ExecCommand/loop detection tests |
| 13 | LOW | Tests | `internal/temporal` coverage 9.1% — missing spec-driven pipeline tests |

## Verified OK

| Component | Status |
|-----------|--------|
| All 6 Go binaries compile | PASS |
| golangci-lint (0 issues) | PASS |
| go vet (0 issues) | PASS |
| TypeScript (web, shared, extension) | PASS |
| All REST endpoints match web UI calls | PASS |
| SSE traces/watch endpoint exists | PASS |
| WebSocket exec endpoint exists | PASS |
| ExecCommand uses exact workdir | PASS |
| StartAgent uses resolveWorkDir | PASS |
| SendInput file-based HITL | PASS |
| Loop detection in sidecar | PASS |
| PipelineConfig passthrough (controller) | PASS |
| Hydration workspace layout (/workspace/<repo>/) | PASS |
| Helm values, templates, RBAC | PASS |
| All 5 Dockerfiles correct | PASS |
| CRD YAML matches Go types | PASS |
| Release chart + image workflows | PASS |
| 15 of 18 OpenSpec specs match implementation | PASS |
| brain/embeddings — live code, not dead | PASS |
| E2E tests reference current API | PASS |

## Proposed Follow-Up Changes

1. **fix-stale-docs-and-comments** — Fix `/workspace/src/` references in proto comments, sidecar-exec spec, and doc staleness script threshold
2. **newrun-model-selection** — Add model tier selector to NewRunView
3. **cleanup-dead-ui-code** — Remove SpecEditor.tsx, use-mobile.tsx, use-toast.ts
4. **update-stale-specs** — Update ui-theming spec to match light/dark reality, update sidecar-exec path
5. **increase-test-coverage** — Add tests for server SSE/exec, sidecar ExecCommand, temporal pipeline
6. **document-extension-architecture** — Clarify the two extension files and their relationship
