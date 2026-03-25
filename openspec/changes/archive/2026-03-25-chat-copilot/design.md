## Context

The platform already has `callLiteLLM()` in `classify.go` doing non-streaming chat completions, and an SSE infrastructure in `sse.go` for real-time run/trace events. The `handleImproveText` endpoint is effectively a single-turn chat. The frontend uses `apiFetch` with ReadableStream for SSE consumers. LiteLLM supports `"stream": true` in `/v1/chat/completions`, returning OpenAI-compatible SSE chunks (`data: {"choices":[{"delta":{"content":"..."}}]}`).

## Goals / Non-Goals

**Goals:**
- Streaming chat responses (no waiting for full completion)
- Scoped spec chat with file content injected as context
- Global copilot accessible from any page via keyboard shortcut
- Page-aware context (each view registers what it's showing)
- Read-only: chat answers questions and suggests, does not mutate state

**Non-Goals:**
- Agentic tool use / function calling (Phase 2)
- Conversation persistence across browser sessions (localStorage deferred)
- Per-user conversation isolation (no auth yet)
- Model selection in the chat UI (uses `"default"` model)
- Vector search / HydrateContext integration (Phase 2)

## Decisions

**1. SSE streaming over WebSocket**
LiteLLM's streaming is SSE-compatible. The backend pipes the upstream SSE chunks directly to the client rather than accumulating. This reuses the existing SSE pattern and avoids WebSocket upgrade complexity. The `ClassifyRunHandler.HTTPClient` timeout (10s) is not suitable — the chat handler gets its own client with no response timeout (only request timeout).

**2. Single endpoint for both scoped and global chat**
`POST /api/v1/chat/stream` accepts `{ messages: [...], context?: { type, content, ... } }`. The context is injected as a system message prefix — the endpoint doesn't know or care whether the caller is ChatSheet or CopilotPanel. Simpler backend, flexible frontend.

**3. Context injection as a system message**
Rather than a RAG pipeline, context (spec content, run status, project name) is prepended as a system message: `"You are a helpful assistant for the uncworks platform. Current context:\n\n<context>"`. This is fast, stateless, and sufficient for v1. The system message is constructed server-side so the frontend never needs to manage prompt templates.

**4. `useCopilotContext()` — React context registration**
A React context provider at Layout level holds `{ type, data }`. Views call `useCopilotContext(data)` on mount (and when their relevant state changes). The CopilotPanel reads from this context to seed chat. This is the same pattern used by browser devtools extensions and IDE plugins — views push context up, the panel reads it.

**5. Sheet over floating bubble for scoped chat**
shadcn `Sheet` (side panel) keeps the spec editor visible while chatting. The global CopilotPanel uses a `Dialog` (full-ish overlay) triggered by `⌘K` / `Ctrl+K` — matching the command-palette shortcut users are already conditioned to.

**6. Streaming frontend rendering**
The frontend reads the SSE stream using `fetch` + `ReadableStream` (same pattern as the existing thinking/trace polling). Each chunk is appended to the last assistant message in state. No external streaming library needed.

## Risks / Trade-offs

- **LiteLLM timeout**: Streaming responses may be slow for long context. Mitigation: set a 60s read deadline server-side; show a typing indicator client-side.
- **Context size**: Large spec files could exceed model context windows. Mitigation: truncate context payload to 8KB server-side before injecting.
- **No auth on chat endpoint**: Anyone on the network can use the LLM via this endpoint. Mitigation: same posture as classify/improve-text — acceptable for internal tooling, flag for future auth layer.
- **CopilotPanel ⌘K conflicts**: May conflict with browser or OS shortcuts. Mitigation: only intercept when focus is not in a text input.

## Migration Plan

1. Deploy backend with new `/api/v1/chat/stream` endpoint — additive, no breaking changes
2. Deploy frontend with ChatSheet + CopilotPanel — behind existing "Chat about this spec" button and new global shortcut
3. No rollback complexity — removing the button/shortcut reverts the UX

## Open Questions

- Should the chat system message include the current user's recent run history (last 5 runs) as additional context? Could be useful for the global copilot without full vector search.
- Should ⌘K open a combined command palette (navigation + chat) or go straight to chat?
