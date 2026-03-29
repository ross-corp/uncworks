// web/src/lib/keybindings/types.ts
// Keybinding action IDs, config shape, and descriptions.

export type ActionID =
  | "nav.workflows"
  | "nav.projects"
  | "nav.settings"
  | "run.new"
  | "run.cancel"
  | "ui.modal.close"
  | "ui.search.focus"
  | "system.reload";

export interface KeybindingsConfig {
  preset: "default" | "vim" | "emacs" | "custom";
  overrides: Record<string, string>;
  whichKeyDelayMs: number;
}

export const ACTION_DESCRIPTIONS: Record<ActionID, string> = {
  "nav.workflows": "Go to Workflows",
  "nav.projects": "Go to Projects",
  "nav.settings": "Open Settings",
  "run.new": "New Run",
  "run.cancel": "Cancel Run",
  "ui.modal.close": "Close Modal",
  "ui.search.focus": "Focus Search",
  "system.reload": "Reload",
};

export type KeyBinding = string;
