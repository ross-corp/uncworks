## Context

A 6-agent analysis surfaced 40+ actionable issues across all major subsystems. The most urgent are two security vulnerabilities in `internal/hydration` (path traversal, token injection), a Temporal determinism violation, and a cluster of silent-failure bugs that make the system hard to debug. This design covers all critical and high severity items as a single hardening pass.

## Goals / Non-Goals

**Goals:**
- Eliminate all CRITICAL severity findings (path traversal, token injection, determinism, silent data loss)
- Fix HIGH severity reliability issues (requeue on error, stale polling closures, OOM list endpoints)
- Require no schema migrations or breaking API changes
- Keep each fix self-contained and independently testable

**Non-Goals:**
- Medium/low severity issues (out of scope for this pass)
- Adding authentication to all server endpoints (larger scope; tracked separately)
- Implementing token counting for LLM calls
- Frontend pagination or virtualization

## Decisions

### 1. Path validation in hydration: allowlist approach

**Decision:** Reject any `Repository.Path` that is absolute, contains `..`, or when `filepath.Clean`'d starts with `..`. Accept only relative, clean paths.

**Alternatives:**
- Symlink-following real path check: requires filesystem access before clone, not available in config validation
- Allowlist chars only: too restrictive for legitimate nested paths like `services/api`

**Rationale:** Simple, testable, covers the attack vectors without over-constraining valid input.

### 2. URL injection fix: parse-then-validate

**Decision:** Use `url.Parse()` in `injectTokenInURL`, validate `u.Scheme == "https"` and `u.Host` matches a configured allowlist (defaulting to `github.com`), then set `u.User`. Return the original URL unchanged for SSH.

**Alternatives:**
- String prefix check (current): fragile, doesn't handle `@`-embedded hostnames
- Strip all credentials before injecting: correct but adds complexity

**Rationale:** `url.Parse` is the canonical approach; host validation prevents token exfiltration to crafted URLs.

### 3. Webhook auth: fail-closed

**Decision:** If `WebhookSecret` is empty string at server startup, log a warning and return 401 for all webhook requests (rather than accepting them without validation).

**Alternatives:**
- Panic on startup if secret missing: too aggressive, breaks existing deployments without a secret configured
- Log warning but allow through: maintains current vulnerable behavior

**Rationale:** Fail-closed is the only secure default. Deployments not using webhooks are unaffected.

### 4. Temporal determinism: selector-based background work

**Decision:** The `workflow.Go()` call in the workflow starts a goroutine that reads from a channel — this is a Temporal SDK violation (goroutines are not replayed). Replace with a `workflow.Go`-free selector loop: use `workflow.NewSelector`, add a receive on the signal channel, and handle it in the main workflow loop.

**Alternatives:**
- Child workflow: heavier, adds orchestration overhead for what is lightweight signal handling
- Activity: doesn't fit the signal-response pattern

**Rationale:** Selector-based is the idiomatic Temporal approach for concurrent signal handling without goroutines.

### 5. Controller requeue: tiered backoff

**Decision:** On transient errors (network, conflict, not-found of a dependency), return `ctrl.Result{RequeueAfter: 10 * time.Second}`. On `status.Update` failure, log the error and return it (controller-runtime will requeue automatically).

**Alternatives:**
- Exponential backoff per resource: requires state; overkill given Kubernetes already does this at controller-runtime level
- Return error immediately: controller-runtime does requeue on error, so returning the error is equivalent but makes the failure visible in logs

**Rationale:** The key fix is stopping silent `nil` returns on real errors.

### 6. Frontend polling: cancelled flag pattern

**Decision:** Standardize all `useEffect` polling hooks on the existing pattern already used in `RunDetailView.tsx` (lines 180–205): `let cancelled = false` at effect start, `cancelled = true` in cleanup, check `if (!cancelled)` before each `setState`.

**Alternatives:**
- AbortController: works for fetch cancellation but doesn't prevent setState after unmount
- React Query / SWR: correct long-term solution but out of scope for this hardening pass

**Rationale:** Minimal change, matches existing proven pattern, can be applied mechanically across 13 views.

### 7. List endpoint cap: middleware-level

**Decision:** Add a `cap` helper in `internal/server` that truncates any slice to max 500 before returning. Apply in each `handleList*` function.

**Alternatives:**
- Kubernetes-native pagination (continue token): requires Kubernetes API support; most endpoints already do full list
- Per-endpoint configurable limits: adds complexity; 500 is sufficient for all current use cases

**Rationale:** Simple, immediate protection against OOM. Can be replaced with proper pagination later.

### 8. Phase constants: single file per group

**Decision:** Add `api/v1alpha1/phases.go` with typed string constants for all phase/status values used in CRD status fields. No new types — keep as `string` to maintain JSON compatibility; just named constants.

**Rationale:** Eliminates magic strings, enables grep-based usage tracking, catches typos at compile time.

### 9. ScheduleSpec mutual exclusivity: CEL XValidation

**Decision:** Add `+kubebuilder:validation:XValidation` rule directly on `ScheduleSpec` using CEL: `!(has(self.chainRef) && has(self.templateRef)) || (self.chainRef == "" || self.templateRef == "")`.

**Alternatives:**
- Server-side validation in handler (already done): correct but not enforced at API level
- Webhook: more infrastructure; CEL is sufficient

**Rationale:** CEL validation is declarative, enforced by API server before any controller code runs.

## Risks / Trade-offs

- **Webhook fail-closed breaks existing unprotected deployments** → Document in change notes; operators must set `WEBHOOK_SECRET` env var before upgrading
- **Path validation may reject valid existing paths with unusual chars** → Conservative allowlist (only `..` and absolute path checks) minimizes false positives
- **Temporal workflow change requires workflow version bump** → Use `workflow.GetVersion` to handle in-flight workflow instances during deploy; or drain in-flight runs before deploying
- **`cancelled` flag doesn't cancel in-flight fetches** → Fetch may complete after unmount but state update is suppressed; acceptable for this pass

## Migration Plan

1. Deploy backend changes (hydration, server, controller, temporal) — no schema migration needed
2. Deploy frontend bundle — polling guards are additive, no behavioral change for users
3. Set `WEBHOOK_SECRET` on any deployments using webhooks (one-time operator action)
4. Re-run CRD apply to pick up CEL validation on ScheduleSpec (requires `kubectl apply -f crds/`)

**Rollback:** All changes are independently reversible. No database schema changes.

## Open Questions

- Should embedding failures in `knowledge_activities.go` be fatal (return error) or soft-fail with a warning? Current proposal: return error so Temporal can retry; depends on how critical knowledge hydration is to run quality.
- Temporal workflow version: need to check current workflow version numbers before adding `GetVersion` gate.
