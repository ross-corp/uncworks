import { test, expect } from "@playwright/test";

test("create workspace appears in sidebar", async ({ page }) => {
  await page.goto("/");

  const wsName = `e2e-ws-${Date.now()}`;

  // Click new workspace
  await page.getByTestId("sidebar-new-workspace").click();

  // Fill workspace editor
  await expect(page.getByTestId("workspace-editor")).toBeVisible();
  await page.getByTestId("workspace-editor-name").fill(wsName);
  await page.getByTestId("workspace-editor-repo-0-url").fill("https://github.com/example/ws-repo");
  await page.getByTestId("workspace-editor-repo-0-branch").fill("main");

  // Save
  await page.getByTestId("workspace-editor-save").click();

  // Verify workspace appears in sidebar
  await expect(page.getByTestId(`sidebar-workspace-${wsName}`)).toBeVisible();
});

test("delete workspace", async ({ page }) => {
  // First create a workspace via localStorage
  const wsName = `e2e-ws-del-${Date.now()}`;
  const workspace = {
    id: `ws-del-${Date.now()}`,
    name: wsName,
    description: "To be deleted",
    repos: [{ url: "https://github.com/example/repo", branch: "main" }],
    createdAt: new Date().toISOString(),
  };

  await page.goto("/");
  await page.evaluate((ws) => {
    localStorage.setItem("aot-workspaces", JSON.stringify([ws]));
  }, workspace);
  await page.reload();

  // Verify workspace is in sidebar
  await expect(page.getByTestId(`sidebar-workspace-${wsName}`)).toBeVisible();

  // Right-click to open editor (contextmenu)
  await page.getByTestId(`sidebar-workspace-${wsName}`).click({ button: "right" });
  await expect(page.getByTestId("workspace-editor")).toBeVisible();

  // Click delete (first click shows confirmation)
  await page.getByTestId("workspace-editor-delete").click();

  // Confirm delete
  await page.getByTestId("workspace-editor-delete-confirm").click();

  // Verify removed from sidebar
  await expect(page.getByTestId(`sidebar-workspace-${wsName}`)).not.toBeVisible();
});
