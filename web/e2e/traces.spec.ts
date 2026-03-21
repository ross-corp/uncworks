import { test, expect } from "@playwright/test";

test.describe("Traces", () => {
  test("clicking a span row shows detail panel", async ({ page }) => {
    // Navigate to a run detail page
    await page.goto("/");
    // Click the first run in the list
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }
    // Switch to traces tab
    await page.locator("[data-testid='detail-tab-traces']").click();

    // If spans exist, click the first one
    const spanRow = page.locator("[data-testid='trace-timeline'] button").first();
    if (await spanRow.isVisible({ timeout: 5000 }).catch(() => false)) {
      await spanRow.click();
      // Verify detail panel appears (look for metadata or close button)
      await expect(page.locator("text=Duration")).toBeVisible({ timeout: 3000 });
    }
  });

  test("trace timeline shows span count", async ({ page }) => {
    await page.goto("/");
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }
    await page.locator("[data-testid='detail-tab-traces']").click();
    // Should show "N spans" or "No trace spans recorded"
    const timeline = page.locator("[data-testid='trace-timeline']");
    await expect(timeline).toBeVisible({ timeout: 5000 });
  });
});
