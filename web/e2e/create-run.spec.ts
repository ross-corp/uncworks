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
    localStorage.setItem("aot-workspaces", JSON.stringify([ws]));
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
