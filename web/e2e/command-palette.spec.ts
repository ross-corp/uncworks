import { test, expect } from "@playwright/test";

const SAMPLE_RUNS = [
  {
    id: "cp-run-1",
    name: "palette-run-alpha",
    spec: {
      backend: "Pod",
      repos: [{ url: "https://github.com/org/repo", branch: "main" }],
      prompt: "Test command palette",
      ttlSeconds: 900,
      modelTier: "default",
      displayName: "Palette Alpha",
    },
    status: { phase: "Running", message: "" },
    createdAt: new Date().toISOString(),
  },
  {
    id: "cp-run-2",
    name: "palette-run-beta",
    spec: {
      backend: "Pod",
      repos: [{ url: "https://github.com/org/repo2", branch: "main" }],
      prompt: "Another run",
      ttlSeconds: 900,
      modelTier: "default",
      displayName: "Palette Beta",
    },
    status: { phase: "Succeeded", message: "" },
    createdAt: new Date().toISOString(),
  },
];

function mockApis(page: import("@playwright/test").Page) {
  return page.route("**/api/v1/runs", (route) => {
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(SAMPLE_RUNS),
    });
  });
}

test.describe("Command Palette", () => {
  test("Ctrl+K opens the command palette", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    // Palette should not be visible initially
    const paletteInput = page.locator("input[placeholder='Type a command or search...']");
    await expect(paletteInput).not.toBeVisible();

    // Press Ctrl+K to open
    await page.keyboard.press("Control+k");

    await expect(paletteInput).toBeVisible();
  });

  test("typing filters results in the palette", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    // Open palette
    await page.keyboard.press("Control+k");
    const paletteInput = page.locator("input[placeholder='Type a command or search...']");
    await expect(paletteInput).toBeVisible();

    // Navigation items should be visible initially
    await expect(page.locator("[cmdk-item]", { hasText: "Go to Runs" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "New Run" })).toBeVisible();

    // Type to filter
    await paletteInput.fill("New Run");
    await expect(page.locator("[cmdk-item]", { hasText: "New Run" })).toBeVisible();

    // Type something that matches nothing
    await paletteInput.fill("xyznonexistent");
    await expect(page.locator("text=No results found.")).toBeVisible();
  });

  test("Esc closes the command palette", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    // Open palette
    await page.keyboard.press("Control+k");
    const paletteInput = page.locator("input[placeholder='Type a command or search...']");
    await expect(paletteInput).toBeVisible();

    // Close with Esc
    await page.keyboard.press("Escape");
    await expect(paletteInput).not.toBeVisible();
  });

  test("Ctrl+K toggles the palette open and closed", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    const paletteInput = page.locator("input[placeholder='Type a command or search...']");

    // Open
    await page.keyboard.press("Control+k");
    await expect(paletteInput).toBeVisible();

    // Close with Ctrl+K again
    await page.keyboard.press("Control+k");
    await expect(paletteInput).not.toBeVisible();
  });

  test("palette shows navigation group", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    await page.keyboard.press("Control+k");

    await expect(page.locator("[cmdk-group-heading]", { hasText: "Navigation" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "Go to Runs" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "New Run" })).toBeVisible();
  });

  test("palette shows theme group", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    await page.keyboard.press("Control+k");

    await expect(page.locator("[cmdk-group-heading]", { hasText: "Theme" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "Toggle dark mode" })).toBeVisible();
  });

  test("palette shows runs group when runs exist", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    // Wait for runs to load in layout
    await page.waitForResponse("**/api/v1/runs");

    await page.keyboard.press("Control+k");

    await expect(page.locator("[cmdk-group-heading]", { hasText: "Runs" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "Palette Alpha" })).toBeVisible();
    await expect(page.locator("[cmdk-item]", { hasText: "Palette Beta" })).toBeVisible();
  });

  test("selecting 'Go to Runs' navigates to /", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    await page.keyboard.press("Control+k");

    const item = page.locator("[cmdk-item]", { hasText: "Go to Runs" });
    await item.click();

    await expect(page).toHaveURL(/\/$/);
  });

  test("selecting 'New Run' navigates to /new", async ({ page }) => {
    await mockApis(page);
    await page.goto("/");

    await page.keyboard.press("Control+k");

    const item = page.locator("[cmdk-item]", { hasText: "New Run" });
    await item.click();

    await expect(page).toHaveURL(/\/new/);
  });
});
