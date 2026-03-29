## Context

The existing `CopilotPanel` is a shadcn `Dialog` triggered by ⌘K. It works but:
1. Auth was broken (no `LITELLM_MASTER_KEY` in Helm)
2. The "default" LiteLLM model (`qwen3:8b`) emits `reasoning_content` tokens before `content` — the hook only reads `content`, so the panel appeared empty
3. The center-modal design disappears when you click away, losing context mid-conversation
4. No session persistence — every navigation resets the chat
5. No ability to guide the user (highlight elements, navigate to a route)

The `useChatStream` hook, `useCopilotContext` provider, and `/api/v1/chat/stream` endpoint are all correct and reusable.

## Goals / Non-Goals

**Goals:**
- Bottom panel UI like VSCode debug/terminal (fixed height, resizable, toggle with ⌘K)
- Panel stays mounted while navigating — messages survive route changes
- Session history in localStorage: create new session, switch between recent sessions
- Context injection keeps working — views register their context via `useCopilotContext`
- Copilot can emit `[NAV: /path]` and `[HIGHLIGHT: selector]` actions parsed client-side
- Handle reasoning_content tokens from thinking models

**Non-Goals:**
- Backend session storage (localStorage only for now)
- Multi-user chat or shared sessions
- Voice/image input
- Full agent tool-use (copilot stays a chat interface, not an agent runner)

## Decisions

**Bottom panel layout**
Use a fixed `div` at the bottom of the Layout (outside `<main>`), with `position: fixed; bottom: 0; width: 100%`. Panel height is draggable (stored in state, default 320px). A thin resize handle at the top edge. When closed, panel is `display: none` (not unmounted) so messages persist.

**State placement**
Move copilot state (messages, sessions, open/closed, height) out of `CopilotPanel` into a new `useCopilot` hook stored in the `CopilotContextProvider` (already at Layout level). This means panel state survives route changes — the panel just stays open.

**Session model**
```ts
interface ChatSession {
  id: string;          // nanoid
  createdAt: number;
  title: string;       // first user message, truncated
  messages: Message[];
}
```
Sessions stored in `localStorage` under `unc:copilot:sessions`. Max 20 sessions (oldest pruned). Active session ID stored in state. "New chat" creates a fresh session. Session list shown as a dropdown in panel header.

**UI guidance actions**
Copilot response text is scanned for structured tokens before display:
- `[NAV:/path]` → calls `navigate(path)` from react-router
- `[HIGHLIGHT:css-selector]` → temporarily adds a `ring-2 ring-primary` class to matching elements, removed after 3s

These tokens are stripped from displayed text. This is opt-in; the system prompt tells the copilot it can use them.

**Context injection**
No changes to `useCopilotContext` — views still call it to register page context. The context is passed with each message to the backend. Enhanced system prompt includes guidance on navigation actions and highlights.

**Reasoning content**
`useChatStream` reads `delta.content ?? delta.reasoning_content` — both contribute to the displayed assistant message. No visual distinction between thinking vs answer (keeps it simple).

## Risks / Trade-offs

- localStorage has ~5MB limit — long sessions may hit it; mitigated by max 20 sessions and trimming old ones
- `[HIGHLIGHT:selector]` parsing is client-side only; if the selector is wrong or element not on screen, it silently fails (acceptable)
- Resizing UX depends on pointer events working smoothly — simple implementation using `onPointerDown` + `onPointerMove` on the resize handle
- Panel `position: fixed` may conflict with existing modals; tested in shadcn Dialog/Sheet which use portals (should be fine)
