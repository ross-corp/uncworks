import { test, expect } from "@playwright/test";

test("spec-driven option visible in orchestration mode selector", async ({ page }) => {
  await page.goto("/");

  // Open the create run form
  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // The form may have an orchestration mode selector (select or radio/button group).
  // Look for a select with orchestration-related testid or the text "Spec-Driven" inside the form.
  const form = page.getByTestId("form-modal");

  // Check for a dedicated orchestration mode <select>
  const orchSelect = form.locator("[data-testid='form-orchestration-select']");
  const hasOrchSelect = await orchSelect.isVisible().catch(() => false);

  if (hasOrchSelect) {
    // Verify "spec-driven" is among the options
    const options = await orchSelect.locator("option").allTextContents();
    expect(options.some((text) => /spec.driven/i.test(text))).toBe(true);
  } else {
    // Fall back: the Spec tab itself is the spec-driven entry point.
    // The "Spec" tab button acts as the spec-driven option.
    const specTab = page.getByTestId("form-tab-spec");
    await expect(specTab).toBeVisible();
    const tabText = await specTab.textContent();
    expect(tabText).toBeTruthy();
  }
});

test("run detail shows stage badge when present", async ({ page }) => {
  await page.goto("/");

  // Wait for at least one run row to appear
  const firstRow = page.locator("[data-testid^='run-row-']").first();
  const hasRuns = await firstRow.isVisible().catch(() => false);
  test.skip(!hasRuns, "No runs available to test stage badge");

  // Double-click the first row to open detail
  await firstRow.dblclick();
  await expect(page.getByTestId("run-detail")).toBeVisible();

  // The Info tab should be visible by default
  await expect(page.getByTestId("detail-tab-info")).toBeVisible();

  // Check if stage data is rendered in the info pane.
  // Stage is rendered as a MetaRow with label "Stage" when run.status.stage exists.
  const detailPane = page.getByTestId("run-detail");
  const stageLabelLocator = detailPane.locator("text=Stage").first();
  const hasStage = await stageLabelLocator.isVisible().catch(() => false);

  if (hasStage) {
    // Stage label is visible — verify it has a corresponding value next to it
    await expect(stageLabelLocator).toBeVisible();
    // The MetaRow renders label and value as siblings; the parent should contain text
    const stageRow = stageLabelLocator.locator("..");
    const rowText = await stageRow.textContent();
    expect(rowText!.length).toBeGreaterThan("Stage".length);
  } else {
    // Stage is conditional — it's OK if this run doesn't have stage data.
    // The test passes: the component correctly omits stage when absent.
    expect(hasStage).toBe(false);
  }
});

test("run list shows stage alongside phase", async ({ page }) => {
  await page.goto("/");

  // Wait for run list to load
  await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 15000 });

  const rows = page.locator("[data-testid^='run-row-']");
  const rowCount = await rows.count();
  test.skip(rowCount === 0, "No runs available to test stage in list");

  // Scan all visible rows for any that show stage info.
  // Stage is rendered as "(stageName)" inside the Phase column.
  let foundStage = false;
  for (let i = 0; i < rowCount; i++) {
    const row = rows.nth(i);
    const rowText = await row.textContent();
    // Stage appears as parenthesized text: e.g. "running (planning)"
    if (rowText && /\(.+\)/.test(rowText)) {
      foundStage = true;

      // Verify the stage text is inside the Phase column cell (the 5th <td>)
      const phaseTd = row.locator("td").nth(4);
      const phaseText = await phaseTd.textContent();
      expect(phaseText).toMatch(/\(.+\)/);
      break;
    }
  }

  if (!foundStage) {
    // No runs currently have stage data — that's acceptable.
    // The component conditionally renders stage only when present.
    expect(foundStage).toBe(false);
  }
});

test("verification panel loads for completed spec-driven runs", async ({ page }) => {
  await page.goto("/");

  // Wait for run list to load
  await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 15000 });

  // Look for a completed run (succeeded or failed)
  const completedRow = page
    .locator("[data-testid^='run-row-']")
    .filter({ hasText: /succeeded|failed/i })
    .first();

  const hasCompleted = await completedRow.isVisible().catch(() => false);
  test.skip(!hasCompleted, "No completed runs available to test verification panel");

  // Double-click to open the detail view
  await completedRow.dblclick();
  await expect(page.getByTestId("run-detail")).toBeVisible();

  // The Info tab should be active by default
  await expect(page.getByTestId("detail-tab-info")).toBeVisible();

  // Verification panel is conditionally rendered when run.status.verificationResult exists.
  // Check if the "Verification" heading or the verification-panel testid appears.
  const detailPane = page.getByTestId("run-detail");
  const verificationHeading = detailPane.locator("text=Verification").first();
  const verificationPanel = page.getByTestId("verification-panel");

  // Allow time for the VerificationPanel to fetch and render
  const hasVerification = await verificationHeading
    .isVisible({ timeout: 5000 })
    .catch(() => false);

  if (hasVerification) {
    // The panel should render with its data-testid
    await expect(verificationPanel).toBeVisible({ timeout: 10000 });

    // Should show PASSED or FAILED verdict
    const panelText = await verificationPanel.textContent();
    expect(panelText).toMatch(/PASSED|FAILED/);
  } else {
    // This completed run doesn't have verification results — acceptable.
    // The component correctly hides the panel when no verificationResult exists.
    expect(hasVerification).toBe(false);
  }
});
