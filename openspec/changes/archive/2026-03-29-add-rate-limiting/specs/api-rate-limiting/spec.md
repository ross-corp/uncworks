## ADDED Requirements

### Requirement: Per-IP rate limiting on apiserver endpoints
The apiserver SHALL enforce per-source-IP rate limits on all HTTP endpoints using a sliding window algorithm. When a client exceeds the configured requests-per-second limit, the server SHALL respond with HTTP 429 and a `Retry-After` header indicating when the client may retry.

#### Scenario: Request within rate limit
- **WHEN** a client sends requests below the configured RPS threshold
- **THEN** requests are processed normally with no rate-limit headers added

#### Scenario: Request exceeds rate limit
- **WHEN** a client exceeds the configured RPS threshold
- **THEN** the server returns HTTP 429 with `Retry-After: <seconds>` and `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers

#### Scenario: Rate limiting disabled
- **WHEN** `apiserver.rateLimit.enabled` is false in Helm values
- **THEN** all requests are passed through with no rate limiting applied

### Requirement: Tighter rate limits on LLM endpoints
The classify (`/api/v1/classify`) and chat stream (`/api/v1/chat/stream`) endpoints SHALL enforce a separate, lower per-IP rate limit to protect LLM gateway throughput.

#### Scenario: LLM endpoint rate limited independently
- **WHEN** a client exceeds the LLM-specific RPS threshold on `/api/v1/classify` or `/api/v1/chat/stream`
- **THEN** the server returns HTTP 429 even if the global endpoint rate limit has not been reached

#### Scenario: LLM limit configurable independently
- **WHEN** `apiserver.rateLimit.llmRps` is set in Helm values
- **THEN** that value governs LLM endpoint limits independently of `apiserver.rateLimit.rps`

### Requirement: Tighter rate limits on webhook endpoint
The webhook endpoint (`/api/v1/webhook`) SHALL enforce a separate, tighter per-IP rate limit.

#### Scenario: Webhook endpoint rate limited
- **WHEN** a source IP sends more requests than `apiserver.rateLimit.webhookRps` per second
- **THEN** the server returns HTTP 429

### Requirement: TTL-evicting IP limiter store
The in-process IP limiter map SHALL evict entries for IPs that have not been seen within the configured TTL to prevent unbounded memory growth.

#### Scenario: Inactive IP evicted
- **WHEN** an IP has not sent a request within `apiserver.rateLimit.ttlMinutes`
- **THEN** its limiter entry is removed from memory on the next sweep cycle

#### Scenario: Active IP retained
- **WHEN** an IP continues to send requests
- **THEN** its limiter entry and current window state are preserved across sweep cycles
