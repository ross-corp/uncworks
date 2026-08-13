import { test, expect } from "@playwright/test";
import { heading, mockRun } from "./helpers";

// The detail view fetches over ConnectRPC. See helpers.ts for the two protobuf
// JSON rules a fixture has to satisfy before the view renders anything at all.
//
// The tab set changed: it is Logs, Traces, Files, Shell, numbered 1 to 4. The
// previous version of this file asserted five tabs including Activity and
// Verify, neither of which exists.

const SAMPLE_RUN = {
  id: "run-detail-1",
  name: "detail-test-run",
  spec: {
    backend: "Pod",
    repos: [{ url: "https://github.com/org/repo", branch: "main" }],
    prompt: "Fix the failing CI pipeline",
    ttlSeconds: 900,
    modelTier: "default",
    displayName: "Detail Test Run",
    orchestrationMode: "single",
  },
  status: {
    phase: "Running",
    message: "",
    podName: "pod-detail-1",
    traceID: "trace-1",
    startedAt: new Date().toISOString(),
  },
  createdAt: new Date().toISOString(),
};

const TABS = ["logs", "traces", "files", "shell"] as const;

test.describe("Run Detail View", () => {
  test.beforeEach(async ({ page }) => {
    await mockRun(page, SAMPLE_RUN);
    await page.goto("/run/run-detail-1");
  });

  test("loads with the run name as the heading", async ({ page }) => {
    await expect(heading(page, "Detail Test Run")).toBeVisible();
  });

  test("shows the footer shortcuts", async ({ page }) => {
    await expect(heading(page, "Detail Test Run")).toBeVisible();
    await expect(page.getByText("esc close/back")).toBeVisible();
    await expect(page.getByText("i info")).toBeVisible();
  });

  test("the tab bar shows every section", async ({ page }) => {
    for (const key of TABS) {
      await expect(page.getByTestId(`detail-tab-${key}`)).toBeVisible();
    }
  });

  test("logs is the active tab by default", async ({ page }) => {
    await expect(page.getByTestId("detail-tab-logs")).toHaveClass(/bg-accent/);
  });

  test("number keys switch tabs", async ({ page }) => {
    await expect(page.getByTestId("detail-tab-logs")).toBeVisible();

    for (const [index, key] of TABS.entries()) {
      await page.keyboard.press(String(index + 1));
      await expect(page.getByTestId(`detail-tab-${key}`)).toHaveClass(/bg-accent/);
    }

    await page.keyboard.press("1");
    await expect(page.getByTestId("detail-tab-logs")).toHaveClass(/bg-accent/);
  });

  test("clicking a tab switches to it", async ({ page }) => {
    await page.getByTestId("detail-tab-files").click();

    await expect(page.getByTestId("detail-tab-files")).toHaveClass(/bg-accent/);
    await expect(page.getByTestId("detail-tab-logs")).not.toHaveClass(/bg-accent/);
  });

  test("Esc navigates back to the run list", async ({ page }) => {
    await expect(heading(page, "Detail Test Run")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page).toHaveURL(/\/$/);
  });

  test("i toggles the info overlay", async ({ page }) => {
    await expect(heading(page, "Detail Test Run")).toBeVisible();
    await expect(page.getByText("run-detail-1", { exact: true })).toBeHidden();

    await page.keyboard.press("i");
    await expect(page.getByText("run-detail-1", { exact: true })).toBeVisible();

    await page.keyboard.press("i");
    await expect(page.getByText("run-detail-1", { exact: true })).toBeHidden();
  });
});
