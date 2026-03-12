#!/usr/bin/env node
/**
 * TUI Dashboard entry point.
 *
 * Usage: tsx packages/tui/src/main.ts [--server http://localhost:50051]
 */

import { createRoot } from "solid-js";
import { createConnectTransport } from "@connectrpc/connect-node";
import { Runtime } from "./runtime.js";
import { parseInput } from "./input.js";
import { createAppState, handleAction } from "./state.js";
import { startRenderLoop } from "./loop.js";
import { DataBinding } from "./data.js";
import { AOTClient } from "../../shared/src/grpc/client.js";

function getServerUrl(): string {
  const idx = process.argv.indexOf("--server");
  if (idx !== -1 && process.argv[idx + 1]) {
    return process.argv[idx + 1];
  }
  return process.env.AOT_SERVER_URL ?? "http://localhost:50051";
}

createRoot(() => {
  const serverUrl = getServerUrl();
  const transport = createConnectTransport({
    baseUrl: serverUrl,
    httpVersion: "1.1",
  });
  const client = new AOTClient({ baseUrl: serverUrl, transport });
  const state = createAppState();
  const runtime = new Runtime();
  const data = new DataBinding(client, state);

  // Wire up the render loop (reactive — re-renders on signal change)
  startRenderLoop(runtime, state);

  // Wire up input handling
  runtime.onKey((key) => {
    const action = parseInput(key);
    if (!action) return;

    // Handle HITL submit
    if (state.viewMode() === "input" && action.type === "enter") {
      const buf = state.inputBuffer();
      if (buf.trim()) {
        const run = state.runs()[state.selectedIndex()];
        if (run) {
          void data.sendInput(run.id, buf).then((accepted) => {
            if (accepted) {
              state.setViewMode("detail");
              state.setInputBuffer("");
            }
          });
        }
      }
      return;
    }

    const shouldQuit = handleAction(state, action);
    if (shouldQuit) {
      data.stop();
      runtime.stop();
      process.exit(0);
    }

    // Watch/unwatch on selection changes or mode transitions
    if (action.type === "enter" && state.viewMode() === "detail") {
      const run = state.runs()[state.selectedIndex()];
      if (run) data.watchRun(run.id);
    }
    if (action.type === "escape" && state.viewMode() === "list") {
      data.stopWatch();
    }
  });

  // Start
  runtime.start();
  void data.fetchRuns();
  data.startAutoRefresh();
});
