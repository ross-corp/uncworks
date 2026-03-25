## Capability: copilot-panel

A global ambient chat panel accessible from any view via `⌘K` / `Ctrl+K`, with automatic page-aware context injection. Mounted once in `Layout.tsx` outside the route outlet.

---

### Requirement: context-provider

A React context `CopilotContextProvider` wraps the app in `Layout.tsx`. It exposes:

```tsx
interface CopilotContextValue {
  context: ChatContext | null;
  setContext: (ctx: ChatContext | null) => void;
}

interface ChatContext {
  type: "spec" | "run" | "project" | "general";
  content: string;
  label: string;
}
```

The `useCopilotContext(ctx)` hook calls `setContext(ctx)` on mount and `setContext(null)` on unmount, so the panel always reflects what's currently on screen.

**Acceptance Criteria:**
- `CopilotContextProvider` is mounted in `Layout.tsx` wrapping both `<GlobalNav>` and `<main>`
- `useCopilotContext(ctx)` registers context on mount, clears on unmount
- Context updates when the `ctx` argument changes (use `useEffect` with deep equality or JSON stringify comparison)

---

### Requirement: keyboard-trigger

The panel opens when the user presses `⌘K` (Mac) or `Ctrl+K` (Windows/Linux), unless focus is currently inside a text input, textarea, or contenteditable element.

**Acceptance Criteria:**
- Global `keydown` listener added in the `CopilotPanel` component (or `Layout.tsx`), cleaned up on unmount
- Does NOT fire when `document.activeElement` is `INPUT`, `TEXTAREA`, or `[contenteditable]`
- Pressing `Escape` closes the panel
- No visible UI button — keyboard-only trigger for v1

---

### Requirement: panel-layout

The panel is implemented as a shadcn `Dialog` (not Sheet — it's more overlay-like for a global tool).

Layout:
1. Dialog header: "Copilot" title + current context label in muted text (e.g., `"spec: openspec/specs/user-auth/spec.md"`)
2. Scrollable message list (same rendering as ChatSheet)
3. Input area at bottom

**Acceptance Criteria:**
- Dialog is `max-w-[600px] w-full` centered
- If no context is registered, header shows `"Copilot"` with no label
- Conversation resets when the panel is closed and reopened (no persistence in v1)

---

### Requirement: context-views

The following views register their context via `useCopilotContext()`:

| View | Context type | Content | Label |
|------|-------------|---------|-------|
| `ProjectDetailView` (spec tab, file selected) | `"spec"` | `editedContent` | `selectedFile` |
| `RunDetailView` | `"run"` | `run.status.phase + ": " + run.spec.prompt` | `run.name` |
| `ProjectDetailView` (project tab) | `"project"` | project description + settings summary | project name |

**Acceptance Criteria:**
- Each listed view calls `useCopilotContext()` with the appropriate data
- Context is updated when the relevant state changes (e.g., switching files in ProjectDetailView)
- Views not listed above do not register context (panel opens with no context, generic system message used)

---

### Requirement: streaming-consumption

Identical to `chat-sheet` streaming requirement — same `fetch` + `ReadableStream` approach, same error handling. The implementation should be extracted into a shared `useChatStream()` hook used by both `ChatSheet` and `CopilotPanel`.

**Acceptance Criteria:**
- `useChatStream()` hook encapsulates: message state, send function, streaming state
- Both `ChatSheet` and `CopilotPanel` use `useChatStream()`
- No duplicated streaming logic between the two components
