// web/src/lib/keybindings/presets.ts — Built-in keybinding preset maps.
import type { ActionID } from "./types";

/**
 * Each preset is a map of ActionID → key sequence string.
 * Key sequences are space-separated tokens:
 *   single key: "n", "/"
 *   chord:      "g r", "space r", "ctrl+x ,"
 *   modifier:   "Escape", "ctrl+g"
 */
export type PresetMap = Partial<Record<ActionID, string>>;

/** Default preset: "g" as chord leader (similar to Spacemacs / vim-unimpaired). */
const DEFAULT_PRESET: PresetMap = {
  "nav.runs":        "g r",
  "nav.projects":    "g p",
  "nav.settings":    "g s",
  "run.new":         "n",
  "ui.modal.close":  "Escape",
  "ui.search.focus": "/",
  "system.reload":   "r",
};

/** Vim preset: Space as chord leader. */
const VIM_PRESET: PresetMap = {
  "nav.runs":        "space r",
  "nav.projects":    "space p",
  "nav.settings":    "space s",
  "run.new":         "space n",
  "ui.modal.close":  "Escape",
  "ui.search.focus": "/",
  "system.reload":   "space R",
};

/** Emacs preset: Ctrl+x as chord leader. */
const EMACS_PRESET: PresetMap = {
  "nav.runs":        "ctrl+x r",
  "nav.projects":    "ctrl+x p",
  "nav.settings":    "ctrl+x ,",
  "run.new":         "ctrl+x n",
  "ui.modal.close":  "ctrl+g",
  "ui.search.focus": "ctrl+s",
  "system.reload":   "ctrl+x R",
};

export const PRESETS: Record<"default" | "vim" | "emacs", PresetMap> = {
  default: DEFAULT_PRESET,
  vim:     VIM_PRESET,
  emacs:   EMACS_PRESET,
};

export const PRESET_LABELS: Record<"default" | "vim" | "emacs", string> = {
  default: "Default (g leader)",
  vim:     "Vim (Space leader)",
  emacs:   "Emacs (C-x leader)",
};
