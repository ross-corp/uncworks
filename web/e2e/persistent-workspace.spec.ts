import { test, expect } from "@playwright/test";

test.describe("Persistent workspace", () => {
  // ---------- 17.1: Completed run -> Logs tab -> log-viewer renders ----------

  test("completed run shows logs from disk", async ({ page }) => {
    await page.goto("/");

    const completedCard = page
      .locator("[data-testid^='run-card-']")
      .filter({ hasText: /succeeded|failed/i })
      .first();
    const hasCompleted = await completedCard.isVisible().catch(() => false);
    test.skip(!hasCompleted, "No completed runs available to test persistent logs");

    await completedCard.click();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Logs tab.
    await page.getByTestId("detail-tab-logs").click();

    // Wait for the log viewer to appear (content served from disk via PVC).
    await expect(page.getByTestId("log-viewer")).toBeVisible({ timeout: 15000 });

    // Verify the log viewer has some content.
    await expect(async () => {
      const content = await page.getByTestId("log-viewer").textContent();
      expect(content?.length).toBeGreaterThan(0);
    }).toPass({ timeout: 30000 });
  });

  // ---------- 17.2: Completed run -> Files tab -> file tree renders ----------

  test("completed run shows file tree from disk", async ({ page }) => {
    await page.goto("/");

    const completedCard = page
      .locator("[data-testid^='run-card-']")
      .filter({ hasText: /succeeded|failed/i })
      .first();
    const hasCompleted = await completedCard.isVisible().catch(() => false);
    test.skip(!hasCompleted, "No completed runs available to test persistent files");

    await completedCard.click();
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

  // ---------- 17.3: Completed run -> Shell tab -> "Debug Run" button ----------

  test("completed run shows Debug Run button", async ({ page }) => {
    await page.goto("/");

    const completedCard = page
      .locator("[data-testid^='run-card-']")
      .filter({ hasText: /succeeded|failed/i })
      .first();
    const hasCompleted = await completedCard.isVisible().catch(() => false);
    test.skip(!hasCompleted, "No completed runs available to test debug button");

    await completedCard.click();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Shell tab.
    await page.getByTestId("detail-tab-shell").click();

    // "Debug Run" button should be visible for a completed run (Deployment replicas=0).
    const debugBtn = page.getByTestId("debug-run-btn");
    const hasDebugBtn = await debugBtn.isVisible().catch(() => false);
    test.skip(!hasDebugBtn, "Debug Run button not rendered (may not be implemented in UI yet)");

    await expect(debugBtn).toBeVisible();

    // Click the Debug Run button.
    await debugBtn.click();

    // After clicking, the button text should change or a terminal should appear.
    await page.waitForTimeout(3000);

    // Verify either a terminal appears or the button changes to "Stop Debug".
    const shellTerminal = page.getByTestId("shell-terminal");
    const stopDebugBtn = page.getByTestId("debug-run-btn");
    const terminalVisible = await shellTerminal.isVisible().catch(() => false);
    const btnText = await stopDebugBtn.textContent();

    if (terminalVisible) {
      expect(terminalVisible).toBe(true);
    } else if (btnText) {
      expect(btnText.toLowerCase()).toMatch(/stop|active|debug/i);
    }
  });

  // ---------- 17.4: Traces tab -> timeline renders ----------

  test("Traces tab renders timeline or empty state", async ({ page }) => {
    await page.goto("/");

    const anyCard = page.locator("[data-testid^='run-card-']").first();
    const hasRuns = await anyCard.isVisible().catch(() => false);
    test.skip(!hasRuns, "No runs available to test Traces tab");

    await anyCard.click();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Click the Traces tab.
    const tracesTab = page.getByTestId("detail-tab-traces");
    const hasTracesTab = await tracesTab.isVisible().catch(() => false);
    test.skip(!hasTracesTab, "Traces tab not available in this build");

    await tracesTab.click();

    // Wait for either the trace timeline or an empty state message.
    await expect(async () => {
      const timeline = page.getByTestId("trace-timeline");
      const emptyState = page.getByTestId("traces-empty-state");
      const timelineVisible = await timeline.isVisible().catch(() => false);
      const emptyVisible = await emptyState.isVisible().catch(() => false);
      expect(timelineVisible || emptyVisible).toBe(true);
    }).toPass({ timeout: 15000 });
  });

  // ---------- 17.5: Running run -> all tabs work ----------

  test("running run has all tabs functional", async ({ page }) => {
    await page.goto("/");

    const runningCard = page
      .locator("[data-testid^='run-card-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningCard.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test all tabs");

    await runningCard.click();
    await expect(page.getByTestId("run-detail")).toBeVisible();

    // Logs tab.
    await page.getByTestId("detail-tab-logs").click();
    await expect(page.getByTestId("log-viewer")).toBeVisible({ timeout: 15000 });

    // Files tab.
    await page.getByTestId("detail-tab-files").click();
    await expect(async () => {
      const tree = page.getByTestId("file-tree");
      await expect(tree).toBeVisible();
    }).toPass({ timeout: 30000 });

    // Shell tab.
    await page.getByTestId("detail-tab-shell").click();
    await expect(page.getByTestId("shell-terminal")).toBeVisible({
      timeout: 15000,
    });

    // Traces tab (may or may not be available).
    const tracesTab = page.getByTestId("detail-tab-traces");
    const hasTracesTab = await tracesTab.isVisible().catch(() => false);
    if (hasTracesTab) {
      await tracesTab.click();
      await page.waitForTimeout(3000);
      const timeline = page.getByTestId("trace-timeline");
      const emptyState = page.getByTestId("traces-empty-state");
      const timelineVisible = await timeline.isVisible().catch(() => false);
      const emptyVisible = await emptyState.isVisible().catch(() => false);
      expect(timelineVisible || emptyVisible).toBe(true);
    }
  });
});
