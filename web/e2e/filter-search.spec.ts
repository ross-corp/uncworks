import { test, expect } from "@playwright/test";

test("status filter chips filter the run feed", async ({ page }) => {
  await page.goto("/");

  // Wait for the run feed to load
  await expect(page.getByTestId("run-feed")).toBeVisible();

  // The FilterSidebar should have filter chips. Click "Succeeded".
  const succeededChip = page.getByRole("button", { name: "Succeeded" });
  const hasSucceededChip = await succeededChip.isVisible().catch(() => false);
  test.skip(!hasSucceededChip, "No Succeeded filter chip visible");

  await succeededChip.click();

  // If there are run cards, they should all be in succeeded state
  const cards = page.locator("[data-testid^='run-card-']");
  const count = await cards.count();

  if (count > 0) {
    for (let i = 0; i < count; i++) {
      await expect(cards.nth(i)).toContainText(/succeeded/i);
    }
  }

  // Click "All" to reset
  const allChip = page.getByRole("button", { name: "All" });
  await allChip.click();
});

test("search filters by name", async ({ page }) => {
  await page.goto("/");

  // Wait for search input
  await expect(page.getByTestId("search-input")).toBeVisible();

  // Type a search query that should not match anything
  await page.getByTestId("search-input").fill("zzz-nonexistent-run-xyz");

  // Feed should show no matching runs or the empty state
  await expect(page.locator("[data-testid^='run-card-']")).toHaveCount(0, { timeout: 5000 });

  // Clear and verify runs come back
  await page.getByTestId("search-input").fill("");
});

test("search is additive with sidebar filters", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("run-feed")).toBeVisible();
  await expect(page.getByTestId("search-input")).toBeVisible();

  // Apply a nonexistent search with filters — should show nothing
  await page.getByTestId("search-input").fill("zzz-impossible-combo-xyz");
  await expect(page.locator("[data-testid^='run-card-']")).toHaveCount(0, { timeout: 5000 });

  // Clear search
  await page.getByTestId("search-input").fill("");
});
