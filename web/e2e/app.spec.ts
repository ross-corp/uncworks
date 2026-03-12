import { test, expect } from "@playwright/test";

test("dashboard renders with title", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("title")).toHaveText("AOT Dashboard");
});

test("dashboard shows connection status", async ({ page }) => {
  await page.goto("/");
  // Status should transition from Loading to Connected (or Error if no API)
  const status = page.getByTestId("status");
  await expect(status).toBeVisible();
  // Wait for data to load (up to 10s)
  await expect(status).toContainText("Status:", { timeout: 10000 });
});

test("agent run list loads from API", async ({ page }) => {
  await page.goto("/");
  const list = page.getByTestId("agent-run-list");
  await expect(list).toBeVisible();

  // Wait for runs to appear (loaded via ConnectRPC from live API)
  // The list should either have items or show empty state
  const hasRuns = await list.locator("li").count();
  if (hasRuns > 0) {
    // Verify at least one run is visible with a phase badge
    const firstRun = list.locator("li").first();
    await expect(firstRun).toBeVisible();
    // Each run shows name and phase
    await expect(firstRun.locator("strong")).toBeVisible();
  } else {
    await expect(page.getByTestId("empty-state")).toHaveText("No agent runs");
  }
});

test("agent run list displays runs fetched via ConnectRPC", async ({ page }) => {
  await page.goto("/");

  // Wait for status to show Connected with run count
  await expect(page.getByTestId("status")).toContainText("Connected", {
    timeout: 10000,
  });

  // Verify the list loaded from the API
  const list = page.getByTestId("agent-run-list");
  const items = list.locator("li");
  const count = await items.count();
  expect(count).toBeGreaterThan(0);

  // Verify each item has required structure (name + phase)
  for (let i = 0; i < count; i++) {
    const item = items.nth(i);
    await expect(item.locator("strong")).toBeVisible();
    // Phase badge should have a data-testid
    const phaseEl = item.locator("[data-testid^='phase-']");
    await expect(phaseEl).toBeVisible();
    const phaseText = await phaseEl.textContent();
    expect(["Pending", "Running", "WaitingForInput", "Succeeded", "Failed", "Cancelled"]).toContain(phaseText);
  }
});

test("clicking an agent run shows detail panel via ConnectRPC data", async ({ page }) => {
  await page.goto("/");

  // Wait for runs to load
  await expect(page.getByTestId("status")).toContainText("Connected", {
    timeout: 10000,
  });

  // Initially no selection
  await expect(page.getByTestId("no-selection")).toBeVisible();

  // Click first run
  const firstRun = page.getByTestId("agent-run-list").locator("li").first();
  await firstRun.click();

  // Detail panel should show the run's data
  await expect(page.getByTestId("detail-name")).toBeVisible();
  await expect(page.getByTestId("detail-phase")).toBeVisible();
  await expect(page.getByTestId("detail-backend")).toBeVisible();
  await expect(page.getByTestId("detail-prompt")).toBeVisible();

  // Phase should be a valid phase string
  const phase = await page.getByTestId("detail-phase").textContent();
  expect(["Pending", "Running", "WaitingForInput", "Succeeded", "Failed", "Cancelled"]).toContain(phase);
});

test("switching agent run selection updates detail", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("status")).toContainText("Connected", {
    timeout: 10000,
  });

  const items = page.getByTestId("agent-run-list").locator("li");
  const count = await items.count();
  if (count < 2) {
    test.skip(count < 2, "Need at least 2 runs to test selection switching");
    return;
  }

  // Click first run
  await items.first().click();
  const firstName = await page.getByTestId("detail-name").textContent();

  // Click second run
  await items.nth(1).click();
  const secondName = await page.getByTestId("detail-name").textContent();

  // Names should differ (different runs selected)
  expect(firstName).not.toBe(secondName);
});
