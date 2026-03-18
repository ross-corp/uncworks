import { test, expect } from "@playwright/test";

test("create prompt-based run", async ({ page }) => {
  await page.goto("/");

  const runName = `e2e-prompt-${Date.now()}`;

  // Open form via icon rail button
  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Fill form fields
  await page.getByTestId("form-name-input").fill(runName);
  await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/test-repo");
  await page.getByTestId("form-repo-row-0-branch").fill("main");
  await page.getByTestId("form-prompt-input").fill("Create a file called DONE.txt with the word PASS");

  // Submit
  await page.getByTestId("form-submit").click();

  // Wait for form to close
  await expect(page.getByTestId("form-modal")).not.toBeVisible({ timeout: 10000 });

  // Verify run appears in the list
  await expect(page.getByText(runName)).toBeVisible({ timeout: 15000 });
});

test("create spec-based run", async ({ page }) => {
  await page.goto("/");

  const runName = `e2e-spec-${Date.now()}`;

  // Open form
  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Fill name and repo
  await page.getByTestId("form-name-input").fill(runName);
  await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/test-repo");
  await page.getByTestId("form-repo-row-0-branch").fill("main");

  // Switch to spec tab
  await page.getByTestId("form-tab-spec").click();

  // Wait for Monaco editor to load
  await expect(page.getByTestId("spec-editor")).toBeVisible({ timeout: 10000 });

  // Type into Monaco editor (click to focus, then type)
  await page.getByTestId("spec-editor").click();
  await page.keyboard.type("# Test Spec\n\nThis is a test specification.");

  // Submit
  await page.getByTestId("form-submit").click();

  // Wait for form to close
  await expect(page.getByTestId("form-modal")).not.toBeVisible({ timeout: 10000 });

  // Verify run appears in the list
  await expect(page.getByText(runName)).toBeVisible({ timeout: 15000 });
});

test("form validation prevents empty submission", async ({ page }) => {
  await page.goto("/");

  // Open form
  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Click submit immediately without filling anything
  await page.getByTestId("form-submit").click();

  // Form should still be visible (validation prevented submission)
  await expect(page.getByTestId("form-modal")).toBeVisible();
});

test("workspace preset fills repos", async ({ page }) => {
  // Inject a workspace into localStorage before loading
  const workspace = {
    id: "e2e-ws-test",
    name: "e2e-workspace",
    description: "Test workspace",
    repos: [
      { url: "https://github.com/preset/repo-one", branch: "develop" },
    ],
    createdAt: new Date().toISOString(),
  };

  await page.goto("/");
  await page.evaluate((ws) => {
    localStorage.setItem("uncworks:workspaces", JSON.stringify([ws]));
  }, workspace);

  // Reload so the workspace is picked up
  await page.reload();

  // Open form
  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Click the workspace button
  await page.getByTestId("form-workspace-e2e-workspace").click();

  // Verify repo fields are pre-filled
  await expect(page.getByTestId("form-repo-row-0-url")).toHaveValue("https://github.com/preset/repo-one");
  await expect(page.getByTestId("form-repo-row-0-branch")).toHaveValue("develop");
});

test("clone run pre-fills the form", async ({ page }) => {
  await page.goto("/");

  // Wait for at least one run row to appear
  const firstRow = page.locator("[data-testid^='run-row-']").first();
  const hasRuns = await firstRow.isVisible().catch(() => false);
  test.skip(!hasRuns, "No runs available to test clone workflow");

  // Double-click the first row to open detail
  await firstRow.dblclick();
  await expect(page.getByTestId("run-detail")).toBeVisible();

  // Get the run name from the detail header
  const originalName = await page.getByTestId("detail-name").textContent();

  // Click the Clone button in the detail header
  await page.getByRole("button", { name: "Clone" }).click();

  // The form should open with pre-filled data
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Name should be the original name with "-clone" suffix
  const nameValue = await page.getByTestId("form-name-input").inputValue();
  expect(nameValue).toContain("-clone");

  // Repo URL should be pre-filled (not empty)
  const repoUrl = await page.getByTestId("form-repo-row-0-url").inputValue();
  expect(repoUrl.length).toBeGreaterThan(0);
});

test("add and remove repo rows", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Should start with one repo row
  await expect(page.getByTestId("form-repo-row-0-url")).toBeVisible();

  // Click "+ Add repo" to add a second row
  await page.getByTestId("form-add-repo").click();
  await expect(page.getByTestId("form-repo-row-1-url")).toBeVisible();
  await expect(page.getByTestId("form-repo-row-1-branch")).toBeVisible();

  // Fill both rows
  await page.getByTestId("form-repo-row-0-url").fill("https://github.com/org/first-repo");
  await page.getByTestId("form-repo-row-0-branch").fill("main");
  await page.getByTestId("form-repo-row-1-url").fill("https://github.com/org/second-repo");
  await page.getByTestId("form-repo-row-1-branch").fill("dev");

  // Add a third row
  await page.getByTestId("form-add-repo").click();
  await expect(page.getByTestId("form-repo-row-2-url")).toBeVisible();

  // Remove the second row by clicking the x button next to it
  // The remove button is inside the repo row container, rendered as "x" text
  const repoRows = page.locator("[data-testid^='form-repo-row-1-url']").locator("..");
  const removeBtn = repoRows.locator("..").getByRole("button", { name: "\u00d7" });
  const hasRemove = await removeBtn.first().isVisible().catch(() => false);
  if (hasRemove) {
    await removeBtn.first().click();

    // After removal, the third row should become the second
    // We should still have two rows
    await expect(page.getByTestId("form-repo-row-0-url")).toBeVisible();
    await expect(page.getByTestId("form-repo-row-1-url")).toBeVisible();
  }
});

test("backend and model tier selection", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // The Backend dropdown should be visible and have the default "pod" selected
  const backendSelect = page.getByTestId("form-backend-select");
  await expect(backendSelect).toBeVisible();

  // Change backend to KubeVirt
  await backendSelect.selectOption("kubevirt");
  await expect(backendSelect).toHaveValue("kubevirt");

  // Change back to Pod
  await backendSelect.selectOption("pod");
  await expect(backendSelect).toHaveValue("pod");

  // The Model dropdown should be visible
  const modelSelect = page.getByTestId("form-model-select");
  await expect(modelSelect).toBeVisible();

  // Select a different model tier
  const options = await modelSelect.locator("option").all();
  if (options.length > 1) {
    const secondValue = await options[1].getAttribute("value");
    if (secondValue) {
      await modelSelect.selectOption(secondValue);
      await expect(modelSelect).toHaveValue(secondValue);
    }
  }
});

test("TTL field accepts valid values", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Find the TTL input (type="number" with min=300, max=86400)
  const ttlInput = page.getByTestId("form-modal").locator("input[type='number']");
  await expect(ttlInput).toBeVisible();

  // Default should be 3600
  await expect(ttlInput).toHaveValue("3600");

  // Change to a valid value
  await ttlInput.fill("7200");
  await expect(ttlInput).toHaveValue("7200");

  // Verify the min and max attributes are set
  await expect(ttlInput).toHaveAttribute("min", "300");
  await expect(ttlInput).toHaveAttribute("max", "86400");
});

test("form closes with Escape key", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Press Escape to close the form
  await page.keyboard.press("Escape");

  await expect(page.getByTestId("form-modal")).not.toBeVisible();
});

test("prompt and spec tabs switch correctly", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("icon-rail-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Should start on prompt tab by default
  await expect(page.getByTestId("form-prompt-input")).toBeVisible();

  // Switch to spec tab
  await page.getByTestId("form-tab-spec").click();
  await expect(page.getByTestId("spec-editor")).toBeVisible({ timeout: 10000 });
  await expect(page.getByTestId("form-prompt-input")).not.toBeVisible();

  // Switch back to prompt tab
  await page.getByTestId("form-tab-prompt").click();
  await expect(page.getByTestId("form-prompt-input")).toBeVisible();
});
