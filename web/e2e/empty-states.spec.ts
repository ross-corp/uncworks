// e2e/empty-states.spec.ts — Tests for empty list states across views.
// All API calls are mocked via page.route() — no real backend needed.
import { test, expect } from "@playwright/test";

// ---------------------------------------------------------------------------
// Run List empty state
// Already covered in run-list.spec.ts ("shows empty state when there are no runs").
// Kept here as a canonical reference alongside the other empty state tests.
// ---------------------------------------------------------------------------
test.describe("Run List — empty state", () => {
  test("shows 'No runs yet' when API returns empty array", async ({ page }) => {
    await page.route("**/api/v1/runs", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    });

    await page.goto("/");

    await expect(page.locator("text=No runs yet")).toBeVisible({ timeout: 8000 });
  });
});

// ---------------------------------------------------------------------------
// Projects List empty state
// ---------------------------------------------------------------------------
test.describe("Projects List — empty state", () => {
  function mockApis(page: import("@playwright/test").Page) {
    return Promise.all([
      page.route("**/api/v1/runs**", (route) => {
        route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
      }),
      page.route("**/api/v1/projects", (route) => {
        route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
      }),
    ]);
  }

  test("shows 'No projects yet' when API returns empty array", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=No projects yet")).toBeVisible({ timeout: 8000 });
  });

  test("empty state contains a '+ new project' call-to-action", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=No projects yet")).toBeVisible();

    // The empty state CTA button should be present
    const cta = page.locator("button", { hasText: "+ new project" });
    await expect(cta).toBeVisible({ timeout: 5000 });
  });

  test("clicking empty-state CTA opens create form", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=No projects yet")).toBeVisible();

    const cta = page.locator("button", { hasText: "+ new project" }).first();
    await cta.click();

    // Create input should appear after clicking the CTA
    const nameInput = page.locator("input[placeholder='project-name (kebab-case)']");
    await expect(nameInput).toBeVisible({ timeout: 5000 });
  });
});

// ---------------------------------------------------------------------------
// Project Detail — Runs tab empty state
// ---------------------------------------------------------------------------
const SAMPLE_PROJECT = {
  name: "test-project",
  displayName: "test-project",
  description: "",
  repos: [],
  configRepoReady: true,
  configRepoURL: "https://github.com/org/test-project-config",
  runCount: 0,
  totalCost: "",
};

function mockProjectDetailApis(page: import("@playwright/test").Page) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (
        route.request().url().includes("/logs") ||
        route.request().url().includes("/traces")
      )
        return;
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
    page.route("**/api/v1/projects/test-project", (route) => {
      if (
        route.request().method() === "GET" &&
        !route.request().url().includes("/files")
      ) {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(SAMPLE_PROJECT),
        });
      } else {
        route.fallback();
      }
    }),
    page.route("**/api/v1/projects/test-project/files", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
  ]);
}

test.describe("Project Detail — Runs tab empty state", () => {
  test("shows 'No runs yet' in Runs tab when there are no runs for the project", async ({
    page,
  }) => {
    await mockProjectDetailApis(page);
    await page.goto("/projects/test-project");

    // Wait for project to load
    await expect(page.locator("text=test-project").first()).toBeVisible({
      timeout: 8000,
    });

    // Switch to Runs tab
    const runsTab = page.locator("button", { hasText: "Runs" }).first();
    await expect(runsTab).toBeVisible();
    await runsTab.click();

    await expect(page.locator("text=No runs yet")).toBeVisible({ timeout: 5000 });
  });

  test("Runs tab empty state shows '+ New Run' link", async ({ page }) => {
    await mockProjectDetailApis(page);
    await page.goto("/projects/test-project");

    await expect(page.locator("text=test-project").first()).toBeVisible({
      timeout: 8000,
    });

    const runsTab = page.locator("button", { hasText: "Runs" }).first();
    await runsTab.click();

    await expect(page.locator("text=No runs yet")).toBeVisible({ timeout: 5000 });
    await expect(page.locator("text=+ New Run")).toBeVisible({ timeout: 3000 });
  });
});

// ---------------------------------------------------------------------------
// Project Detail — Specs tab empty state
// ---------------------------------------------------------------------------
test.describe("Project Detail — Specs tab empty state", () => {
  test("shows 'No specs yet' when project has no spec files", async ({ page }) => {
    await mockProjectDetailApis(page);
    await page.goto("/projects/test-project");

    await expect(page.locator("text=test-project").first()).toBeVisible({
      timeout: 8000,
    });

    // Specs tab should be active by default
    await expect(page.locator("text=No specs yet")).toBeVisible({ timeout: 5000 });
  });

  test("Specs tab shows 'Select a file to view' placeholder when no file selected", async ({
    page,
  }) => {
    await mockProjectDetailApis(page);
    await page.goto("/projects/test-project");

    await expect(page.locator("text=test-project").first()).toBeVisible({
      timeout: 8000,
    });

    await expect(
      page.locator("text=Select a file to view")
    ).toBeVisible({ timeout: 5000 });
  });
});
