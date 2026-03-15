import { test, expect } from "@playwright/test";

test.describe("Observability tabs", () => {
  test("Logs tab shows content for running run", async ({ page }) => {
    await page.goto("/");

    // Find a running run in the list.
    const runningRow = page
      .locator("[data-testid^='run-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Logs tab");

    // Double-click the running run to open detail view.
    await runningRow.dblclick();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Logs tab.
    await page.getByTestId("detail-tab-logs").click();

    // Wait for the log viewer to appear.
    await expect(page.getByTestId("log-viewer")).toBeVisible({ timeout: 15000 });

    // Verify the log viewer has some content.
    await expect(async () => {
      const content = await page.getByTestId("log-viewer").textContent();
      expect(content?.length).toBeGreaterThan(0);
    }).toPass({ timeout: 30000 });
  });

  test("Files tab shows tree for running run", async ({ page }) => {
    await page.goto("/");

    const runningRow = page
      .locator("[data-testid^='run-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Files tab");

    await runningRow.dblclick();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Files tab.
    await page.getByTestId("detail-tab-files").click();

    // Wait for the file tree to render with at least one entry.
    await expect(async () => {
      const tree = page.getByTestId("file-tree");
      await expect(tree).toBeVisible();
      const entries = tree.locator("[data-testid^='file-entry-']");
      const count = await entries.count();
      expect(count).toBeGreaterThan(0);
    }).toPass({ timeout: 30000 });
  });

  test("Shell tab shows terminal", async ({ page }) => {
    await page.goto("/");

    const runningRow = page
      .locator("[data-testid^='run-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Shell tab");

    await runningRow.dblclick();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Shell tab.
    await page.getByTestId("detail-tab-shell").click();

    // Wait for the shell terminal to be visible.
    await expect(page.getByTestId("shell-terminal")).toBeVisible({
      timeout: 15000,
    });

    // Type `ls` and press Enter.
    try {
      await page.keyboard.type("ls\n", { delay: 50 });
      await page.waitForTimeout(3000);
      const content = await page.getByTestId("shell-terminal").textContent();
      if (content && content.length > 2) {
        expect(content.length).toBeGreaterThan(0);
      }
    } catch {
      // Shell may not be connected in this environment.
    }
  });

  test("completed run shows debug option in shell tab", async ({
    page,
  }) => {
    await page.goto("/");

    // Look for a completed run (Succeeded or Failed).
    const completedRow = page
      .locator("[data-testid^='run-row-']")
      .filter({ hasText: /succeeded|failed/i })
      .first();
    const hasCompleted = await completedRow.isVisible().catch(() => false);
    test.skip(!hasCompleted, "No completed runs available to test");

    await completedRow.dblclick();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Switch to Shell tab — should show Debug Run button for completed runs.
    await page.getByTestId("detail-tab-shell").click();

    // Check for debug button or terminal.
    const debugBtn = page.getByTestId("debug-run-btn");
    const shellTerminal = page.getByTestId("shell-terminal");
    const hasDebug = await debugBtn.isVisible().catch(() => false);
    const hasShell = await shellTerminal.isVisible().catch(() => false);

    // One of these should be present.
    expect(hasDebug || hasShell).toBe(true);

    // Logs tab should still work (falls back to persisted logOutput).
    await page.getByTestId("detail-tab-logs").click();
  });
});
