// web/src/lib/keybindings/types.ts — Core types for the keybinding system.

/** All dispatchable action identifiers. */
export type ActionID =
  | "nav.runs"
  | "nav.projects"
  | "nav.settings"
  | "run.new"
  | "ui.modal.close"
  | "ui.search.focus"
  | "system.reload";

/** Human-readable description for each action, used in Settings UI. */
export const ACTION_DESCRIPTIONS: Record<ActionID, string> = {
  "nav.runs":         "Go to Runs",
  "nav.projects":     "Go to Projects",
  "nav.settings":     "Go to Settings",
  "run.new":          "New Run",
  "ui.modal.close":   "Close Modal",
  "ui.search.focus":  "Focus Search",
  "system.reload":    "Reload App",
};

/** All known action IDs in a stable order for the Settings table. */
export const ALL_ACTIONS: ActionID[] = [
  "nav.runs",
  "nav.projects",
  "nav.settings",
  "run.new",
  "ui.modal.close",
  "ui.search.focus",
  "system.reload",
];

/**
 * A key binding entry: maps a key sequence string to an action.
 * Key sequences are space-separated tokens, e.g. "g r" or "ctrl+x ,".
 */
export interface KeyBinding {
  /** The key sequence, e.g. "g r" or "Escape" or "ctrl+x ," */
  keys: string;
  action: ActionID;
}

/** Keybinding configuration stored in AppSettings. */
export interface KeybindingsConfig {
  /** One of "default" | "vim" | "emacs" | "custom". Defaults to "default". */
  preset: "default" | "vim" | "emacs" | "custom";
  /** Sparse overrides: actionID → key sequence, delta from the chosen preset. */
  overrides: Record<string, string>;
  /** How long (ms) to wait before showing the which-key popup. Defaults to 500. */
  whichKeyDelayMs: number;
}

export const KEYBINDINGS_DEFAULTS: KeybindingsConfig = {
  preset: "default",
  overrides: {},
  whichKeyDelayMs: 500,
};
