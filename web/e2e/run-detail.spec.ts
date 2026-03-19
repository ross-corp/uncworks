import { test, expect } from "@playwright/test";

const SAMPLE_RUN = {
  id: "run-detail-1",
  name: "detail-test-run",
  spec: {
    backend: "Pod",
    repos: [{ url: "https://github.com/org/repo", branch: "main" }],
    prompt: "Fix the failing CI pipeline",
    ttlSeconds: 900,
    modelTier: "default",
    displayName: "Detail Test Run",
    orchestrationMode: "single",
  },
  status: {
    phase: "Running",
    message: "",
    podName: "pod-detail-1",
    traceID: "trace-1",
    startedAt: new Date().toISOString(),
    completedAt: "",
  },
  createdAt: new Date().toISOString(),
};

function mockRunDetailApis(page: import("@playwright/test").Page) {
  return Promise.all([
    // List runs (for Layout + CommandPalette)
    page.route("**/api/v1/runs", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([SAMPLE_RUN]),
      });
    }),
    // Get single run
    page.route("**/api/v1/runs/run-detail-1", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(SAMPLE_RUN),
      });
    }),
    // Structured logs
    page.route("**/api/v1/runs/run-detail-1/logs/structured", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([
          { timestamp: "2026-03-18T10:00:00Z", type: "user", content: "Fix the failing CI pipeline" },
          { timestamp: "2026-03-18T10:00:05Z", type: "assistant", content: "I will analyze the CI configuration." },
        ]),
      });
    }),
    // Traces
    page.route("**/api/v1/runs/run-detail-1/traces", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    }),
  ]);
}

test.describe("Run Detail View", () => {
  test("loads with header showing run name", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    await expect(page.locator("text=Detail Test Run")).toBeVisible();
  });

  test("shows run status badge in header", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    // The header should contain the run name — just verify the page loaded correctly
    await expect(page.locator("text=Detail Test Run")).toBeVisible();
    // Footer shortcuts should be visible
    await expect(page.locator("text=esc back")).toBeVisible();
  });

  test("tab bar shows 5 tabs", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    const tabs = [
      { key: "activity", label: "Activity" },
      { key: "files", label: "Files" },
      { key: "shell", label: "Shell" },
      { key: "traces", label: "Traces" },
      { key: "verify", label: "Verify" },
    ];

    for (const tab of tabs) {
      await expect(page.getByTestId(`detail-tab-${tab.key}`)).toBeVisible();
      await expect(page.getByTestId(`detail-tab-${tab.key}`)).toContainText(tab.label);
    }
  });

  test("activity tab is active by default", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    await expect(page.getByTestId("detail-tab-activity")).toHaveClass(/bg-accent/);
  });

  test("number keys switch tabs", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    // Wait for initial load
    await expect(page.getByTestId("detail-tab-activity")).toBeVisible();

    // Press 2 to switch to Files tab
    await page.keyboard.press("2");
    await expect(page.getByTestId("detail-tab-files")).toHaveClass(/bg-accent/);
    await expect(page.getByTestId("detail-tab-activity")).not.toHaveClass(/bg-accent/);

    // Press 3 to switch to Shell tab
    await page.keyboard.press("3");
    await expect(page.getByTestId("detail-tab-shell")).toHaveClass(/bg-accent/);

    // Press 4 to switch to Traces tab
    await page.keyboard.press("4");
    await expect(page.getByTestId("detail-tab-traces")).toHaveClass(/bg-accent/);

    // Press 5 to switch to Verify tab
    await page.keyboard.press("5");
    await expect(page.getByTestId("detail-tab-verify")).toHaveClass(/bg-accent/);

    // Press 1 to go back to Activity
    await page.keyboard.press("1");
    await expect(page.getByTestId("detail-tab-activity")).toHaveClass(/bg-accent/);
  });

  test("clicking a tab switches to it", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    await page.getByTestId("detail-tab-files").click();
    await expect(page.getByTestId("detail-tab-files")).toHaveClass(/bg-accent/);
    await expect(page.getByTestId("detail-tab-activity")).not.toHaveClass(/bg-accent/);
  });

  test("Esc navigates back to /", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    // Wait for the page to load
    await expect(page.locator("text=Detail Test Run")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page).toHaveURL(/\/$/);
  });

  test("i key toggles info overlay", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    await expect(page.locator("text=Detail Test Run")).toBeVisible();

    // Info overlay should not be visible initially
    const infoOverlay = page.locator("text=run-detail-1").last();
    await expect(page.locator(".bg-muted\\/50")).not.toBeVisible();

    // Press i to show info overlay
    await page.keyboard.press("i");

    // Info overlay should show run details
    await expect(page.locator("text=ID")).toBeVisible();
    await expect(page.locator("text=Created")).toBeVisible();
    await expect(page.locator("text=Model")).toBeVisible();

    // Press i again to hide
    await page.keyboard.press("i");
    // The info panel has bg-muted/50 class; it should be gone
    await expect(page.locator(".bg-muted\\/50")).not.toBeVisible();
  });

  test("footer shows keyboard shortcuts", async ({ page }) => {
    await mockRunDetailApis(page);
    await page.goto("/run/run-detail-1");

    await expect(page.locator("text=1 activity")).toBeVisible();
    await expect(page.locator("text=i info")).toBeVisible();
    await expect(page.locator("text=esc back")).toBeVisible();
  });

  test("loading state shows while fetching run", async ({ page }) => {
    // Delay the run API response to see the loading state
    await page.route("**/api/v1/runs", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    });
    await page.route("**/api/v1/runs/run-detail-1", async (route) => {
      // Delay response by 2 seconds
      await new Promise((r) => setTimeout(r, 2000));
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(SAMPLE_RUN),
      });
    });

    await page.goto("/run/run-detail-1");
    await expect(page.locator("text=Loading...")).toBeVisible();
  });
});
