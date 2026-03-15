import { test, expect } from "@playwright/test";

test("Monaco loads on spec tab", async ({ page }) => {
  await page.goto("/");

  // Open form
  await page.getByTestId("sidebar-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Switch to spec tab
  await page.getByTestId("form-tab-spec").click();

  // Wait for spec editor to be visible (lazy loaded)
  await expect(page.getByTestId("spec-editor")).toBeVisible({ timeout: 10000 });

  // Verify Monaco loaded (should not show loading text)
  await expect(page.getByText("Loading editor...")).not.toBeVisible({ timeout: 10000 });
});

test("GitHub Load modal", async ({ page }) => {
  await page.goto("/");

  // Open form and switch to spec tab
  await page.getByTestId("sidebar-new-run").click();
  await page.getByTestId("form-tab-spec").click();
  await expect(page.getByTestId("spec-editor")).toBeVisible({ timeout: 10000 });

  // Mock the API for spec loading
  await page.route("**/api/v1/specs/pull**", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      json: { content: "# Loaded Spec\n\nThis was loaded from GitHub.", sha: "abc123" },
    })
  );

  // Click "Load from GitHub"
  await page.getByText("Load from GitHub").click();

  // GitHub modal should appear
  await expect(page.getByTestId("github-modal")).toBeVisible();

  // Fill in repo and path
  await page.getByTestId("github-modal-repo").fill("example/test-repo");
  await page.getByTestId("github-modal-path").fill("specs/test.md");

  // Submit
  await page.getByTestId("github-modal-submit").click();

  // Modal should close
  await expect(page.getByTestId("github-modal")).not.toBeVisible({ timeout: 5000 });
});
