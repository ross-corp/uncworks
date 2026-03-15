import { test, expect } from "@playwright/test";

test("theme toggle switches between dark and light mode", async ({ page }) => {
  await page.goto("/");

  // By default, dark mode should be active (or the html element has .dark)
  const htmlEl = page.locator("html");

  // Click the theme toggle button
  const toggle = page.getByRole("button", { name: "Toggle theme" });
  await expect(toggle).toBeVisible();

  // Get initial theme
  const initialHasDark = await htmlEl.evaluate((el) => el.classList.contains("dark"));

  // Click toggle
  await toggle.click();

  // Theme should have changed
  const afterToggle = await htmlEl.evaluate((el) => el.classList.contains("dark"));
  expect(afterToggle).toBe(!initialHasDark);

  // Toggle back
  await toggle.click();
  const afterSecondToggle = await htmlEl.evaluate((el) => el.classList.contains("dark"));
  expect(afterSecondToggle).toBe(initialHasDark);
});

test("theme persists across reload", async ({ page }) => {
  await page.goto("/");

  const htmlEl = page.locator("html");
  const toggle = page.getByRole("button", { name: "Toggle theme" });
  await expect(toggle).toBeVisible();

  // Set to light mode
  const hasDark = await htmlEl.evaluate((el) => el.classList.contains("dark"));
  if (hasDark) {
    await toggle.click();
  }

  // Verify we're in light mode
  await expect(htmlEl).not.toHaveClass(/dark/);

  // Reload
  await page.reload();

  // Should still be light mode
  await expect(htmlEl).not.toHaveClass(/dark/);

  // Restore dark mode
  const toggleAfterReload = page.getByRole("button", { name: "Toggle theme" });
  await toggleAfterReload.click();
  await expect(htmlEl).toHaveClass(/dark/);
});
