## Capability: chat-sheet

A scoped chat panel implemented as a shadcn Sheet, opened from the spec editor in `ProjectDetailView`. Pre-seeded with the current spec file content as context.

---

### Requirement: component-interface

`ChatSheet` is a React component accepting:
```tsx
interface ChatSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  context?: {
    type: "spec" | "run" | "project" | "general";
    content: string;
    label: string;
  };
  title?: string; // defaults to "Chat"
}
```

---

### Requirement: sheet-layout

The Sheet opens from the right side (`side="right"`), width `w-[420px]` on desktop.

Layout (top to bottom):
1. `SheetHeader` with `SheetTitle` showing `title` prop
2. Scrollable message list (flex-col, overflow-y-auto, flex-1)
3. Input area pinned to bottom: single-line `Input` + Send `Button`

**Acceptance Criteria:**
- Message list scrolls independently of the sheet header and input
- Auto-scrolls to the bottom when a new message is added
- Empty state shows: `"Ask anything about this spec"` in muted text

---

### Requirement: message-rendering

Messages are rendered in a list. Each message has:
- User messages: right-aligned, `bg-primary text-primary-foreground` pill bubble
- Assistant messages: left-aligned, plain text with `prose-sm` markdown rendering (use `ReactMarkdown` if available, otherwise `<pre className="whitespace-pre-wrap">`)
- A typing indicator (animated ellipsis) shown while streaming

**Acceptance Criteria:**
- Messages are never empty — user messages show the sent text, assistant messages stream in token by token
- The Send button is disabled while a response is streaming

---

### Requirement: streaming-consumption

On send, the component calls `POST /api/v1/chat/stream` via `fetch` (not `apiFetch`) with the full conversation history and the `context` prop.

It reads the SSE stream using `ReadableStream`:
```ts
const reader = resp.body.getReader();
const decoder = new TextDecoder();
// read chunks, parse "data: {...}" lines, extract delta.content, append to last message
```

**Acceptance Criteria:**
- Each `data:` SSE line is parsed; `choices[0].delta.content` is appended to the streaming assistant message
- `data: [DONE]` ends the stream and marks the message complete
- `data: {"error": "..."}` shows an error toast and removes the pending message
- On fetch error (network), shows `toast.error("Chat unavailable")` and removes pending message

---

### Requirement: wire-into-spec-editor

In `ProjectDetailView.tsx`, the "Chat about this spec" button opens `ChatSheet` with:
- `context.type = "spec"`
- `context.content = editedContent` (current file content)
- `context.label = selectedFile`
- `title = "Chat: " + selectedFile.split("/").pop()`

**Acceptance Criteria:**
- Button click opens the sheet (replaces the `toast.info("Coming soon")` placeholder)
- Sheet closes on the X button or pressing Escape
- Switching to a different spec file while the sheet is open does NOT auto-update the context (user must reopen)
