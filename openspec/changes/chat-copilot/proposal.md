## Why

The platform has a full LLM gateway (LiteLLM) and spec/run infrastructure but no conversational interface — users must leave the UI to get AI help with specs, understand run failures, or craft better prompts. An integrated chat copilot eliminates that friction and surfaces platform knowledge directly in context.

## What Changes

- New streaming chat endpoint `POST /api/v1/chat/stream` (SSE) that accepts conversation history + a context blob and proxies to LiteLLM with `"stream": true`
- New `ChatSheet` frontend component: a shadcn Sheet panel with message list, streaming rendering, and a text input
- "Chat about this spec" button in `ProjectDetailView` opens `ChatSheet` pre-loaded with spec content as context
- Global `CopilotPanel` component: a `⌘K`-triggered floating chat accessible from any view, with page-aware context injection via `useCopilotContext()` hook
- `Layout.tsx` hosts the global panel so it persists across navigation

## Capabilities

### New Capabilities
- `chat-stream-api`: Streaming `/api/v1/chat/stream` endpoint — accepts `messages[]` + optional `context` object, returns SSE token stream from LiteLLM
- `chat-sheet`: Scoped `ChatSheet` component for spec-context chat — opens from spec editor, pre-seeds context with file content + project name
- `copilot-panel`: Global ambient chat panel triggered by `⌘K` — page-aware context via `useCopilotContext()` hook registered by each view

### Modified Capabilities

## Impact

- New file: `internal/server/chat.go` (streaming handler)
- New files: `web/src/components/ChatSheet.tsx`, `web/src/components/CopilotPanel.tsx`, `web/src/hooks/useCopilotContext.tsx`
- Modified: `web/src/views/ProjectDetailView.tsx` (wire ChatSheet to button)
- Modified: `web/src/views/Layout.tsx` (mount CopilotPanel)
- Modified: `cmd/apiserver/main.go` or equivalent (register chat handler)
- No schema changes, no new CRDs, no database migrations
- Depends on: `LITELLM_BASE_URL` and `LITELLM_MASTER_KEY` env vars (already required by classify endpoint)
