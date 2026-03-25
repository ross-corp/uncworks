## 1. Rate Limiter Middleware

- [x] 1.1 Create `internal/server/ratelimit.go` — `RateLimiter` struct with `sync.Map` of per-IP `rate.Limiter` entries and last-seen timestamps
- [x] 1.2 Add TTL sweep goroutine started at construction (5-minute interval, evicts entries not seen in `ttlMinutes`)
- [x] 1.3 Implement `Allow(ip string) bool` — creates limiter on first use, updates last-seen, returns `rate.Limiter.Allow()`
- [x] 1.4 Implement `Remaining(ip string) int` — returns `int(limiter.Tokens())` for response headers
- [x] 1.5 Add `RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler` — wraps handler, returns 429 with `Retry-After` + `X-RateLimit-*` headers on deny
- [x] 1.6 Add `RateLimiterConfig` struct: `Enabled bool`, `RPS float64`, `Burst int`, `TTLMinutes int`

## 2. LLM and Webhook Specific Limiters

- [x] 2.1 Instantiate separate `RateLimiter` instances for global, LLM endpoints, and webhook in `cmd/apiserver/main.go` (or server constructor)
- [x] 2.2 Apply global limiter middleware to root mux
- [x] 2.3 Apply LLM limiter middleware specifically to `/api/v1/classify` and `/api/v1/chat/stream`
- [x] 2.4 Apply webhook limiter middleware to `/api/v1/webhook`

## 3. Helm Configuration

- [x] 3.1 Add `apiserver.rateLimit` section to `deploy/helm/aot/values.yaml`: `enabled: false`, `rps: 100`, `burst: 20`, `llmRps: 10`, `llmBurst: 5`, `webhookRps: 5`, `webhookBurst: 2`, `ttlMinutes: 10`
- [x] 3.2 Pass rate limit values as env vars in `deploy/helm/aot/templates/apiserver.yaml`
- [x] 3.3 Read env vars in server startup and construct `RateLimiterConfig` instances
- [x] 3.4 Update `aot-local/dev-values.yaml` with permissive dev limits (enabled: false by default)

## 4. Trusted Proxy Support

- [x] 4.1 Add `apiserver.rateLimit.trustProxy: false` Helm value
- [x] 4.2 When `trustProxy: true`, extract client IP from `X-Forwarded-For` first header; fall back to `r.RemoteAddr`
- [x] 4.3 Strip port from `r.RemoteAddr` for consistent IP keying

## 5. Tests

- [x] 5.1 Unit test `RateLimiter.Allow()` — verify allows up to burst, then denies, then recovers after window
- [x] 5.2 Unit test TTL eviction — advance mock clock, verify entry removed after TTL
- [x] 5.3 Unit test middleware handler — verify 429 + headers when limiter denies
- [x] 5.4 Integration test: `TestChatHandler` series — verify rate limiting doesn't break normal flow when disabled
- [x] 5.5 Integration test: verify 429 returned when limit exceeded (use tiny RPS in test config)
