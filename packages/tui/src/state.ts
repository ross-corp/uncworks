/**
 * Application state — SolidJS signals for reactive TUI rendering.
 */

import { createSignal } from "solid-js";
import type { AgentRunView } from "./views";
import type { InputAction } from "./input";

export type ViewMode = "list" | "detail" | "input";

export interface AppState {
  runs: () => AgentRunView[];
  setRuns: (runs: AgentRunView[]) => void;
  selectedIndex: () => number;
  setSelectedIndex: (i: number) => void;
  viewMode: () => ViewMode;
  setViewMode: (mode: ViewMode) => void;
  inputBuffer: () => string;
  setInputBuffer: (s: string) => void;
  error: () => string | null;
  setError: (e: string | null) => void;
}

/** Create the reactive application state. */
export function createAppState(): AppState {
  const [runs, setRuns] = createSignal<AgentRunView[]>([]);
  const [selectedIndex, setSelectedIndex] = createSignal(0);
  const [viewMode, setViewMode] = createSignal<ViewMode>("list");
  const [inputBuffer, setInputBuffer] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);

  return {
    runs,
    setRuns,
    selectedIndex,
    setSelectedIndex,
    viewMode,
    setViewMode,
    inputBuffer,
    setInputBuffer,
    error,
    setError,
  };
}

/** Handle an input action against the current state. Returns true if the app should quit. */
export function handleAction(state: AppState, action: InputAction): boolean {
  const mode = state.viewMode();

  // Input mode: collect keystrokes
  if (mode === "input") {
    switch (action.type) {
      case "escape":
        state.setViewMode("detail");
        state.setInputBuffer("");
        return false;
      case "enter":
        // Submit is handled externally — signal via return value
        return false;
      case "backspace": {
        const buf = state.inputBuffer();
        state.setInputBuffer(buf.slice(0, -1));
        return false;
      }
      case "char":
        state.setInputBuffer(state.inputBuffer() + action.char);
        return false;
      default:
        return false;
    }
  }

  // List/detail mode
  switch (action.type) {
    case "quit":
      return true;
    case "up": {
      const idx = state.selectedIndex();
      if (idx > 0) state.setSelectedIndex(idx - 1);
      return false;
    }
    case "down": {
      const idx = state.selectedIndex();
      const max = state.runs().length - 1;
      if (idx < max) state.setSelectedIndex(idx + 1);
      return false;
    }
    case "enter": {
      if (mode === "list") {
        const run = state.runs()[state.selectedIndex()];
        if (run?.phase === "WaitingForInput") {
          state.setViewMode("input");
          state.setInputBuffer("");
        } else {
          state.setViewMode("detail");
        }
      }
      return false;
    }
    case "escape":
      if (mode === "detail") {
        state.setViewMode("list");
      }
      return false;
    default:
      return false;
  }
}
