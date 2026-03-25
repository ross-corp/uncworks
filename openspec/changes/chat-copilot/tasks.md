## 1. Backend — Chat Streaming Endpoint

- [x] 1.1 Create `internal/server/chat.go` with `ChatHandler` struct (fields: `LiteLLMBaseURL string`, `HTTPClient *http.Client` with no response timeout)
- [x] 1.2 Implement request parsing: decode JSON body (64KB limit), validate `messages` non-empty, truncate `context.content` to 8192 bytes
- [x] 1.3 Build system message from context: `"You are a helpful assistant for the uncworks AI agent platform.\nCurrent context ({type} — {label}):\n\n{content}\n\nAnswer questions about the context above. Be concise and specific."` (or minimal system message if no context)
- [x] 1.4 Call LiteLLM `/v1/chat/completions` with `"stream": true`, inject `Authorization` header from `LITELLM_MASTER_KEY` env var
- [x] 1.5 Proxy SSE stream to client: set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`; pipe upstream body line by line; write `data: [DONE]\n\n` on completion; write `data: {"error":"..."}\n\n` on upstream error
- [x] 1.6 Register `POST /api/v1/chat/stream` in the apiserver mux setup (find where `ClassifyRunHandler` is registered and add `ChatHandler` next to it)
- [x] 1.7 Add a unit test `TestChatHandler_MissingMessages_Returns400` and `TestChatHandler_ContextTruncation` in `internal/server/chat_test.go`

## 2. Frontend — Shared `useChatStream` Hook

- [x] 2.1 Create `web/src/hooks/useChatStream.ts` — exports `useChatStream()` returning `{ messages, send, isStreaming, reset }`
- [x] 2.2 Implement `send(userText, context?)`: appends user message to state, calls `POST /api/v1/chat/stream` via `fetch`, reads SSE stream via `ReadableStream` + `TextDecoder`
- [x] 2.3 Parse SSE chunks: split by `\n`, find `data:` lines, parse JSON, extract `choices[0].delta.content`, append to last assistant message in state
- [x] 2.4 Handle `data: [DONE]` (mark streaming done), `data: {"error":...}` (toast.error + remove pending message), fetch errors (toast.error + remove pending)
- [x] 2.5 Export `Message` type: `{ role: "user" | "assistant", content: string, streaming?: boolean }`

## 3. Frontend — `ChatSheet` Component

- [x] 3.1 Create `web/src/components/ChatSheet.tsx` using shadcn `Sheet` (`side="right"`, `className="w-[420px]"`)
- [x] 3.2 Wire `useChatStream()` for message state and send function
- [x] 3.3 Render message list: user messages right-aligned with `bg-primary text-primary-foreground` pill, assistant messages left-aligned plain text; auto-scroll to bottom on new message (use `useEffect` + `scrollIntoView` on a bottom anchor ref)
- [x] 3.4 Show typing indicator (three animated dots) when `isStreaming` is true
- [x] 3.5 Input row: `Input` + Send `Button` (disabled while streaming); submit on Enter key or button click
- [x] 3.6 Empty state: muted text `"Ask anything about this spec"` centered in message area when `messages.length === 0`
- [x] 3.7 Reset conversation on close (`onOpenChange(false)` calls `reset()`)

## 4. Frontend — Wire `ChatSheet` into `ProjectDetailView`

- [x] 4.1 Import `ChatSheet` in `ProjectDetailView.tsx`; add `chatOpen` state (`useState(false)`)
- [x] 4.2 Replace `toast.info("Coming soon")` on "Chat about this spec" button with `setChatOpen(true)`
- [x] 4.3 Render `<ChatSheet open={chatOpen} onOpenChange={setChatOpen} context={{ type: "spec", content: editedContent, label: selectedFile }} title={"Chat: " + selectedFile.split("/").pop()} />` at the bottom of the component JSX

## 5. Frontend — `CopilotContextProvider` and `useCopilotContext` Hook

- [x] 5.1 Create `web/src/hooks/useCopilotContext.tsx` — exports `CopilotContextProvider` (React context provider) and `useCopilotContext(ctx)` hook
- [x] 5.2 `CopilotContextProvider` holds `context: ChatContext | null` in state, provides `{ context, setContext }` via React context
- [x] 5.3 `useCopilotContext(ctx)` hook: calls `setContext(ctx)` on mount, `setContext(null)` on unmount; re-calls `setContext(ctx)` when `JSON.stringify(ctx)` changes
- [x] 5.4 Mount `CopilotContextProvider` in `Layout.tsx` wrapping both `GlobalNav` and `main`

## 6. Frontend — `CopilotPanel` Component

- [x] 6.1 Create `web/src/components/CopilotPanel.tsx` using shadcn `Dialog`
- [x] 6.2 Add global `keydown` listener: open on `⌘K` / `Ctrl+K` unless `document.activeElement` is INPUT/TEXTAREA/contenteditable; close on `Escape`
- [x] 6.3 Read `context` from `CopilotContextProvider` via `useContext`; display label in dialog header when present
- [x] 6.4 Wire `useChatStream()` for messages; pass `context` to `send()` on each message
- [x] 6.5 Same message rendering as `ChatSheet` (reuse the message list + input structure)
- [x] 6.6 Reset conversation on dialog close
- [x] 6.7 Mount `<CopilotPanel />` in `Layout.tsx` (alongside `CopilotContextProvider`)

## 7. Frontend — Context Registration in Views

- [x] 7.1 In `RunDetailView.tsx`: call `useCopilotContext({ type: "run", content: run.status.phase + ": " + run.spec.prompt, label: run.name })` when `run` is loaded
- [x] 7.2 In `ProjectDetailView.tsx` (spec tab with file selected): call `useCopilotContext({ type: "spec", content: editedContent, label: selectedFile })` when `selectedFile` changes
- [x] 7.3 In `ProjectDetailView.tsx` (project tab): call `useCopilotContext({ type: "project", content: project.spec.description || project.name, label: project.name })` when project loads

## 8. Verification

- [x] 8.1 `go test ./internal/server/...` passes including new chat handler tests
- [x] 8.2 `cd web && npx tsc --noEmit` passes with no errors
- [ ] 8.3 Manual: open a spec file in ProjectDetailView, click "Chat about this spec", send a message, verify streaming response appears token by token
- [ ] 8.4 Manual: navigate to a run, press `⌘K`, verify panel opens with run context in header, send a message, verify streaming response
- [ ] 8.5 Manual: verify `⌘K` does NOT open the panel when cursor is in the spec editor Monaco input
