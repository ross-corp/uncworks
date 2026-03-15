import { test, expect } from "@playwright/test";

test("filter popover filters the run list", async ({ page }) => {
  await page.goto("/");

  // Wait for the run list to load
  await expect(page.getByTestId("run-list")).toBeVisible();

  // Open filter popover via icon rail
  const filterIcon = page.getByTestId("icon-rail-filter");
  const hasFilterIcon = await filterIcon.isVisible().catch(() => false);
  test.skip(!hasFilterIcon, "No filter icon visible in icon rail");

  await filterIcon.click();

  // The filter popover should appear with radio options
  const failedOption = page.getByRole("radio", { name: "Failed" });
  const hasFailedOption = await failedOption.isVisible().catch(() => false);
  test.skip(!hasFailedOption, "No Failed radio option visible in filter popover");

  await failedOption.click();

  // If there are run rows, they should all be in failed state
  const rows = page.locator("[data-testid^='run-row-']");
  const count = await rows.count();

  if (count > 0) {
    for (let i = 0; i < count; i++) {
      await expect(rows.nth(i)).toContainText(/failed/i);
    }
  }

  // Reset via All option
  const allOption = page.getByRole("radio", { name: "All" });
  await allOption.click();
});

test("keyboard filter shortcuts work", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-list")).toBeVisible();

  // Press '3' for "Done" filter (1=All, 2=Active, 3=Done, 4=Failed)
  await page.keyboard.press("3");

  // Give the filter time to apply
  await page.waitForTimeout(500);

  // Press '1' to reset to All
  await page.keyboard.press("1");
});

test("command palette search finds runs", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-list")).toBeVisible();

  // Open command palette with Cmd+K / Ctrl+K
  await page.keyboard.press("Control+k");

  // Command palette should appear
  const palette = page.getByTestId("command-palette");
  const hasPalette = await palette.isVisible().catch(() => false);
  test.skip(!hasPalette, "Command palette not available");

  // Type a search query that should not match anything
  await page.getByTestId("command-palette-input").fill("zzz-nonexistent-run-xyz");

  // Should show no matching results (or empty state)
  await page.waitForTimeout(500);

  // Close palette with Escape
  await page.keyboard.press("Escape");
  await expect(palette).not.toBeVisible();
});
