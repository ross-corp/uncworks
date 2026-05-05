// web/e2e-app/settings-persist.spec.ts — Settings persist across window hide/show.
import { test, expect } from "@playwright/test";
import { connectToApp } from "./helpers";
import type { Browser } from "@playwright/test";

const TEST_NAMESPACE = `e2e-test-${Date.now()}`;

let browser: Browser;

test.beforeAll(async () => {
  ({ browser } = await connectToApp());
});

test.afterAll(async () => {
  await browser.close();
});

test("namespace change persists after window hide and show", async () => {
  const page = browser.contexts()[0].pages()[0];

  // Go to settings
  await page.getByRole("link", { name: /settings/i }).first().click();

  // Change namespace
  const namespaceInput = page.getByLabel(/namespace/i).first();
  await expect(namespaceInput).toBeVisible({ timeout: 5_000 });
  await namespaceInput.fill(TEST_NAMESPACE);

  // Save
  const saveBtn = page.getByRole("button", { name: /save/i }).first();
  await expect(saveBtn).toBeEnabled({ timeout: 2_000 });
  await saveBtn.click();
  await page.waitForTimeout(500);

  // Simulate hide/show via JS (can't do native macOS hide from Playwright)
  // Instead verify the value persists by navigating away and back
  await page.getByRole("link", { name: /projects/i }).first().click();
  await page.waitForTimeout(300);
  await page.getByRole("link", { name: /settings/i }).first().click();

  // Value should still be present
  const namespaceAfter = page.getByLabel(/namespace/i).first();
  await expect(namespaceAfter).toHaveValue(TEST_NAMESPACE, { timeout: 5_000 });

  // Clean up — restore original namespace
  await namespaceAfter.fill("uncworks");
  await page.getByRole("button", { name: /save/i }).first().click();
});
