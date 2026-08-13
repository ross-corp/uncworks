import { test, expect } from "@playwright/test";
import { heading, mockRuns } from "./helpers";

// Every list view now carries a real heading. `getByText("Runs")` matched the
// sidebar link, the heading, and any row that said Runs, which reported as a
// strict-mode violation rather than as the thing the test meant to check.

test.describe("Navigation smoke tests", () => {
  test.beforeEach(async ({ page }) => {
    await mockRuns(page);
  });

  const listViews: Array<[string, string]> = [
    ["/", "Runs"],
    ["/templates", "Templates"],
    ["/chains", "Chains"],
    ["/schedules", "Schedules"],
    ["/projects", "Projects"],
  ];

  for (const [path, title] of listViews) {
    test(`${path} shows the ${title} heading`, async ({ page }) => {
      await page.goto(path);
      await expect(heading(page, title)).toBeVisible();
    });
  }

  test("/chains/new has a name input", async ({ page }) => {
    await page.goto("/chains/new");
    await expect(page.locator('input[placeholder="my-chain"]')).toBeVisible();
  });

  test("/schedules/new has a cron input", async ({ page }) => {
    await page.goto("/schedules/new");
    await expect(page.locator('input[placeholder="0 * * * *"]')).toBeVisible();
  });

  test("/templates/new has a name input", async ({ page }) => {
    await page.goto("/templates/new");
    await expect(page.locator('input[placeholder="my-template"]')).toBeVisible();
  });
});
