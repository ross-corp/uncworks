import { test, expect } from "@playwright/test";

test.describe("Error handling", () => {
  test("API 500 on list runs shows error gracefully", async ({ page }) => {
    // Intercept the ConnectRPC ListAgentRuns call and return 500
    await page.route("**/aot.api.v1.AOTService/ListAgentRuns", (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ code: "internal", message: "server error" }),
      })
    );

    await page.goto("/");

    // The run list container should still render (empty or error state)
    await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 10000 });
  });

  test("API 500 on create run shows error toast", async ({ page }) => {
    await page.goto("/");

    // Let list load normally, but intercept create calls
    await page.route("**/aot.api.v1.AOTService/CreateAgentRun", (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ code: "internal", message: "create failed" }),
      })
    );

    // Open form and fill in valid data
    await page.getByTestId("icon-rail-new-run").click();
    await expect(page.getByTestId("form-modal")).toBeVisible();

    await page.getByTestId("form-name-input").fill("e2e-error-test");
    await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/repo");
    await page.getByTestId("form-repo-row-0-branch").fill("main");
    await page.getByTestId("form-prompt-input").fill("This should fail");

    // Submit
    await page.getByTestId("form-submit").click();

    // Error toast should appear
    await expect(page.getByTestId("toast")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("toast")).toContainText(/failed/i);
  });

  test("form validation: empty name prevents submission", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("icon-rail-new-run").click();
    await expect(page.getByTestId("form-modal")).toBeVisible();

    // Fill repo and prompt but leave name empty
    await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/repo");
    await page.getByTestId("form-repo-row-0-branch").fill("main");
    await page.getByTestId("form-prompt-input").fill("Some prompt");

    await page.getByTestId("form-submit").click();

    // Form should remain open (validation prevented submission)
    await expect(page.getByTestId("form-modal")).toBeVisible();
  });

  test("form validation: empty repo URL prevents submission", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("icon-rail-new-run").click();
    await expect(page.getByTestId("form-modal")).toBeVisible();

    // Fill name and prompt but leave repo URL empty
    await page.getByTestId("form-name-input").fill("e2e-no-repo");
    await page.getByTestId("form-repo-row-0-url").fill("");
    await page.getByTestId("form-prompt-input").fill("Some prompt");

    await page.getByTestId("form-submit").click();

    // Form should remain open (validation prevents submission with no valid repos)
    await expect(page.getByTestId("form-modal")).toBeVisible();
  });

  test("form validation: empty prompt prevents submission", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("icon-rail-new-run").click();
    await expect(page.getByTestId("form-modal")).toBeVisible();

    // Fill name and repo but leave prompt empty
    await page.getByTestId("form-name-input").fill("e2e-no-prompt");
    await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/repo");
    await page.getByTestId("form-repo-row-0-branch").fill("main");

    await page.getByTestId("form-submit").click();

    // Form should remain open
    await expect(page.getByTestId("form-modal")).toBeVisible();
  });

  test("network recovery: list resumes after API comes back", async ({ page }) => {
    // Start with a working API
    await page.goto("/");
    await expect(page.getByTestId("run-list")).toBeVisible({ timeout: 10000 });

    // Block all API calls to simulate network failure
    await page.route("**/aot.api.v1.AOTService/ListAgentRuns", (route) =>
      route.abort("connectionrefused")
    );

    // Wait for a poll cycle to fail (polling is every 5s)
    await page.waitForTimeout(6000);

    // The run list should still be rendered (stale data or empty state)
    await expect(page.getByTestId("run-list")).toBeVisible();

    // Restore the API by removing the route
    await page.unroute("**/aot.api.v1.AOTService/ListAgentRuns");

    // Wait for the next poll to succeed and verify the list is still visible
    await page.waitForTimeout(6000);
    await expect(page.getByTestId("run-list")).toBeVisible();
  });

  test("successful create run shows success toast", async ({ page }) => {
    // Mock the create endpoint to return a success response
    await page.route("**/aot.api.v1.AOTService/CreateAgentRun", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          agentRun: {
            id: "e2e-toast-test",
            name: "e2e-toast-test",
            spec: {
              backend: "Pod",
              repos: [{ url: "https://github.com/example/repo", branch: "main" }],
              prompt: "test",
              ttlSeconds: 3600,
              modelTier: "default",
              envVars: {},
              devboxConfig: "",
            },
            status: {
              phase: "Pending",
              message: "",
              podName: "",
              traceID: "",
              startedAt: "",
              completedAt: "",
            },
            createdAt: new Date().toISOString(),
          },
        }),
      })
    );

    await page.goto("/");

    await page.getByTestId("icon-rail-new-run").click();
    await expect(page.getByTestId("form-modal")).toBeVisible();

    await page.getByTestId("form-name-input").fill("e2e-toast-test");
    await page.getByTestId("form-repo-row-0-url").fill("https://github.com/example/repo");
    await page.getByTestId("form-repo-row-0-branch").fill("main");
    await page.getByTestId("form-prompt-input").fill("Test toast");

    await page.getByTestId("form-submit").click();

    // Form should close on success
    await expect(page.getByTestId("form-modal")).not.toBeVisible({ timeout: 10000 });

    // Success toast should appear
    await expect(page.getByTestId("toast")).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId("toast")).toContainText(/created/i);
  });
});
