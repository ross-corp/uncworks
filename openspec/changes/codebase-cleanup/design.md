## Context

Codebase audit after the spec-driven-agent-runs change revealed accumulated dead code from the pod→deployment migration, duplicate logic across handlers, hardcoded config that should be environment-driven, and unused frontend components. The knowledge system (brain, embeddings, search) is intentionally kept but not initialized — it's on the roadmap, not dead code.

## Goals / Non-Goals

**Goals:**
- Remove all deprecated Temporal activities and their types/constants
- Make pipeline configuration actually configurable via env vars
- Eliminate code duplication (pod lookup, LogsTab)
- Remove unused frontend components
- Verify spec-driven activity registration works at runtime

**Non-Goals:**
- Removing the knowledge system code (brain, embeddings, search) — it's on the roadmap
- Refactoring the sidecar's `execInSidecar` to use direct exec instead of agent spawning — future optimization
- Changing any external behavior or API contracts

## Decisions

### Decision 1: Delete deprecated activities, don't deprecation-warn

Simply delete `CreateAgentPod`, `CleanupPod`, `CollectLogs`, and `CollectJuniorResults`. No deprecation period needed — they were already marked deprecated and are internal-only (not in the public API).

### Decision 2: Env var fallback pattern for pipeline config

Read env vars with fallback to current defaults:
```go
maxRetries := envOrDefaultInt("AOT_PIPELINE_MAX_RETRIES", 3)
planTimeout := envOrDefaultDuration("AOT_PIPELINE_PLAN_TIMEOUT", 2*time.Minute)
```

### Decision 3: Shared pod lookup as package-level function

Extract `lookupRunningPod(ctx, k8sClient, namespace, runID)` as a package-level function in `internal/server/`, removing the identical methods from FileHandler and ExecHandler.

### Decision 4: LogsTab as standalone component

Extract `LogsTab` from RunDetail.tsx into `web/src/components/LogsTab.tsx`, imported by both RunDetail and DetailPane.

## Risks / Trade-offs

- **Removing registered activities from the worker** — if any in-flight Temporal workflows reference them, those workflows will fail on replay. Mitigated by: all deprecated activities haven't been called since the deployment migration; any stuck workflows would have failed long ago.
- **Changing function signatures for shared lookup** — callers need updating but it's a mechanical refactor within one package.
