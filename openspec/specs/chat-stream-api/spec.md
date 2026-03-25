## Capability: chat-stream-api

Streaming LLM chat endpoint that accepts conversation history and optional context, proxies to LiteLLM with streaming enabled, and returns an SSE token stream.

---

### Requirement: endpoint-registration

The handler `ChatHandler` must be registered at `POST /api/v1/chat/stream` on the main HTTP mux in the apiserver setup.

**Acceptance Criteria:**
- Route exists and returns 405 for GET requests
- Handler is instantiated with `LiteLLMBaseURL` and `LITELLM_MASTER_KEY` (same env vars as `ClassifyRunHandler`)

---

### Requirement: request-schema

The endpoint accepts JSON body:
```json
{
  "messages": [
    { "role": "user" | "assistant", "content": "string" }
  ],
  "context": {
    "type": "spec" | "run" | "project" | "general",
    "content": "string (raw text, max 8KB after truncation)",
    "label": "string (human-readable, e.g. filename or run ID)"
  }
}
```

**Acceptance Criteria:**
- `messages` is required and must be non-empty; return 400 if missing or empty
- `context` is optional
- `content` in context is truncated server-side to 8192 bytes if longer
- Request body limited to 64KB via `io.LimitReader`

---

### Requirement: system-message-injection

When `context` is provided, the handler prepends a system message before the user messages:

```
You are a helpful assistant for the uncworks AI agent platform.
Current context ({type} — {label}):

{content}

Answer questions about the context above. Be concise and specific.
```

**Acceptance Criteria:**
- System message is injected as `{"role": "system", "content": "..."}` as the first element in the LiteLLM request messages array
- When no context is provided, a minimal system message is still injected: `"You are a helpful assistant for the uncworks AI agent platform."`

---

### Requirement: streaming-proxy

The handler calls LiteLLM's `/v1/chat/completions` with `"stream": true` and pipes the response body directly to the client as `text/event-stream`.

**Acceptance Criteria:**
- Response `Content-Type: text/event-stream` with `Cache-Control: no-cache` and `X-Accel-Buffering: no`
- LiteLLM SSE chunks forwarded as-is to the client (no re-serialization)
- A final `data: [DONE]\n\n` is written when the upstream stream ends
- If LiteLLM returns non-200, write `data: {"error": "..."}\n\n` and close
- HTTP client for streaming has no response timeout (uses request context for cancellation); connect timeout remains 10s

---

### Requirement: no-auth-warning

The endpoint has no authentication (same posture as `/api/v1/classify`). A code comment must note this is intentional and flag it for future auth layer addition.
