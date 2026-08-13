import { test, expect } from "@playwright/test";
import { heading, mockRuns } from "./helpers";

// mockRuns stubs both REST and ConnectRPC. The app lists runs over ConnectRPC,
// so stubbing only /api/v1/runs left every row missing and this whole file red.
const mockRunsApi = mockRuns;

const SAMPLE_RUNS = [
  {
    id: "run-1",
    name: "test-run-alpha",
    spec: {
      backend: "Pod",
      repos: [{ url: "https://github.com/org/repo", branch: "main" }],
      prompt: "Fix the tests",
      ttlSeconds: 900,
      modelTier: "default",
      displayName: "Alpha Run",
    },
    status: { phase: "Running", message: "", podName: "pod-1", traceID: "", startedAt: "", completedAt: "" },
    createdAt: new Date().toISOString(),
  },
  {
    id: "run-2",
    name: "test-run-beta",
    spec: {
      backend: "Pod",
      repos: [{ url: "https://github.com/org/repo2", branch: "main" }],
      prompt: "Add feature",
      ttlSeconds: 900,
      modelTier: "default",
      displayName: "Beta Run",
    },
    status: { phase: "Succeeded", message: "", podName: "pod-2", traceID: "", startedAt: "", completedAt: "" },
    createdAt: new Date(Date.now() - 3600_000).toISOString(),
  },
  {
    id: "run-3",
    name: "test-run-gamma",
    spec: {
      backend: "Pod",
      repos: [{ url: "https://github.com/org/repo3", branch: "dev" }],
      prompt: "Deploy staging",
      ttlSeconds: 900,
      modelTier: "default",
      displayName: "Gamma Run",
    },
    status: { phase: "Failed", message: "OOM", podName: "pod-3", traceID: "", startedAt: "", completedAt: "" },
    createdAt: new Date(Date.now() - 86400_000).toISOString(),
  },
];

test.describe("Run List View", () => {
  test("page loads and shows the product header", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");

    await expect(page.getByText("UNCWORKS").first()).toBeVisible();
    await expect(heading(page, "Runs")).toBeVisible();
  });

  test("shows runs in the list", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");

    await expect(page.getByTestId("run-row-run-1")).toBeVisible();
    await expect(page.getByTestId("run-row-run-2")).toBeVisible();
    await expect(page.getByTestId("run-row-run-3")).toBeVisible();

    // Check run names are displayed
    await expect(page.locator("text=Alpha Run")).toBeVisible();
    await expect(page.locator("text=Beta Run")).toBeVisible();
    await expect(page.locator("text=Gamma Run")).toBeVisible();
  });

  test("shows empty state when there are no runs", async ({ page }) => {
    await mockRunsApi(page, []);
    await page.goto("/");

    await expect(page.locator("text=No runs yet")).toBeVisible();
  });

  test("j/k navigation moves selection via bg-accent class", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");

    // Wait for runs to load
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // First item should be selected by default (index 0)
    await expect(page.getByTestId("run-row-run-1")).toHaveClass(/bg-accent/);

    // Press j to move down
    await page.keyboard.press("j");
    await expect(page.getByTestId("run-row-run-2")).toHaveClass(/bg-accent/);
    await expect(page.getByTestId("run-row-run-1")).not.toHaveClass(/bg-accent/);

    // Press j again
    await page.keyboard.press("j");
    await expect(page.getByTestId("run-row-run-3")).toHaveClass(/bg-accent/);

    // Press k to go back up
    await page.keyboard.press("k");
    await expect(page.getByTestId("run-row-run-2")).toHaveClass(/bg-accent/);

    // Press k again to return to first
    await page.keyboard.press("k");
    await expect(page.getByTestId("run-row-run-1")).toHaveClass(/bg-accent/);
  });

  test("j does not go past the last item", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // Press j 10 times — should stop at the last item
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press("j");
    }
    await expect(page.getByTestId("run-row-run-3")).toHaveClass(/bg-accent/);
  });

  test("k does not go past the first item", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // Press k several times — should stay on first item
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press("k");
    }
    await expect(page.getByTestId("run-row-run-1")).toHaveClass(/bg-accent/);
  });

  test("/ filters by name, and Esc clears the filter", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // The filter box is always present. Pressing / focuses it and selects the
    // name field, which is what the placeholder change reports.
    const filterInput = page.getByRole("textbox", { name: /filter/i }).or(
      page.locator("input[placeholder^='filter']"),
    ).first();
    await page.keyboard.press("/");
    await expect(filterInput).toBeVisible();
    await expect(filterInput).toBeFocused();

    await filterInput.fill("alpha");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();
    await expect(page.getByTestId("run-row-run-2")).toBeHidden();
    await expect(page.getByTestId("run-row-run-3")).toBeHidden();

    await page.keyboard.press("Escape");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();
    await expect(page.getByTestId("run-row-run-2")).toBeVisible();
    await expect(page.getByTestId("run-row-run-3")).toBeVisible();
  });

  test("/ filter shows 'No runs match filter' for no results", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    await page.keyboard.press("/");
    await page.locator("input[placeholder^='filter']").first().fill("nonexistent-query");

    await expect(page.getByText("No runs match filter")).toBeVisible();
  });

  test("n navigates to /new", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    await page.keyboard.press("n");
    await expect(page).toHaveURL(/\/new/);
  });

  test("enter on a selected run navigates to /run/:id", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // First run is selected by default; press Enter
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/run\/run-1/);
  });

  test("enter after j navigates to the correct run", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-1")).toBeVisible();

    // Move to second run and press enter
    await page.keyboard.press("j");
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/run\/run-2/);
  });

  test("clicking a run row navigates to its detail", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");
    await expect(page.getByTestId("run-row-run-2")).toBeVisible();

    await page.getByTestId("run-row-run-2").click();
    await expect(page).toHaveURL(/\/run\/run-2/);
  });

  test("the phase filter buttons are visible", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");

    for (const phase of ["running", "waiting", "failed", "succeeded"]) {
      await expect(page.getByRole("button", { name: phase, exact: true })).toBeVisible();
    }
  });

  test("footer shortcuts are visible", async ({ page }) => {
    await mockRunsApi(page, SAMPLE_RUNS);
    await page.goto("/");

    await expect(page.getByText("j/k nav")).toBeVisible();
  });
});
