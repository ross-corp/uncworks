import { test, expect } from "@playwright/test";
import { mockRun } from "./helpers";

// The positive half of this file, "at least one of implement, manage, or system
// is visible", needed a live run and asserted whichever label happened to
// appear. The label set is a pure mapping, so it is pinned by a unit test in
// src/lib/__tests__/role-styles.test.ts instead. What is left here is the part
// that needs a browser: the deprecated labels must not reach the page.

const RUN = {
  id: "feed-run-1",
  name: "feed-run",
  spec: { backend: "Pod", repos: [], prompt: "p", modelTier: "default", displayName: "Feed Run" },
  status: { phase: "Running", message: "" },
  createdAt: new Date().toISOString(),
};

test.describe("Activity Feed", () => {
  test("does not show the deprecated role labels", async ({ page }) => {
    await mockRun(page, RUN);
    await page.goto(`/run/${RUN.id}`);

    await expect(page.getByTestId("detail-tab-logs")).toBeVisible();

    const body = (await page.textContent("body")) ?? "";
    expect(body).not.toContain(">neph<");
    expect(body).not.toContain(">unc<");
  });
});
