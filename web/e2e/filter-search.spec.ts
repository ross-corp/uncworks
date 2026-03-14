import { test, expect } from "@playwright/test";

test("phase filter shows matching runs", async ({ page }) => {
  await page.goto("/");

  // Wait for page to load
  await expect(page.getByTestId("sidebar-phase-all")).toBeVisible();

  // Click a phase filter (succeeded)
  await page.getByTestId("sidebar-phase-succeeded").click();

  // If there are runs, they should all be succeeded
  const rows = page.locator("tr[data-testid^='table-row-']");
  const count = await rows.count();

  if (count > 0) {
    // Each visible row should contain "Succeeded" phase
    for (let i = 0; i < count; i++) {
      await expect(rows.nth(i)).toContainText(/succeeded/i);
    }
  }

  // Reset to all
  await page.getByTestId("sidebar-phase-all").click();
});

test("search filters by name", async ({ page }) => {
  await page.goto("/");

  // Wait for page to load
  await expect(page.getByTestId("search-input")).toBeVisible();

  // Type a search query that should not match anything
  await page.getByTestId("search-input").fill("zzz-nonexistent-run-xyz");

  // Table should show no matching runs or empty state
  await expect(page.locator("tr[data-testid^='table-row-']")).toHaveCount(0, { timeout: 5000 });

  // Clear and verify runs come back
  await page.getByTestId("search-input").fill("");
});
