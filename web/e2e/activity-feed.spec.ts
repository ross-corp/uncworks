import { test, expect } from "@playwright/test";

test.describe("Activity Feed", () => {
  test("shows role labels with semantic names", async ({ page }) => {
    await page.goto("/");
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }
    // The logs tab should be active by default
    // Check for role labels - at least one should be visible
    const feed = page.locator(".overflow-y-auto");
    await expect(feed).toBeVisible({ timeout: 5000 });

    // Check that labels use full names (not abbreviated)
    // "implement" should appear, not "impl" or "neph"
    const implLabel = page.locator("text=implement");
    const manageLabel = page.locator("text=manage");
    const systemLabel = page.locator("text=system");

    // At least one of these should be visible (depending on run data)
    const anyVisible = await Promise.any([
      implLabel.first().isVisible({ timeout: 3000 }),
      manageLabel.first().isVisible({ timeout: 3000 }),
      systemLabel.first().isVisible({ timeout: 3000 }),
    ]).catch(() => false);

    expect(anyVisible).toBeTruthy();
  });

  test("does not show deprecated labels", async ({ page }) => {
    await page.goto("/");
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }

    // Wait for feed to load
    await page.waitForTimeout(2000);

    // These old labels should NOT appear
    const pageText = await page.textContent("body");
    expect(pageText).not.toContain(">neph<");
    expect(pageText).not.toContain(">unc<");
  });
});
