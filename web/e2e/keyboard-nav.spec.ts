import { test, expect } from "@playwright/test";

test("j/k keyboard navigation moves selection through run cards", async ({ page }) => {
  await page.goto("/");

  // Wait for run feed to load with cards
  await expect(page.getByTestId("run-feed")).toBeVisible();
  const cards = page.locator("[data-testid^='run-card-']");
  const count = await cards.count();
  test.skip(count < 2, "Need at least 2 runs to test j/k navigation");

  // Press 'j' to select the first card
  await page.keyboard.press("j");

  // The first card should now have a left border indicating selection
  const firstCard = cards.first();
  const firstBorderWidth = await firstCard.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(firstBorderWidth)).toBeGreaterThan(0);

  // Press 'j' again to move to second card
  await page.keyboard.press("j");

  // Second card should now be selected
  const secondCard = cards.nth(1);
  const secondBorderWidth = await secondCard.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(secondBorderWidth)).toBeGreaterThan(0);

  // Press 'k' to go back to first
  await page.keyboard.press("k");
  const firstBorderWidthAgain = await firstCard.evaluate(
    (el) => getComputedStyle(el).borderLeftWidth
  );
  expect(parseInt(firstBorderWidthAgain)).toBeGreaterThan(0);
});

test("Enter opens detail and Escape closes it", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-feed")).toBeVisible();
  const cards = page.locator("[data-testid^='run-card-']");
  const count = await cards.count();
  test.skip(count === 0, "No runs available to test Enter/Escape");

  // Select first card with 'j'
  await page.keyboard.press("j");

  // Press Enter to open detail
  await page.keyboard.press("Enter");

  // RunDetail should now be visible (clicking the card opens it)
  // Note: the keyboard Enter triggers click on the selected run
  await expect(page.getByTestId("run-detail")).toBeVisible({ timeout: 5000 });

  // Press Escape to close
  await page.keyboard.press("Escape");

  // Should be back to the feed
  await expect(page.getByTestId("run-feed")).toBeVisible({ timeout: 5000 });
});

test("/ focuses search input", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("search-input")).toBeVisible();

  // Press '/' to focus search
  await page.keyboard.press("/");

  // The search input should be focused
  const isFocused = await page.getByTestId("search-input").evaluate(
    (el) => document.activeElement === el
  );
  expect(isFocused).toBe(true);
});

test("keyboard shortcuts are disabled in inputs", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("search-input")).toBeVisible();

  // Focus the search input
  await page.getByTestId("search-input").click();

  // Type 'j' — should type into the input, not navigate
  await page.keyboard.type("j");

  // The search input should contain 'j'
  await expect(page.getByTestId("search-input")).toHaveValue("j");

  // Clean up
  await page.getByTestId("search-input").fill("");
});
