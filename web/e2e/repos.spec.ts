import { test, expect } from "@playwright/test";

test("add repo to registry", async ({ page }) => {
  await page.goto("/");

  // Navigate to repos view by clicking "Manage Repos" in sidebar
  await page.getByText("Manage Repos").click();

  // Fill in a repo URL
  const repoUrl = `https://github.com/e2e-test/repo-${Date.now()}`;
  await page.getByTestId("repos-add-input").fill(repoUrl);

  // Click add
  await page.getByTestId("repos-add-button").click();

  // Verify the repo appears in the list
  await expect(page.getByText(repoUrl)).toBeVisible();
});

test("remove repo", async ({ page }) => {
  await page.goto("/");

  // Inject a repo into localStorage
  const repoUrl = `https://github.com/e2e-test/removable-repo-${Date.now()}`;
  await page.evaluate((url) => {
    const existing = JSON.parse(localStorage.getItem("aot-repo-registry") || "[]");
    existing.push(url);
    localStorage.setItem("aot-repo-registry", JSON.stringify(existing));
  }, repoUrl);

  await page.reload();

  // Navigate to repos view
  await page.getByText("Manage Repos").click();

  // Verify the repo is visible
  await expect(page.getByText(repoUrl)).toBeVisible();

  // Click remove (hover is needed since opacity-0 by default, use force click)
  const row = page.getByText(repoUrl).locator("../..");
  await row.locator("button:has-text('Remove')").click({ force: true });

  // Verify repo is removed
  await expect(page.getByText(repoUrl)).not.toBeVisible();
});
