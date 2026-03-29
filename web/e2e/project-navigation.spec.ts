// e2e/project-navigation.spec.ts — Navigation flows for the Projects section.
// All API calls are mocked via page.route() — no real backend needed.
import { test, expect } from "@playwright/test";

const SAMPLE_PROJECTS = [
  {
    name: "alpha-project",
    displayName: "alpha-project",
    description: "First project",
    repos: [{ url: "https://github.com/org/alpha", branch: "main" }],
    configRepoReady: true,
    runCount: 3,
    lastRunId: "run-1",
    totalCost: "$0.12",
    createdAt: new Date(Date.now() - 86400_000).toISOString(),
  },
  {
    name: "beta-project",
    displayName: "beta-project",
    description: "",
    repos: [],
    configRepoReady: false,
    configRepoMessage: "provisioning",
    runCount: 0,
    lastRunId: "",
    totalCost: "",
    createdAt: new Date().toISOString(),
  },
];

const SAMPLE_PROJECT_DETAIL = {
  name: "alpha-project",
  displayName: "alpha-project",
  description: "First project",
  repos: [{ url: "https://github.com/org/alpha", branch: "main" }],
  configRepoReady: true,
  configRepoURL: "https://github.com/org/alpha-config",
  runCount: 3,
  totalCost: "$0.12",
};

function mockApis(page: import("@playwright/test").Page) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (
        route.request().url().includes("/logs") ||
        route.request().url().includes("/traces")
      )
        return;
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
    page.route("**/api/v1/projects", (route) => {
      if (route.request().method() === "GET") {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(SAMPLE_PROJECTS),
        });
      } else {
        route.fallback();
      }
    }),
    page.route("**/api/v1/projects/alpha-project", (route) => {
      if (
        route.request().method() === "GET" &&
        !route.request().url().includes("/files")
      ) {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(SAMPLE_PROJECT_DETAIL),
        });
      } else {
        route.fallback();
      }
    }),
    page.route("**/api/v1/projects/alpha-project/files", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
    page.route("**/api/v1/projects/beta-project", (route) => {
      if (
        route.request().method() === "GET" &&
        !route.request().url().includes("/files")
      ) {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            ...SAMPLE_PROJECTS[1],
            configRepoURL: "",
            runCount: 0,
            totalCost: "",
          }),
        });
      } else {
        route.fallback();
      }
    }),
    page.route("**/api/v1/projects/beta-project/files", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
  ]);
}

test.describe("Projects List — navigation", () => {
  test("shows project rows for each project", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=alpha-project")).toBeVisible({ timeout: 8000 });
    await expect(page.locator("text=beta-project")).toBeVisible({ timeout: 5000 });
  });

  test("clicking a project row navigates to /projects/:name", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    // Click the project row (the outer clickable div)
    await page.locator("text=alpha-project").first().click();

    await expect(page).toHaveURL(/\/projects\/alpha-project/, { timeout: 5000 });
  });

  test("project detail page loads with project name in header", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });
  });

  test("project detail page shows breadcrumb with Projects link", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=Projects").first()).toBeVisible({ timeout: 8000 });
  });

  test("breadcrumb 'Projects' link navigates back to /projects", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    // The breadcrumb "Projects" link
    const breadcrumb = page.locator("a", { hasText: "Projects" }).first();
    await expect(breadcrumb).toBeVisible();
    await breadcrumb.click();

    await expect(page).toHaveURL(/\/projects$/, { timeout: 5000 });
  });

  test("project detail shows Specs / Runs / Settings tabs", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    await expect(page.locator("button", { hasText: "Specs" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Runs" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Settings" })).toBeVisible();
  });

  test("clicking Runs tab switches to Runs content", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    await page.locator("button", { hasText: "Runs" }).first().click();

    // Runs tab shows "No runs yet" or a run list
    const runsContent = page.locator("text=No runs yet");
    await expect(runsContent).toBeVisible({ timeout: 5000 });
  });

  test("clicking Settings tab switches to Settings content", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    await page.locator("button", { hasText: "Settings" }).first().click();

    // Settings tab should show the Description field
    await expect(page.locator("text=Description")).toBeVisible({ timeout: 5000 });
  });

  test("'+ new run' button in project detail links to /new?project=:name", async ({
    page,
  }) => {
    await mockApis(page);
    await page.goto("/projects/alpha-project");

    await expect(page.locator("text=alpha-project").first()).toBeVisible({
      timeout: 8000,
    });

    const newRunBtn = page.locator("button", { hasText: "+ new run" }).first();
    await expect(newRunBtn).toBeVisible({ timeout: 5000 });
    await newRunBtn.click();

    await expect(page).toHaveURL(/\/new\?project=alpha-project/, { timeout: 5000 });
  });

  test("beta-project row shows 'provisioning' badge", async ({ page }) => {
    await mockApis(page);
    await page.goto("/projects");

    await expect(page.locator("text=beta-project")).toBeVisible({ timeout: 8000 });
    await expect(page.locator("text=provisioning").first()).toBeVisible({ timeout: 5000 });
  });
});
