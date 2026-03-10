import { test, expect } from "@playwright/test";

test("dashboard renders with title", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("title")).toHaveText("AOT Dashboard");
});

test("dashboard shows ready status", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("status")).toHaveText("Status: Ready");
});

test("agent run list displays runs", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("agent-run-list")).toBeVisible();
  await expect(page.getByTestId("run-ar-1")).toBeVisible();
  await expect(page.getByTestId("run-ar-2")).toBeVisible();
  await expect(page.getByTestId("run-ar-3")).toBeVisible();
});

test("agent run list shows phase badges", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("phase-ar-1")).toHaveText("Running");
  await expect(page.getByTestId("phase-ar-2")).toHaveText("Succeeded");
  await expect(page.getByTestId("phase-ar-3")).toHaveText("Pending");
});

test("clicking an agent run shows detail panel", async ({ page }) => {
  await page.goto("/");

  // Initially no selection
  await expect(page.getByTestId("no-selection")).toBeVisible();

  // Click on first run
  await page.getByTestId("run-ar-1").click();

  await expect(page.getByTestId("detail-name")).toHaveText("fix-auth-bug");
  await expect(page.getByTestId("detail-phase")).toHaveText("Running");
  await expect(page.getByTestId("detail-backend")).toHaveText("Pod");
  await expect(page.getByTestId("detail-pod")).toHaveText("agentrun-fix-auth-bug-pod");
  await expect(page.getByTestId("detail-trace")).toHaveText("abc123def456");
});

test("switching agent run selection updates detail", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("run-ar-1").click();
  await expect(page.getByTestId("detail-name")).toHaveText("fix-auth-bug");

  await page.getByTestId("run-ar-2").click();
  await expect(page.getByTestId("detail-name")).toHaveText("add-tests");
  await expect(page.getByTestId("detail-phase")).toHaveText("Succeeded");
});
