## Context

The apiserver is a plain `net/http` mux with no rate limiting. The BFF has a rudimentary IP bucket (`internal/bff/middleware.go`) but it's unbounded and covers only BFF traffic. LLM-backed endpoints (`/api/v1/classify`, `/api/v1/chat/stream`) are the highest-risk: a single client looping requests can saturate Ollama throughput and block all other users.

The `go-concurrency-fixes` change will fix the unbounded `sync.Map` bug in the BFF middleware — this design builds on that foundation rather than duplicating it.

## Goals / Non-Goals

**Goals:**
- Per-IP sliding-window rate limiting on all apiserver endpoints
- Tighter limits specifically on LLM endpoints (classify, chat)
- Tighter limits on webhook endpoint (abuse prevention)
- Standard HTTP 429 + `Retry-After` + `X-RateLimit-*` headers
- Helm-configurable limits (RPS, burst, TTL)
- TTL-evicting IP store (no unbounded memory growth)

**Non-Goals:**
- Per-user or per-API-key rate limiting (requires auth, future milestone)
- Distributed rate limiting across replicas (in-process only for now)
- Token bucket on Ollama/LiteLLM directly (out of scope)

## Decisions

### Sliding window vs token bucket
**Choice:** Sliding window counter (per IP, per 1-second window).
**Why:** Simpler implementation, no "burst debt" accumulation, standard behaviour users expect from API rate limiters. Token bucket is better for bursty-but-average workloads; our LLM endpoints should be uniformly throttled.

### In-process vs external (Redis)
**Choice:** In-process `sync.Map` with TTL eviction.
**Why:** Platform is single-replica in dev; adding Redis is significant infra overhead. Noted as non-goal. If/when multi-replica, migrate to Redis.

### Library vs hand-rolled
**Choice:** `golang.org/x/time/rate` (token limiter per IP, leaky bucket semantics).
**Why:** Already in Go standard library extended packages, battle-tested, zero new dependencies. Per-IP limiters stored in a TTL map.

### TTL eviction strategy
**Choice:** Background goroutine sweeps the IP map every 5 minutes, evicts entries not seen in `ttl` duration (default 10 minutes).
**Why:** Same pattern being adopted in `go-concurrency-fixes` for BFF middleware. Consistent, simple, no external deps.

### Config surface
**Choice:** Helm values `apiserver.rateLimit.{enabled, rps, burst, llmRps, llmBurst, webhookRps, ttlMinutes}`.
**Why:** Operators need to tune limits per deployment. Dev default: generous limits (100 RPS global, 10 RPS LLM). Prod: tighter.

## Risks / Trade-offs

- **Single-replica only** → Distributed deploys would need Redis. Acceptable for now; document as known limitation in Helm values.
- **IP spoofing via X-Forwarded-For** → Use `r.RemoteAddr` by default; add opt-in `trustProxy` flag that reads `X-Forwarded-For` when behind a known proxy (ingress controller).
- **Legitimate burst traffic** → `burst` parameter allows short spikes. Set conservatively in defaults.
- **golang.org/x/time/rate is per-limiter** → Need one limiter per IP, stored in the TTL map. Memory per limiter is ~100 bytes; 10k active IPs = ~1 MB. Negligible.

## Migration Plan

1. Deploy with `apiserver.rateLimit.enabled: false` (default) — no behaviour change
2. Enable in dev, validate limits don't interfere with normal usage
3. Set `enabled: true` in prod values with conservative limits
4. Monitor 429 rate in logs; tune as needed
5. Rollback: set `enabled: false` in Helm, rolling restart
