// web/e2e-app/smoke.spec.ts — Smoke e2e tests for the UNCWORKS desktop app.
// Requires UNCWORKS to be running: task build:app:devtools && open /Applications/UNCWORKS.app
import { test, expect } from "@playwright/test";
import { connectToApp } from "./helpers";
import type { Browser } from "@playwright/test";

let browser: Browser;

test.beforeAll(async () => {
  ({ browser } = await connectToApp());
});

test.afterAll(async () => {
  await browser.close();
});

test("app loads and shows main navigation", async () => {
  const page = browser.contexts()[0].pages()[0];
  // Navigation sidebar or top bar should be visible
  await expect(page.locator("nav, [role=navigation]").first()).toBeVisible({ timeout: 10_000 });
});

test("navigate to projects page", async () => {
  const page = browser.contexts()[0].pages()[0];
  const projectsLink = page.getByRole("link", { name: /projects/i }).first();
  await projectsLink.click();
  // Should show project list or empty state
  await expect(page.locator("body")).toContainText(/project/i, { timeout: 5_000 });
});

test("navigate to runs page", async () => {
  const page = browser.contexts()[0].pages()[0];
  const runsLink = page.getByRole("link", { name: /runs/i }).first();
  await runsLink.click();
  await expect(page.locator("body")).toContainText(/run/i, { timeout: 5_000 });
});

test("navigate to settings page", async () => {
  const page = browser.contexts()[0].pages()[0];
  const settingsLink = page.getByRole("link", { name: /settings/i }).first();
  await settingsLink.click();
  // Namespace field should be visible
  await expect(page.getByLabel(/namespace/i).first()).toBeVisible({ timeout: 5_000 });
});
