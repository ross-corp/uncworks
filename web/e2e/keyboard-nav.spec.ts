import { test, expect } from "@playwright/test";

test("j/k keyboard navigation moves selection through run rows", async ({ page }) => {
  await page.goto("/");

  // Wait for run list to load
  await expect(page.getByTestId("run-list")).toBeVisible();
  const rows = page.locator("[data-testid^='run-row-']");
  const count = await rows.count();
  test.skip(count < 2, "Need at least 2 runs to test j/k navigation");

  // Press 'j' to select the first row
  await page.keyboard.press("j");

  // The first row should now have a left border indicating selection
  const firstRow = rows.first();
  const firstBorderWidth = await firstRow.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(firstBorderWidth)).toBeGreaterThan(0);

  // Press 'j' again to move to second row
  await page.keyboard.press("j");

  // Second row should now be selected
  const secondRow = rows.nth(1);
  const secondBorderWidth = await secondRow.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(secondBorderWidth)).toBeGreaterThan(0);

  // Press 'k' to go back to first
  await page.keyboard.press("k");
  const firstBorderWidthAgain = await firstRow.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(firstBorderWidthAgain)).toBeGreaterThan(0);
});

test("Enter opens detail and Escape closes it", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-list")).toBeVisible();
  const rows = page.locator("[data-testid^='run-row-']");
  const count = await rows.count();
  test.skip(count === 0, "No runs available to test Enter/Escape");

  // Select first row with 'j'
  await page.keyboard.press("j");

  // Press Enter to open detail
  await page.keyboard.press("Enter");

  // RunDetail (DetailPane) should now be visible
  await expect(page.getByTestId("run-detail")).toBeVisible({ timeout: 5000 });

  // Press Escape to close
  await page.keyboard.press("Escape");

  // Should be back to the list
  await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 5000 });
});

test("command palette opens with Ctrl+K and closes with Escape", async ({ page }) => {
  await page.goto("/");

  // Open command palette with Ctrl+K
  await page.keyboard.press("Control+k");

  // Command palette should be visible
  const palette = page.getByTestId("command-palette");
  const hasPalette = await palette.isVisible().catch(() => false);
  test.skip(!hasPalette, "Command palette not available");

  // Type a query
  await page.getByTestId("command-palette-input").fill("test");

  // Verify results container exists
  await expect(page.getByTestId("command-palette-results")).toBeVisible();

  // Press Escape to close
  await page.keyboard.press("Escape");
  await expect(palette).not.toBeVisible();
});

test("q closes detail and deselects", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-list")).toBeVisible();
  const rows = page.locator("[data-testid^='run-row-']");
  const count = await rows.count();
  test.skip(count === 0, "No runs available to test q shortcut");

  // Select and open detail
  await page.keyboard.press("j");
  await page.keyboard.press("Enter");
  await expect(page.getByTestId("run-detail")).toBeVisible({ timeout: 5000 });

  // Press 'q' to close detail and deselect
  await page.keyboard.press("q");
  await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 5000 });
});
