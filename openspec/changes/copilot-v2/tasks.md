## 1. Copilot State & Session Model

- [x] 1.1 Create `useCopilotSessions` hook: `ChatSession` type, localStorage read/write, max 20 sessions, pruning
- [x] 1.2 Add `activeSessionId`, `sessions`, `setActiveSession`, `createSession`, `updateSession` to `CopilotContextProvider`
- [x] 1.3 Migrate `useChatStream` messages state into session model — messages live in the active session
- [x] 1.4 On mount: restore last active session from localStorage; on new message: auto-create session if none active

## 2. Bottom Panel Component

- [x] 2.1 Create `CopilotBottomPanel.tsx` replacing `CopilotPanel.tsx` — fixed bottom, full width, z-50
- [x] 2.2 Add resize handle: pointer events drag to set panel height (min 200px, max 70vh), stored in state
- [x] 2.3 Add panel header: "Copilot" title, context label, session dropdown (shadcn Popover + list), "New chat" button
- [x] 2.4 Wire ⌘K/Ctrl+K global shortcut (guard against INPUT/TEXTAREA/contenteditable) — toggle open state in context
- [x] 2.5 Escape key closes panel; panel stays mounted (not destroyed) when closed to preserve messages
- [x] 2.6 Message list: same rendering as current CopilotPanel (user right-aligned bubble, assistant left plain text + TypingDots)
- [x] 2.7 Input row: same Input + Send button; auto-focus when panel opens

## 3. Layout Integration

- [x] 3.1 Remove `<CopilotPanel />` from Layout, add `<CopilotBottomPanel />`
- [x] 3.2 Add bottom padding to `<main>` equal to panel height when panel is open (so content isn't hidden behind panel)

## 4. UI Guidance Actions

- [x] 4.1 Create `parseGuidanceActions(text)` utility: extracts `[NAV:/path]` and `[HIGHLIGHT:selector]` tokens, returns cleaned text + actions array
- [x] 4.2 Apply `parseGuidanceActions` to each assistant message before display; execute actions (navigate / highlight) when message finalizes (streaming=false)
- [x] 4.3 `applyHighlight(selector)`: querySelectorAll, add `ring-2 ring-primary ring-offset-1` class, remove after 3s

## 5. System Prompt Update

- [ ] 5.1 Update `buildChatSystemMessage` in `chat.go` to include guidance on `[NAV:/path]` and `[HIGHLIGHT:selector]` tokens in extended system prompt
- [x] 5.2 Update `chat_test.go` to cover extended system prompt content

## 6. Cleanup

- [x] 6.1 Remove old `CopilotPanel.tsx` file
- [x] 6.2 Remove `ChatSheet.tsx` open/close state from `ProjectDetailView` — replace with opening the bottom panel (set context + open)
- [x] 6.3 Verify TypeScript compiles clean
- [ ] 6.4 Manual: open panel with ⌘K, send a message, verify streaming response visible
- [ ] 6.5 Manual: navigate to a different route while panel is open, verify messages persist
- [ ] 6.6 Manual: refresh page, verify last session restored
- [ ] 6.7 Manual: drag resize handle, verify height changes
