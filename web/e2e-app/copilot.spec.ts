// web/e2e-app/copilot.spec.ts — Copilot message round-trip in the real app.
// Requires a live cluster with OpenRouter configured.
import { test, expect } from "@playwright/test";
import { connectToApp } from "./helpers";
import type { Browser } from "@playwright/test";

let browser: Browser;

test.beforeAll(async () => {
  ({ browser } = await connectToApp());
});

test.afterAll(async () => {
  await browser.close();
});

test("copilot sends message and receives response", async () => {
  const page = browser.contexts()[0].pages()[0];

  // Open the copilot panel if it's not already open (look for toggle button)
  const copilotToggle = page.getByRole("button", { name: /copilot|chat/i }).first();
  if (await copilotToggle.isVisible()) {
    await copilotToggle.click();
  }

  // Find the chat input
  const input = page.locator("textarea, input[placeholder*=message i], input[placeholder*=ask i]").last();
  await expect(input).toBeVisible({ timeout: 5_000 });

  // Type a message and submit
  await input.fill("Say exactly: hello");
  await input.press("Enter");

  // Wait for assistant response to appear (30s timeout for LLM)
  const assistantMessage = page.locator("[data-role=assistant], .assistant-message, [data-testid=assistant-reply]").first();
  await expect(assistantMessage).toBeVisible({ timeout: 30_000 });
  await expect(assistantMessage).not.toBeEmpty();
});
