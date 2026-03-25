## Why

The existing CopilotPanel (⌘K dialog) doesn't work — chat returns no visible response due to reasoning-only tokens — and its modal design breaks flow. Users need a persistent, VSCode-style bottom panel that stays open while they work, has access to full UI context, can guide navigation, and persists conversation history across page changes.

## What Changes

- Replace center-modal CopilotPanel with a bottom drawer panel (like VSCode's terminal/debug panel)
- Wire `LITELLM_MASTER_KEY` in Helm so the chat endpoint authenticates correctly
- Handle `reasoning_content` tokens from thinking models (Qwen3, DeepSeek)
- Expand context injection: full page state (run trace, spec content, project metadata, errors visible on screen)
- Add session-based chat history persisted to `localStorage` — conversations survive navigation
- Add guided navigation support: copilot can suggest and highlight UI elements
- Add a "New chat" / session selector to the panel header

## Capabilities

### New Capabilities
- `copilot-bottom-panel`: Bottom-anchored resizable panel replacing the center dialog, toggled by ⌘K
- `copilot-session-history`: Chat sessions persisted in localStorage, viewable/switchable from panel header
- `copilot-ui-guidance`: Copilot can emit navigation actions (highlight element, suggest route) via structured output

### Modified Capabilities
- `copilot-panel`: Panel layout changes from Dialog to bottom drawer; existing message rendering reused
- `chat-stream-api`: No API changes; client-side fix for reasoning_content tokens
