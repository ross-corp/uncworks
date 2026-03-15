import { test, expect } from "@playwright/test";

test("run status transitions in list", async ({ page }) => {
  await page.goto("/");

  const runName = `e2e-lifecycle-${Date.now()}`;

  // Create a run via the form
  await page.getByTestId("icon-rail-new-run").click();
  await page.getByTestId("form-name-input").fill(runName);
  await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/test-repo");
  await page.getByTestId("form-repo-row-0-branch").fill("main");
  await page.getByTestId("form-prompt-input").fill("Create a file called DONE.txt with PASS");
  await page.getByTestId("form-submit").click();

  // Wait for form to close
  await expect(page.getByTestId("form-modal")).not.toBeVisible({ timeout: 10000 });

  // Wait for the run to appear in the list
  await expect(page.getByText(runName)).toBeVisible({ timeout: 15000 });

  // Wait for phase to transition from pending (generous timeout for real LLM)
  await expect(async () => {
    const row = page.locator(`[data-testid^="run-row-"]`).filter({ hasText: runName });
    const rowText = await row.textContent();
    expect(rowText).not.toContain("Pending");
  }).toPass({ timeout: 180000 });
});

test("detail view shows run data", async ({ page }) => {
  await page.goto("/");

  // Wait for at least one run row to appear
  const firstRow = page.locator("[data-testid^='run-row-']").first();
  const hasRuns = await firstRow.isVisible().catch(() => false);
  test.skip(!hasRuns, "No runs available to test detail view");

  // Double-click the first row to open detail
  await firstRow.dblclick();

  // Verify RunDetail (DetailPane) is visible
  await expect(page.getByTestId("run-detail")).toBeVisible();
  await expect(page.getByTestId("detail-name")).toHaveText(/.+/);
  await expect(page.getByTestId("detail-phase")).toBeVisible();
});

test("cancel running run", async ({ page }) => {
  await page.goto("/");

  const runName = `e2e-cancel-${Date.now()}`;

  // Create a run
  await page.getByTestId("icon-rail-new-run").click();
  await page.getByTestId("form-name-input").fill(runName);
  await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/test-repo");
  await page.getByTestId("form-repo-row-0-branch").fill("main");
  await page.getByTestId("form-prompt-input").fill("Do a very thorough analysis of every file in the repository");
  await page.getByTestId("form-submit").click();
  await expect(page.getByTestId("form-modal")).not.toBeVisible({ timeout: 10000 });

  // Wait for run to appear and double-click to open detail
  await expect(page.getByText(runName)).toBeVisible({ timeout: 15000 });
  await page.getByText(runName).dblclick();

  // Verify detail view opens
  await expect(page.getByTestId("run-detail")).toBeVisible();

  // Wait for cancel button to appear
  const cancelButton = page.getByTestId("detail-cancel");
  const canCancel = await cancelButton.isVisible().catch(() => false);
  test.skip(!canCancel, "Run is not in a cancellable state");

  // Click cancel
  await cancelButton.click();

  // Verify phase transitions to cancelled
  await expect(page.getByTestId("detail-phase")).toContainText(/cancelled/i, { timeout: 30000 });
});

test("HITL input flow", async ({ page }) => {
  await page.goto("/");

  // Look for a run in waiting_for_input state
  const waitingRow = page.locator("[data-testid^='run-row-']").filter({
    hasText: /waiting/i,
  }).first();

  const hasWaiting = await waitingRow.isVisible().catch(() => false);
  test.skip(!hasWaiting, "No run in waiting_for_input state available");

  // Double-click the waiting run
  await waitingRow.dblclick();
  await expect(page.getByTestId("run-detail")).toBeVisible();

  // Type in HITL input
  await page.getByTestId("detail-hitl-input").fill("Approved, proceed with changes");

  // Click send
  await page.getByTestId("detail-hitl-send").click();

  // Verify the phase transitions (input was sent)
  await expect(page.getByTestId("detail-hitl-input")).toHaveValue("");
});
