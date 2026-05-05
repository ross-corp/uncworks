// web/playwright-app.config.ts — Playwright e2e config for driving the UNCWORKS
// desktop app (Wails v2) via CDP. Requires the app to be running with devtools
// enabled (built with `task build:app:devtools`).
//
// Run with: task test:e2e:app
// The task checks that the app is running first.
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e-app",
  fullyParallel: false, // Serial: single Wails window
  forbidOnly: !!process.env.CI,
  retries: 2,
  workers: 1,
  reporter: "list",
  timeout: 30_000,
  use: {
    actionTimeout: 10_000,
    trace: "on-first-retry",
  },
  // No webServer — we connect to the already-running Wails app via CDP.
});
