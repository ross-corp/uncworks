import { test, expect } from "@playwright/test";

// Mock the runs API so the layout doesn't error
function mockApis(page: import("@playwright/test").Page) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (route.request().method() === "GET") {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify([]),
        });
      } else {
        // POST — create run
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            id: "new-run-1",
            name: "created-run",
            spec: {
              backend: "Pod",
              repos: [{ url: "https://github.com/org/repo", branch: "main" }],
              prompt: "test prompt",
              ttlSeconds: 900,
              modelTier: "default",
            },
            status: { phase: "Pending", message: "" },
            createdAt: new Date().toISOString(),
          }),
        });
      }
    }),
  ]);
}

test.describe("New Run View", () => {
  test("/new page loads with prompt textarea", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    // Header should say "New Run"
    await expect(page.locator("text=New Run")).toBeVisible();

    // Prompt textarea should be present and focused
    const textarea = page.locator("textarea[placeholder='What should the agent do?']");
    await expect(textarea).toBeVisible();
  });

  test("can type a prompt", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const textarea = page.locator("textarea[placeholder='What should the agent do?']");
    await textarea.fill("Fix all the broken tests in the repo");

    await expect(textarea).toHaveValue("Fix all the broken tests in the repo");
  });

  test("Prompt/Spec tab toggle works", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    // Prompt tab is active by default
    const promptTab = page.locator("button", { hasText: "Prompt" });
    const specTab = page.locator("button", { hasText: "Spec" });

    await expect(promptTab).toHaveClass(/bg-accent/);
    await expect(specTab).not.toHaveClass(/bg-accent/);

    // Spec textarea should not be visible in prompt mode
    const specTextarea = page.locator("textarea[placeholder='Paste your spec (markdown)...']");
    await expect(specTextarea).not.toBeVisible();

    // Click Spec tab
    await specTab.click();
    await expect(specTab).toHaveClass(/bg-accent/);
    await expect(promptTab).not.toHaveClass(/bg-accent/);

    // Spec textarea should now be visible
    await expect(specTextarea).toBeVisible();

    // Switch back to Prompt
    await promptTab.click();
    await expect(promptTab).toHaveClass(/bg-accent/);
    await expect(specTextarea).not.toBeVisible();
  });

  test("spec mode shows both prompt and spec textareas", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const specTab = page.locator("button", { hasText: "Spec" });
    await specTab.click();

    // Both textareas should be visible
    const promptTextarea = page.locator("textarea[placeholder='What should the agent do?']");
    const specTextarea = page.locator("textarea[placeholder='Paste your spec (markdown)...']");

    await expect(promptTextarea).toBeVisible();
    await expect(specTextarea).toBeVisible();

    // Can type in both
    await promptTextarea.fill("Run the spec");
    await specTextarea.fill("## Task\n- Step 1\n- Step 2");

    await expect(promptTextarea).toHaveValue("Run the spec");
    await expect(specTextarea).toHaveValue("## Task\n- Step 1\n- Step 2");
  });

  test("Cancel button navigates back to /", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const cancelBtn = page.locator("button", { hasText: "Cancel" });
    await expect(cancelBtn).toBeVisible();

    await cancelBtn.click();
    await expect(page).toHaveURL(/\/$/);
  });

  test("Run button is disabled when prompt is empty", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeDisabled();
  });

  test("Run button is enabled when prompt has text", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const textarea = page.locator("textarea[placeholder='What should the agent do?']");
    await textarea.fill("Do something useful");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeEnabled();
  });

  test("Ctrl+Enter submits the form", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const textarea = page.locator("textarea[placeholder='What should the agent do?']");
    await textarea.fill("Create a new feature");

    // Ctrl+Enter to submit
    await textarea.press("Control+Enter");

    // Should navigate to the run detail page after creation
    await expect(page).toHaveURL(/\/run\/new-run-1/);
  });

  test("repository input fields are present", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    // Repo URL and branch inputs
    const repoInput = page.locator("input[placeholder='https://github.com/org/repo']");
    const branchInput = page.locator("input[placeholder='main']");

    await expect(repoInput).toBeVisible();
    await expect(branchInput).toBeVisible();

    // Default repo should be pre-filled
    await expect(repoInput).toHaveValue("https://github.com/roshbhatia/neph.nvim");
    await expect(branchInput).toHaveValue("main");
  });

  test("config summary shows default settings", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    await expect(page.locator("text=qwen3:8b")).toBeVisible();
  });
});
