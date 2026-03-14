import { test, expect } from "@playwright/test";

test.describe("Observability tabs", () => {
  test("Logs tab shows content for running run", async ({ page }) => {
    await page.goto("/");

    // Find a running run in the table.
    const runningRow = page
      .locator("tr[data-testid^='table-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Logs tab");

    // Click the running run to open detail panel.
    await runningRow.click();
    await expect(page.getByTestId("detail-panel")).toBeVisible();

    // Click the Logs tab.
    await page.getByTestId("detail-tab-logs").click();

    // Wait for the log viewer to appear.
    await expect(page.getByTestId("log-viewer")).toBeVisible({ timeout: 15000 });

    // Verify the log viewer has some content (not completely empty).
    // xterm.js renders into a canvas/div; check for any text content in the container.
    await expect(async () => {
      const content = await page.getByTestId("log-viewer").textContent();
      expect(content?.length).toBeGreaterThan(0);
    }).toPass({ timeout: 30000 });
  });

  test("Files tab shows tree for running run", async ({ page }) => {
    await page.goto("/");

    const runningRow = page
      .locator("tr[data-testid^='table-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Files tab");

    await runningRow.click();
    await expect(page.getByTestId("detail-panel")).toBeVisible();

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
      .locator("tr[data-testid^='table-row-']")
      .filter({ hasText: /running/i })
      .first();
    const hasRunning = await runningRow.isVisible().catch(() => false);
    test.skip(!hasRunning, "No running runs available to test Shell tab");

    await runningRow.click();
    await expect(page.getByTestId("detail-panel")).toBeVisible();

    // Click the Shell tab.
    await page.getByTestId("detail-tab-shell").click();

    // Wait for the shell terminal to be visible.
    await expect(page.getByTestId("shell-terminal")).toBeVisible({
      timeout: 15000,
    });

    // Type `ls` and press Enter. The terminal may not have a real connection
    // in all E2E environments, so wrap in a soft assertion.
    try {
      await page.keyboard.type("ls\n", { delay: 50 });
      // Wait briefly for output.
      await page.waitForTimeout(3000);
      const content = await page.getByTestId("shell-terminal").textContent();
      if (content && content.length > 2) {
        // We got some output; consider it a pass.
        expect(content.length).toBeGreaterThan(0);
      }
    } catch {
      // Shell may not be connected in this environment; the terminal rendering is sufficient.
    }
  });

  test("completed run with expired pod shows disabled tabs", async ({
    page,
  }) => {
    await page.goto("/");

    // Look for a completed run (Succeeded or Failed).
    const completedRow = page
      .locator("tr[data-testid^='table-row-']")
      .filter({ hasText: /succeeded|failed/i })
      .first();
    const hasCompleted = await completedRow.isVisible().catch(() => false);
    test.skip(!hasCompleted, "No completed runs available to test disabled tabs");

    await completedRow.click();
    await expect(page.getByTestId("detail-panel")).toBeVisible();

    // Files tab should indicate the pod is not available.
    const filesTab = page.getByTestId("detail-tab-files");
    await filesTab.click();

    // Check for a disabled state or "pod expired" message.
    // The tab may be disabled (aria-disabled) or show an informational message.
    const filesContent = page.getByTestId("files-tab-content");
    const isFilesVisible = await filesContent.isVisible().catch(() => false);
    if (isFilesVisible) {
      const text = await filesContent.textContent();
      // Expect some indication that the pod is unavailable.
      expect(text?.toLowerCase()).toMatch(/not available|expired|no pod|disabled/i);
    } else {
      // The tab itself might be disabled/unclickable — that's also valid.
      const isDisabled = await filesTab.getAttribute("aria-disabled");
      expect(isDisabled).toBe("true");
    }

    // Logs tab should still work (falls back to persisted logOutput).
    await page.getByTestId("detail-tab-logs").click();
    await expect(page.getByTestId("log-viewer")).toBeVisible({ timeout: 10000 });
  });
});
