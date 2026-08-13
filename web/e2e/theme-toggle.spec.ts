import { test, expect } from "@playwright/test";
import { mockRuns } from "./helpers";

// The sun and moon footer button is gone. Theme is chosen in Settings, with
// System, Light, and Dark buttons, so that is what this checks.

test.describe("Theme", () => {
  test.beforeEach(async ({ page }) => {
    await mockRuns(page, []);
    await page.goto("/settings");
  });

  test("Settings offers the three modes", async ({ page }) => {
    for (const mode of ["System", "Light", "Dark"]) {
      await expect(page.getByRole("button", { name: mode, exact: true })).toBeVisible();
    }
  });

  test("choosing Dark and Light switches the root class", async ({ page }) => {
    const isDark = () => page.locator("html").evaluate((el) => el.classList.contains("dark"));

    await page.getByRole("button", { name: "Dark", exact: true }).click();
    await expect.poll(isDark).toBe(true);

    await page.getByRole("button", { name: "Light", exact: true }).click();
    await expect.poll(isDark).toBe(false);
  });

  test("the choice survives a reload", async ({ page }) => {
    await page.getByRole("button", { name: "Dark", exact: true }).click();
    await expect.poll(() => page.locator("html").evaluate((el) => el.classList.contains("dark"))).toBe(true);

    await page.reload();
    await expect.poll(() => page.locator("html").evaluate((el) => el.classList.contains("dark"))).toBe(true);
  });
});
