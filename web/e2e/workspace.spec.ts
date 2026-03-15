import { test, expect } from "@playwright/test";

test("workspace preset fills repos in create form", async ({ page }) => {
  // Inject a workspace into localStorage before loading
  const wsName = `e2e-ws-${Date.now()}`;
  const workspace = {
    id: `ws-${Date.now()}`,
    name: wsName,
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
  await page.getByTestId("sidebar-new-run").click();
  await expect(page.getByTestId("form-modal")).toBeVisible();

  // Click the workspace button
  await page.getByTestId(`form-workspace-${wsName}`).click();

  // Verify repo fields are pre-filled
  await expect(page.getByTestId("form-repo-row-0-url")).toHaveValue("https://github.com/preset/repo-one");
  await expect(page.getByTestId("form-repo-row-0-branch")).toHaveValue("develop");
});
