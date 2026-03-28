## Context

Zero keyboard handling exists today. The GlobalNav uses click-driven React Router Links. `KeybindingHint.tsx` exists as a static display component only. `useSettings` context handles persistence via Wails `GetSettings`/`SaveSettings`.

## Goals / Non-Goals

**Goals:** Global key dispatch, preset switching, chord sequences, which-key popup, configurable delay, Settings UI for per-binding capture.

**Non-Goals:** Wails native menu accelerators, Windows/Linux specifics, multi-key combos beyond 2-key chords.

## Decisions

**D1: Three preset leader styles**
- Default: `g` as leader (`g w` = workflows, `g r` = run, etc.)
- Vim: `space` as leader (`space w`, `space r`, etc.)
- Emacs: `ctrl+x` as leader (`ctrl+x w`, `ctrl+x ,` for settings, etc.)

**D2: Data model — preset + sparse overrides**
```typescript
interface KeybindingsConfig {
  preset: "default" | "vim" | "emacs" | "custom";
  overrides: Record<string, string>; // actionID -> keys, delta from preset
  whichKeyDelayMs: number; // default 500
}
```
`effectiveBindings = merge(PRESETS[preset], overrides)`

**D3: Chord state machine**
States: IDLE | CHORD_PENDING
- IDLE: keydown → if full match dispatch; if prefix → CHORD_PENDING + start timer
- CHORD_PENDING: keydown → if completes chord dispatch; Escape → IDLE; unrecognized → IDLE
- Skip if target is INPUT/TEXTAREA/SELECT/[contenteditable]

**D4: Which-key popup positioning**
Fixed bottom-right (`bottom: 24px; right: 24px`), compact card, fade+translateY animation 120ms, dismiss 80ms.

**D5: ModalContext for close/confirm dispatch**
Thin stack: modals register on mount/deregister on unmount. `dispatch("ui.modal.close")` calls `modalContext.closeTop()`. Avoids DOM event coupling.

**D6: Key capture UI**
Click ✎ → capture next keydown → validate conflict → write to overrides. Inline ⚠ warning on conflict. Per-action ↺ reset.

## Risks / Trade-offs

- Browser key interception conflicts with OS shortcuts → only preventDefault on keys we handle
- Input field interception → guard with isTextInput() check on every keydown
- Preset switch losing overrides → require inline confirmation before clearing
