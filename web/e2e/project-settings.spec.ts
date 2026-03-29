// e2e/project-settings.spec.ts — Settings tab save flow for ProjectDetailView.
// All API calls are mocked via page.route() — no real backend needed.
import { test, expect } from "@playwright/test";

const BASE_PROJECT = {
  name: "settings-project",
  displayName: "settings-project",
  description: "Original description",
  repos: [],
  devbox: { packages: ["go@1.22"] },
  defaults: {
    modelTier: "default",
    manageModelTier: "",
    implementModelTier: "",
    ttlSeconds: 900,
    orchestrationMode: "spec-driven",
    autoPush: false,
    autoPR: false,
    prBaseBranch: "main",
  },
  configRepoReady: true,
  configRepoURL: "https://github.com/org/settings-project-config",
  runCount: 0,
  totalCost: "",
};

function mockApis(
  page: import("@playwright/test").Page,
  savedProject = BASE_PROJECT
) {
  return Promise.all([
    page.route("**/api/v1/runs", (route) => {
      if (
        route.request().url().includes("/logs") ||
        route.request().url().includes("/traces")
      )
        return;
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
    page.route("**/api/v1/projects/settings-project", (route) => {
      const url = route.request().url();
      if (url.includes("/files")) return;
      if (route.request().method() === "GET") {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(BASE_PROJECT),
        });
      } else if (route.request().method() === "PUT") {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(savedProject),
        });
      } else {
        route.fallback();
      }
    }),
    page.route("**/api/v1/projects/settings-project/files", (route) => {
      route.fulfill({ status: 200, contentType: "application/json", body: "[]" });
    }),
  ]);
}

async function goToSettingsTab(page: import("@playwright/test").Page) {
  await page.goto("/projects/settings-project");
  await expect(page.locator("text=settings-project").first()).toBeVisible({
    timeout: 8000,
  });
  await page.locator("button", { hasText: "Settings" }).first().click();
  await expect(page.locator("text=Description")).toBeVisible({ timeout: 5000 });
}

test.describe("Project Settings — Save flow", () => {
  test("Settings tab loads with description field pre-filled", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const descInput = page.locator("input[placeholder='Project description']");
    await expect(descInput).toBeVisible();
    await expect(descInput).toHaveValue("Original description");
  });

  test("'Save Settings' button is not visible when settings are unchanged", async ({
    page,
  }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    // Save Settings button should not be shown until something is dirty
    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).not.toBeVisible();
  });

  test("editing description makes Save Settings button appear", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const descInput = page.locator("input[placeholder='Project description']");
    await descInput.fill("Updated description");

    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).toBeVisible({ timeout: 3000 });
  });

  test("'Discard' button resets description to original value", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const descInput = page.locator("input[placeholder='Project description']");
    await descInput.fill("Dirty value I will discard");

    // Save Settings button appears
    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).toBeVisible({ timeout: 3000 });

    // Click Discard
    const discardBtn = page.locator("button", { hasText: "Discard" }).first();
    await expect(discardBtn).toBeVisible();
    await discardBtn.click();

    // Input should be reverted
    await expect(descInput).toHaveValue("Original description");

    // Save Settings button should disappear
    await expect(saveBtn).not.toBeVisible({ timeout: 3000 });
  });

  test("clicking Save Settings calls PUT and hides the button", async ({ page }) => {
    const updatedProject = { ...BASE_PROJECT, description: "Updated description" };
    await mockApis(page, updatedProject);
    await goToSettingsTab(page);

    const descInput = page.locator("input[placeholder='Project description']");
    await descInput.fill("Updated description");

    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).toBeVisible({ timeout: 3000 });
    await saveBtn.click();

    // After a successful save the button should disappear (dirty flag cleared)
    await expect(saveBtn).not.toBeVisible({ timeout: 5000 });
  });

  test("Settings tab shows existing devbox packages", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    // go@1.22 is in the mock project's devbox packages
    await expect(page.locator("text=go@1.22")).toBeVisible({ timeout: 5000 });
  });

  test("adding a devbox package makes settings dirty", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const pkgInput = page.locator("input[placeholder='e.g. go@1.22']");
    await expect(pkgInput).toBeVisible();
    await pkgInput.fill("nodejs@20");
    await pkgInput.press("Enter");

    // New package should appear in the list
    await expect(page.locator("text=nodejs@20")).toBeVisible({ timeout: 3000 });

    // Save Settings button should appear
    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).toBeVisible({ timeout: 3000 });
  });

  test("removing a devbox package makes settings dirty", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    // go@1.22 should be visible; find its remove button
    await expect(page.locator("text=go@1.22")).toBeVisible({ timeout: 5000 });
    const removeBtn = page
      .locator("text=go@1.22")
      .locator("..")
      .locator("button");
    await expect(removeBtn).toBeVisible();
    await removeBtn.click();

    // Package should be gone
    await expect(page.locator("text=go@1.22")).not.toBeVisible({ timeout: 3000 });

    // Save Settings button should appear
    const saveBtn = page.locator("button", { hasText: "Save Settings" });
    await expect(saveBtn).toBeVisible({ timeout: 3000 });
  });

  test("Config Repo URL is displayed in settings", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    await expect(
      page.locator("text=https://github.com/org/settings-project-config")
    ).toBeVisible({ timeout: 5000 });
  });

  test("Rename button appears next to project name in settings", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    await expect(page.locator("button", { hasText: "Rename" })).toBeVisible({
      timeout: 5000,
    });
  });

  test("clicking Rename shows rename input", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const renameBtn = page.locator("button", { hasText: "Rename" });
    await renameBtn.click();

    // The input should appear pre-filled with the current name
    const renameInput = page
      .locator("input")
      .filter({ hasValue: "settings-project" });
    await expect(renameInput).toBeVisible({ timeout: 3000 });
  });

  test("Esc cancels rename and hides the input", async ({ page }) => {
    await mockApis(page);
    await goToSettingsTab(page);

    const renameBtn = page.locator("button", { hasText: "Rename" });
    await renameBtn.click();

    const renameInput = page
      .locator("input")
      .filter({ hasValue: "settings-project" });
    await expect(renameInput).toBeVisible({ timeout: 3000 });

    await renameInput.press("Escape");

    await expect(renameInput).not.toBeVisible({ timeout: 3000 });
    // Original name text should be visible again
    await expect(page.locator("text=settings-project").first()).toBeVisible();
  });
});
