import { test, expect } from "@playwright/test";

test("dashboard renders with sidebar and table", async ({ page }) => {
  await page.goto("/");
  // Sidebar should be visible
  await expect(page.getByTestId("sidebar-phase-all")).toBeVisible();
  // New run button should be visible
  await expect(page.getByTestId("new-run-button")).toBeVisible();
  // Search input should be visible
  await expect(page.getByTestId("search-input")).toBeVisible();
});
