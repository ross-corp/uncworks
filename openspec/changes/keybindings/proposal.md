## Why

UNCWORKS has zero keyboard shortcuts today — all navigation and actions are click-only. Power users (especially developers who live in vim/emacs) expect keyboard-first navigation. Adding keybindings with a which-key popup makes the app feel native and efficient without requiring users to memorize shortcuts upfront.

## What Changes

- **Configurable keybindings** — three preset modes (default, vim, emacs) plus user-defined custom overrides
- **Global chord state machine** — intercepts keydown events, routes to actions, handles two-key chord sequences
- **Which-key popup** — unobtrusive bottom-right floating panel that appears after a configurable delay when a chord prefix is held, showing available completions
- **Settings UI** — new Keybindings section in Settings with preset selector, delay slider, and per-binding edit/capture UI
- **ModalContext** — thin modal stack context for `ui.modal.close` dispatch

## Capabilities

### New Capabilities

- `keybindings-system`: Global keybinding dispatch, chord state machine, preset maps, effective binding resolution
- `which-key-popup`: Bottom-right floating which-key panel triggered by chord prefix with configurable delay
- `keybindings-settings-ui`: Settings section for preset selection, delay config, per-binding key capture with conflict detection

### Modified Capabilities

## Impact

- `web/src/lib/keybindings/` — new module (types, presets, resolve)
- `web/src/contexts/KeybindingsContext.tsx` — provider + state machine
- `web/src/contexts/ModalContext.tsx` — thin modal stack
- `web/src/components/WhichKeyPopup.tsx` — popup display
- `web/src/App.tsx` — add providers
- `web/src/views/SettingsView.tsx` — add KeybindingsSettingsSection
- `web/src/components/KeybindingHint.tsx` — macOS symbol normalization
- Go AppSettings struct — add KeybindingsConfig field
