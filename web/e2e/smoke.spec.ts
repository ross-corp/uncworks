import { test, expect } from "@playwright/test";

test("dashboard renders with sidebar and run feed", async ({ page }) => {
  await page.goto("/");
  // Sidebar new-run button should be visible
  await expect(page.getByTestId("sidebar-new-run")).toBeVisible();
  // Search input should be visible
  await expect(page.getByTestId("search-input")).toBeVisible();
  // Run feed should be visible
  await expect(page.getByTestId("run-feed")).toBeVisible();
});
