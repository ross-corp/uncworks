import { test, expect } from "@playwright/test";

test.describe("Theme Toggle", () => {
  test("toggle button exists in footer", async ({ page }) => {
    await page.goto("/");
    // Look for sun or moon character in a button
    const toggle = page.locator("button").filter({ hasText: /[\u2600\u263E]/ });
    await expect(toggle).toBeVisible({ timeout: 5000 });
  });

  test("clicking toggle switches theme", async ({ page }) => {
    await page.goto("/");
    const toggle = page.locator("button").filter({ hasText: /[\u2600\u263E]/ });

    // Get initial dark class state
    const initialHasDark = await page.locator("html").evaluate(el => el.classList.contains("dark"));

    // Click toggle
    await toggle.click();

    // Verify class changed
    const afterHasDark = await page.locator("html").evaluate(el => el.classList.contains("dark"));
    expect(afterHasDark).not.toBe(initialHasDark);

    // Click again to restore
    await toggle.click();
    const restoredHasDark = await page.locator("html").evaluate(el => el.classList.contains("dark"));
    expect(restoredHasDark).toBe(initialHasDark);
  });
});
