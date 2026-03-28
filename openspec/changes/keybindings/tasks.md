## 1. Data Model and Persistence

- [ ] 1.1 Add `KeybindingsConfig` struct to Go `AppSettings` in `cmd/uncworks-app/settings.go`: fields `Preset string`, `Overrides map[string]string`, `WhichKeyDelayMs int`
- [ ] 1.2 Create `web/src/lib/keybindings/types.ts`: `ActionID` union, `KeybindingsConfig` interface, `ACTION_DESCRIPTIONS` map, `KeyBinding` type
- [ ] 1.3 Create `web/src/lib/keybindings/presets.ts`: `PRESETS` constant with default/vim/emacs maps
- [ ] 1.4 Create `web/src/lib/keybindings/resolve.ts`: `resolveEffectiveBindings()`, `resolveChordPrefixes()`, `isTextInput()` helpers

## 2. KeybindingsContext and Chord State Machine

- [ ] 2.1 Create `web/src/contexts/ModalContext.tsx`: modal stack, `useModal` hook, `ModalProvider`
- [ ] 2.2 Create `web/src/contexts/KeybindingsContext.tsx`: `KeybindingsProvider` with global keydown listener, chord state machine (IDLE/CHORD_PENDING), `dispatch(action)`, `useKeybindings` hook
- [ ] 2.3 Implement all `dispatch` cases: navigation (useNavigate), run.new (emit event or ModalContext), ui.modal.close (ModalContext), ui.search.focus (querySelector), system.reload (emit event)
- [ ] 2.4 Wire `NewRunModal` and `WebhookModal` to `ModalContext` (register on mount, deregister on unmount)
- [ ] 2.5 Integrate `KeybindingsProvider` and `ModalProvider` into `web/src/App.tsx`

## 3. Which-Key Popup

- [ ] 3.1 Create `web/src/components/WhichKeyPopup.tsx`: accepts `visible`, `prefix`, `entries[]`; fixed bottom-right positioning; monospace key column; fade+translateY animation via CSS
- [ ] 3.2 Update `web/src/components/KeybindingHint.tsx`: add macOS symbol normalization (meta→⌘, ctrl→⌃, alt→⌥, shift→⇧)
- [ ] 3.3 Wire popup visibility to chord state machine timer in `KeybindingsContext`
- [ ] 3.4 Render `WhichKeyPopup` as a React portal in `App.tsx`
- [ ] 3.5 Add pointerdown outside handler to dismiss popup

## 4. Settings UI

- [ ] 4.1 Add `KeybindingsSettingsSection` component in `web/src/views/SettingsView.tsx`
- [ ] 4.2 Implement preset radio group with inline "clear overrides" confirmation warning
- [ ] 4.3 Implement which-key delay slider (0–2000ms, step 50ms)
- [ ] 4.4 Implement collapsible bindings table with `<kbd>` display per row
- [ ] 4.5 Implement key capture mode: click edit icon, capture next keydown, handle chord second-key wait
- [ ] 4.6 Implement inline conflict detection warning
- [ ] 4.7 Implement per-action reset icon (removes single override)
- [ ] 4.8 Implement "Reset All to Preset Defaults" button (clears all overrides)

## 5. Polish

- [ ] 5.1 Prevent default browser behavior on keys we handle (stop browser zoom on ctrl++, etc.)
- [ ] 5.2 Smart single-match skip: if only one binding matches prefix and whichKeyDelayMs=0, dispatch immediately without showing popup
- [ ] 5.3 Add aria-label to which-key popup for accessibility
- [ ] 5.4 Verify text inputs in modals still accept free typing (no regression)
