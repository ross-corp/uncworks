import { test, expect } from "@playwright/test";

test("dashboard renders with icon rail and run list", async ({ page }) => {
  await page.goto("/");
  // Icon rail new-run button should be visible
  await expect(page.getByTestId("icon-rail-new-run")).toBeVisible();
  // Run list (dense table) should be visible
  await expect(page.getByTestId("run-list")).toBeVisible();
});
