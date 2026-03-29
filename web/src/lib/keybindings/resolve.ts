// web/src/lib/keybindings/resolve.ts — Effective binding resolution and helpers.
import type { ActionID, KeybindingsConfig, KeyBinding } from "./types";
import { PRESETS } from "./presets";

/**
 * Merge the chosen preset with any user overrides to produce the effective
 * set of bindings. Overrides with an empty string remove a binding.
 */
export function resolveEffectiveBindings(cfg: KeybindingsConfig): KeyBinding[] {
  const presetKey = cfg.preset === "custom" ? "default" : cfg.preset;
  const base = PRESETS[presetKey] ?? PRESETS["default"];

  // Merge: start with preset, apply overrides on top.
  const merged: Partial<Record<ActionID, string>> = { ...base };
  for (const [action, keys] of Object.entries(cfg.overrides ?? {})) {
    if (keys.trim() === "") {
      delete merged[action as ActionID];
    } else {
      merged[action as ActionID] = keys.trim();
    }
  }

  return Object.entries(merged)
    .filter(([, keys]) => Boolean(keys))
    .map(([action, keys]) => ({ action: action as ActionID, keys: keys as string }));
}

/**
 * Build the set of chord prefixes from the effective bindings.
 * A chord prefix is the first token of any multi-token key sequence.
 * e.g. for "g r", the prefix is "g".
 */
export function resolveChordPrefixes(bindings: KeyBinding[]): Set<string> {
  const prefixes = new Set<string>();
  for (const { keys } of bindings) {
    const tokens = keys.split(" ").map((t) => t.trim()).filter(Boolean);
    if (tokens.length >= 2) {
      prefixes.add(tokens[0]);
    }
  }
  return prefixes;
}

/**
 * Returns true when the keyboard event's target is a text-entry element.
 * Used to suppress keybinding interception while the user is typing.
 */
export function isTextInput(target: EventTarget | null): boolean {
  if (!target || !(target instanceof Element)) return false;
  const tag = target.tagName.toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return true;
  if ((target as HTMLElement).isContentEditable) return true;
  return false;
}

/**
 * Normalise a raw KeyboardEvent into a canonical key sequence token.
 * Modifier order: ctrl, alt, meta, shift — then the key.
 * Special cases: "Control" → "ctrl", "Meta" → "meta", " " → "space", etc.
 */
export function normaliseKey(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey)  parts.push("ctrl");
  if (e.altKey)   parts.push("alt");
  if (e.metaKey)  parts.push("meta");

  let key = e.key;

  // Normalise modifier-only keypresses (skip pure-modifier events)
  if (["Control", "Alt", "Meta", "Shift"].includes(key)) return "";

  if (key === " ") key = "space";
  if (key === "Escape") key = "Escape";

  // Lowercase single printable chars (but preserve shift+letter as uppercase
  // only when shift is alone — we don't emit "shift+R", we emit "R" or "r").
  if (!e.ctrlKey && !e.altKey && !e.metaKey && key.length === 1) {
    // If shift is active we get the uppercase char already (e.g. "R").
    // Don't add "shift+" prefix for printable characters.
    parts.push(key);
    return parts.join("+");
  }

  if (e.shiftKey && key.length > 1) parts.push("shift");
  parts.push(key);
  return parts.join("+");
}
