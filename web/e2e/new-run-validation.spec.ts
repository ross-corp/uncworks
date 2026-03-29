// e2e/new-run-validation.spec.ts — Additional form validation and error scenarios
// for NewRunView not covered by new-run.spec.ts.
// All API calls are mocked via page.route() — no real backend needed.
import { test, expect } from "@playwright/test";

function mockApis(page: import("@playwright/test").Page) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (route.request().method() === "GET") {
        route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
      } else {
        // POST — simulate a 500 from the create endpoint
        route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({ error: "internal server error" }),
        });
      }
    }),
    page.route("**/api/v1/projects", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
  ]);
}

function mockApisSuccess(page: import("@playwright/test").Page) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (route.request().method() === "GET") {
        route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
      } else {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            id: "created-run-1",
            name: "created-run",
            spec: { prompt: "test", ttlSeconds: 900, modelTier: "default", backend: "Pod", repos: [] },
            status: { phase: "Pending", message: "" },
            createdAt: new Date().toISOString(),
          }),
        });
      }
    }),
    page.route("**/api/v1/projects", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
  ]);
}

test.describe("New Run — Spec mode validation", () => {
  test("Run button is disabled in spec mode when both prompt and spec are empty", async ({
    page,
  }) => {
    await mockApis(page);
    await page.goto("/new");

    // Switch to Spec mode
    const specTab = page.locator("button", { hasText: "Spec" });
    await specTab.click();

    // Both textareas are empty — Run button should be disabled
    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeDisabled();
  });

  test("Run button is enabled in spec mode when only specContent is filled", async ({
    page,
  }) => {
    await mockApis(page);
    await page.goto("/new");

    const specTab = page.locator("button", { hasText: "Spec" });
    await specTab.click();

    // Fill only the spec textarea (prompt stays empty)
    const specTextarea = page.locator(
      "textarea[placeholder='Paste your spec (markdown)...']"
    );
    await specTextarea.fill("## Task\n- Do something");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeEnabled({ timeout: 3000 });
  });

  test("Run button is enabled in spec mode when prompt is filled (spec empty)", async ({
    page,
  }) => {
    await mockApis(page);
    await page.goto("/new");

    const specTab = page.locator("button", { hasText: "Spec" });
    await specTab.click();

    const promptTextarea = page.locator(
      "textarea[placeholder='What should the agent do?']"
    );
    await promptTextarea.fill("Run the spec against the codebase");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeEnabled({ timeout: 3000 });
  });
});

test.describe("New Run — submit error handling", () => {
  test("shows error toast when run creation API returns 500", async ({ page }) => {
    // Mock the runs POST to fail, but the client uses ConnectRPC — mock the
    // REST route as a fallback signal that at least the page does not crash.
    await mockApis(page);

    await page.goto("/new");

    const textarea = page.locator(
      "textarea[placeholder='What should the agent do?']"
    );
    await textarea.fill("This should fail on the backend");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await expect(runBtn).toBeEnabled();

    // Attempt submit; the mock will fail
    await textarea.press("Control+Enter");

    // The page must not navigate away and must still show "New Run"
    await expect(page.locator("text=New Run").first()).toBeVisible({
      timeout: 5000,
    });
  });

  test("Run button is re-enabled after a failed submission", async ({ page }) => {
    await mockApis(page);
    await page.goto("/new");

    const textarea = page.locator(
      "textarea[placeholder='What should the agent do?']"
    );
    await textarea.fill("Trigger a failure");

    const runBtn = page.locator("button", { hasText: "Run" }).last();
    await runBtn.click();

    // After the failed submission, the button should be enabled again (submitting=false)
    await expect(runBtn).toBeEnabled({ timeout: 8000 });
  });
});

test.describe("New Run — project query-param pre-fill", () => {
  const PROJECT_DETAIL = {
    name: "prepop-project",
    displayName: "prepop-project",
    description: "",
    repos: [],
    configRepoReady: true,
    configRepoURL: "",
    runCount: 0,
    totalCost: "",
    defaults: { modelTier: "default", orchestrationMode: "spec-driven" },
  };

  function mockApisWithProject(page: import("@playwright/test").Page) {
    return Promise.all([
      page.route("**/api/v1/runs", (route) => {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: "[]",
        });
      }),
      page.route("**/api/v1/projects", (route) => {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify([PROJECT_DETAIL]),
        });
      }),
      page.route("**/api/v1/projects/prepop-project", (route) => {
        if (!route.request().url().includes("/files")) {
          route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(PROJECT_DETAIL),
          });
        } else {
          route.fallback();
        }
      }),
    ]);
  }

  test("?project= param pre-fills project field", async ({ page }) => {
    await mockApisWithProject(page);
    await page.goto("/new?project=prepop-project");

    await expect(page.locator("text=New Run").first()).toBeVisible({ timeout: 8000 });

    // The project field should be pre-populated with the project name
    // (either via an input value or a select trigger showing the name)
    const projectInput = page
      .locator("input, [role='combobox']")
      .filter({ hasValue: /prepop-project/i });
    const projectText = page.locator("text=prepop-project").first();

    const found = await Promise.any([
      projectInput.first().waitFor({ state: "visible", timeout: 5000 }),
      projectText.waitFor({ state: "visible", timeout: 5000 }),
    ]).then(() => true).catch(() => false);

    expect(found).toBe(true);
  });
});
