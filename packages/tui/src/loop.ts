/**
 * Render loop — connects SolidJS reactive state to the terminal via createEffect.
 */

import { createEffect } from "solid-js";
import { renderToString } from "./renderer.js";
import { dashboardView } from "./views.js";
import type { Runtime } from "./runtime.js";
import type { AppState } from "./state.js";

const SCREEN_CLEAR = "\x1b[2J\x1b[H";

/** Start a reactive render loop that re-renders whenever state signals change. */
export function startRenderLoop(runtime: Runtime, state: AppState): void {
  createEffect(() => {
    const runs = state.runs();
    const idx = state.selectedIndex();
    const selectedRun = runs[idx] ?? null;
    const mode = state.viewMode();
    const inputBuf = state.inputBuffer();
    const err = state.error();

    const view = dashboardView(runs, idx, selectedRun, mode);
    let output = renderToString(view);

    // Append input line in input mode
    if (mode === "input") {
      output += "\n  > " + inputBuf + "█";
    }

    // Append error if present
    if (err) {
      output += "\n  \x1b[31m" + err + "\x1b[0m";
    }

    runtime.write(SCREEN_CLEAR + output + "\n");
  });
}
