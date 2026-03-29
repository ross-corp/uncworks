// playwright.config.ts — Playwright e2e configuration for the uncworks web UI.
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "list",
  use: {
    // Dev server runs on port 3000 (see vite.config.ts)
    baseURL: "http://localhost:3000",
    actionTimeout: 10_000,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3000",
    reuseExistingServer: !process.env.CI,
    env: {
      // Point the app at the local API server.
      // Override at runtime: VITE_API_URL=http://localhost:50055 npm run test:e2e
      VITE_API_URL: process.env.VITE_API_URL ?? "http://localhost:50055",
    },
  },
});
