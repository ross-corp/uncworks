// web/e2e-app/custom-select.spec.ts — Verify CustomSelect works in WKWebView.
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

test("CustomSelect opens and selects a value in WKWebView", async () => {
  const page = browser.contexts()[0].pages()[0];

  // Navigate to Settings where CustomSelect dropdowns exist
  await page.getByRole("link", { name: /settings/i }).first().click();
  await page.waitForTimeout(500);

  // Find a CustomSelect (details element with a summary)
  const customSelect = page.locator("details").first();
  await expect(customSelect).toBeVisible({ timeout: 5_000 });

  // Open it
  await customSelect.locator("summary").click();

  // An option list should appear inside
  const optionList = customSelect.locator("ul, [role=listbox], .cs-options");
  await expect(optionList).toBeVisible({ timeout: 2_000 });

  // Click the first option
  const firstOption = optionList.locator("li, [role=option]").first();
  await firstOption.click();

  // Dropdown should close
  await expect(optionList).not.toBeVisible({ timeout: 2_000 });
});
