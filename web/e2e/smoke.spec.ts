// e2e/smoke.spec.ts — End-to-end smoke tests for key uncworks web UI flows.
// Assumes the API is running at http://localhost:50055 (proxied via vite dev server).
// Run with: cd web && npm run test:e2e
import { test, expect } from "@playwright/test";

// ---------------------------------------------------------------------------
// Minimal API mocks so the layout does not error when the cluster is down.
// ---------------------------------------------------------------------------
async function mockApis(page: import("@playwright/test").Page) {
  await Promise.all([
    page.route("**/api/v1/runs**", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    }),
    page.route("**/api/v1/projects**", (route) => {
      if (route.request().method() === "POST") {
        const body = JSON.parse(route.request().postData() ?? "{}");
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            name: body.name ?? "new-project",
            displayName: body.name ?? "new-project",
            description: "",
            repos: [],
            configRepoReady: false,
            runCount: 0,
            lastRunId: "",
            totalCost: "",
            createdAt: new Date().toISOString(),
          }),
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify([]),
        });
      }
    }),
  ]);
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------
test.describe("Projects", () => {
  test("can navigate to Projects list", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=Projects")).toBeVisible();
  });

  test("can create a project with a kebab-case name", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    // Open create form
    const newBtn = page.locator("button", { hasText: "+ new project" }).first();
    await expect(newBtn).toBeVisible();
    await newBtn.click();

    // Input should appear
    const nameInput = page.locator("input[placeholder='project-name (kebab-case)']");
    await expect(nameInput).toBeVisible();

    // Type a kebab-case name
    await nameInput.fill("my-test-project");
    await expect(nameInput).toHaveValue("my-test-project");

    // Submit
    const createBtn = page.locator("button", { hasText: "Create" });
    await expect(createBtn).toBeEnabled();
    await createBtn.click();

    // Form should close after creation
    await expect(nameInput).not.toBeVisible({ timeout: 5000 });
  });
});

// ---------------------------------------------------------------------------
// New Run form
// ---------------------------------------------------------------------------
test.describe("New Run form", () => {
  test("can navigate to New Run form", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    await expect(page.locator("text=New Run")).toBeVisible();
  });

  test("New Run form shows prompt field", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const textarea = page.locator("textarea[placeholder='What should the agent do?']");
    await expect(textarea).toBeVisible();
  });

  test("New Run form has no repositories field visible by default", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    // There should be no dedicated "Repositories" section label by default
    await expect(page.locator("text=Repositories")).not.toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------
test.describe("Settings", () => {
  test("can navigate to Settings", async ({ page }) => {
    await mockApis(page);
    await page.goto("/settings");

    await expect(page.locator("text=Settings")).toBeVisible();
  });

  test("Settings page loads with Appearance section", async ({ page }) => {
    await mockApis(page);
    await page.goto("/settings");

    // Appearance section heading
    await expect(page.locator("text=Appearance")).toBeVisible();
  });

  test("Settings page loads with GitHub section", async ({ page }) => {
    await mockApis(page);
    await page.goto("/settings");

    await expect(page.locator("text=GitHub")).toBeVisible();
  });
});
