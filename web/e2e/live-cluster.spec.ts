// e2e/live-cluster.spec.ts — End-to-end tests against the REAL cluster.
// NO route mocking. Requires:
//   kubectl port-forward -n uncworks svc/uncworks-aot-apiserver 50055:50055
// Run with: VITE_API_URL=http://localhost:50055 npx playwright test live-cluster
//
// These tests are the source of truth for "does the app actually work."
import { test, expect } from "@playwright/test";

// Generous timeouts — real API calls are slower than mocks.
const LOAD_TIMEOUT = 15_000;
const NAV_TIMEOUT = 10_000;

test.describe("Live cluster — projects", () => {
  test("projects list loads at least one project", async ({ page }) => {
    await page.goto("/projects");
    // Project rows have this specific class combination
    await expect(
      page.locator("[class*='border-border']").first()
    ).toBeVisible({ timeout: LOAD_TIMEOUT });
  });

  test("clicking a project navigates to detail view without crashing", async ({
    page,
  }) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => errors.push(e.message));

    await page.goto("/projects");

    // Wait for a project row (has specific class from ProjectListView)
    const firstRow = page.locator("[class*='border-b'][class*='cursor-pointer']").first();
    await expect(firstRow).toBeVisible({ timeout: LOAD_TIMEOUT });

    // Click it
    await firstRow.click();

    // Should navigate to /projects/:name
    await expect(page).toHaveURL(/\/projects\//, { timeout: NAV_TIMEOUT });

    // Page should not have crashed (no "Something went wrong")
    await expect(page.locator("text=Something went wrong")).not.toBeVisible({
      timeout: 3000,
    });

    // The project name should appear in the header somewhere
    const projectName = page.url().split("/projects/")[1];
    await expect(
      page.locator(`text=${projectName}`).first()
    ).toBeVisible({ timeout: LOAD_TIMEOUT });

    expect(errors, `JS errors: ${errors.join(", ")}`).toHaveLength(0);
  });

  test("project detail shows Specs / Runs / Settings tabs", async ({ page }) => {
    await page.goto("/projects");
    const firstRow = page.locator("[class*='border-b'][class*='cursor-pointer']").first();
    await expect(firstRow).toBeVisible({ timeout: LOAD_TIMEOUT });
    await firstRow.click();
    await expect(page).toHaveURL(/\/projects\//, { timeout: NAV_TIMEOUT });

    await expect(page.getByRole("tab", { name: /specs/i }).or(page.locator("button", { hasText: "Specs" }))).toBeVisible({ timeout: LOAD_TIMEOUT });
    await expect(page.getByRole("tab", { name: /runs/i }).or(page.locator("button", { hasText: "Runs" }))).toBeVisible();
    await expect(page.getByRole("tab", { name: /settings/i }).or(page.locator("button", { hasText: "Settings" }))).toBeVisible();
  });
});

test.describe("Live cluster — runs list", () => {
  test("runs list loads without crashing", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => errors.push(e.message));

    await page.goto("/");

    await expect(page.locator("text=Something went wrong")).not.toBeVisible({
      timeout: 3000,
    });

    expect(errors, `JS errors: ${errors.join(", ")}`).toHaveLength(0);
  });

  test("clicking a run navigates to detail view without crashing", async ({
    page,
  }) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => errors.push(e.message));

    await page.goto("/");

    // Wait for run list to load — either a run row (data-testid="run-row-*") or "No runs yet"
    const runRow = page.locator("[data-testid^='run-row-']").first();
    const noRuns = page.locator("text=No runs yet");

    await expect(runRow.or(noRuns)).toBeVisible({ timeout: LOAD_TIMEOUT });

    const hasRun = await runRow.isVisible();
    if (!hasRun) {
      test.skip();
      return;
    }

    await runRow.click();
    // UnifiedRunRow navigates to /run/:id for agent runs, /chainrun/:id for chain runs
    await expect(page).toHaveURL(/\/(run|chainrun)\//, { timeout: NAV_TIMEOUT });
    await expect(page.locator("text=Something went wrong")).not.toBeVisible({ timeout: 3000 });

    expect(errors, `JS errors: ${errors.join(", ")}`).toHaveLength(0);
  });
});

test.describe("Live cluster — settings page", () => {
  test("settings page loads without crashing", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => errors.push(e.message));

    await page.goto("/settings");

    await expect(page.locator("text=Something went wrong")).not.toBeVisible({
      timeout: 3000,
    });
    expect(errors, `JS errors: ${errors.join(", ")}`).toHaveLength(0);
  });
});
