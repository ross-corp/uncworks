import { test, expect } from "@playwright/test";

test.describe("Navigation smoke tests", () => {
  test("/ shows Runs heading", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Runs")).toBeVisible();
  });

  test("/templates shows Templates heading", async ({ page }) => {
    await page.goto("/templates");
    await expect(page.getByText("Templates")).toBeVisible();
  });

  test("/chains shows Chains heading", async ({ page }) => {
    await page.goto("/chains");
    await expect(page.getByText("Chains")).toBeVisible();
  });

  test("/schedules shows Schedules heading", async ({ page }) => {
    await page.goto("/schedules");
    await expect(page.getByText("Schedules")).toBeVisible();
  });

  test("/projects shows Projects heading", async ({ page }) => {
    await page.goto("/projects");
    await expect(page.getByText("Projects")).toBeVisible();
  });

  test("/chains/new has a name input", async ({ page }) => {
    await page.goto("/chains/new");
    // The chain new form has an input with placeholder "my-chain"
    await expect(page.locator('input[placeholder="my-chain"]')).toBeVisible();
  });

  test("/schedules/new has a cron input", async ({ page }) => {
    await page.goto("/schedules/new");
    // The schedule new form has a cron Expression input
    await expect(page.getByText("Cron Expression")).toBeVisible();
  });

  test("/templates/new has a name input", async ({ page }) => {
    await page.goto("/templates/new");
    // The template new form has an input with placeholder "my-template"
    await expect(page.locator('input[placeholder="my-template"]')).toBeVisible();
  });
});
