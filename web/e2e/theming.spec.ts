import { test, expect } from "@playwright/test";

function mockApis(page: import("@playwright/test").Page) {
  return page.route("**/api/v1/runs", (route) => {
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify([]),
    });
  });
}

test.describe("Theming", () => {
  test("page loads with dark mode by default (system prefers-color-scheme: dark)", async ({ page }) => {
    // Emulate dark color scheme (system preference)
    await page.emulateMedia({ colorScheme: "dark" });
    await mockApis(page);
    await page.goto("/");

    // The <html> element should have the "dark" class
    const htmlElement = page.locator("html");
    await expect(htmlElement).toHaveClass(/dark/);
  });

  test("page loads with light mode when system prefers light", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "light" });
    await mockApis(page);
    await page.goto("/");

    // Without explicit mode set, it follows system — should not have "dark" class
    const htmlElement = page.locator("html");
    await expect(htmlElement).not.toHaveClass(/dark/);
  });

  test("theme color preference persists in localStorage", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "dark" });
    await mockApis(page);
    await page.goto("/");

    // Set a theme via localStorage directly (simulating what the app does)
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-color", "blue");
    });

    // Reload and verify the stored value persists
    await page.reload();

    const storedTheme = await page.evaluate(() => localStorage.getItem("aot-theme-color"));
    expect(storedTheme).toBe("blue");
  });

  test("theme mode preference persists in localStorage", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "light" });
    await mockApis(page);
    await page.goto("/");

    // Set mode to dark explicitly via localStorage
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-mode", "dark");
    });

    await page.reload();

    const storedMode = await page.evaluate(() => localStorage.getItem("aot-theme-mode"));
    expect(storedMode).toBe("dark");

    // After reload, app should apply the stored mode
    const htmlElement = page.locator("html");
    await expect(htmlElement).toHaveClass(/dark/);
  });

  test("stored dark mode overrides system light preference", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "light" });
    await mockApis(page);

    // Pre-set the mode before navigating
    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-mode", "dark");
    });
    await page.reload();

    const htmlElement = page.locator("html");
    await expect(htmlElement).toHaveClass(/dark/);
  });

  test("stored light mode overrides system dark preference", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "dark" });
    await mockApis(page);

    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-mode", "light");
    });
    await page.reload();

    const htmlElement = page.locator("html");
    await expect(htmlElement).not.toHaveClass(/dark/);
  });

  test("theme color class is applied to html element", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "dark" });
    await mockApis(page);

    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-color", "rose");
    });
    await page.reload();

    const htmlElement = page.locator("html");
    await expect(htmlElement).toHaveClass(/theme-rose/);
  });

  test("zinc theme does not add a theme class (it is the default)", async ({ page }) => {
    await page.emulateMedia({ colorScheme: "dark" });
    await mockApis(page);

    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("aot-theme-color", "zinc");
    });
    await page.reload();

    const htmlElement = page.locator("html");
    // zinc is the default and should not add a theme-zinc class
    await expect(htmlElement).not.toHaveClass(/theme-zinc/);
  });
});
