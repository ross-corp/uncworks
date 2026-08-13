import { test, expect } from "@playwright/test";
import { mockRuns } from "./helpers";

// This file replaces command-palette.spec.ts. The cmdk palette was deleted in
// March 2026 by "fix: ui audit — broken routes, dead code, layout fixes", which
// removed CommandPaletteNew.tsx and left its nine tests behind. Ctrl+K now
// toggles the Copilot panel, so that is what is tested here.

const PANEL_INPUT = "input[placeholder*='Ask a question']";

test.describe("Copilot panel", () => {
  test.beforeEach(async ({ page }) => {
    await mockRuns(page, []);
    await page.goto("/");
  });

  test("Ctrl+K opens the panel", async ({ page }) => {
    await expect(page.locator(PANEL_INPUT)).toBeHidden();

    await page.keyboard.press("Control+k");
    await expect(page.locator(PANEL_INPUT)).toBeVisible();
  });

  test("Ctrl+K toggles the panel closed again", async ({ page }) => {
    await page.keyboard.press("Control+k");
    await expect(page.locator(PANEL_INPUT)).toBeVisible();

    await page.keyboard.press("Control+k");
    await expect(page.locator(PANEL_INPUT)).toBeHidden();
  });

  test("Esc closes the panel", async ({ page }) => {
    await page.keyboard.press("Control+k");
    await expect(page.locator(PANEL_INPUT)).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page.locator(PANEL_INPUT)).toBeHidden();
  });

  test("the panel accepts input and enables Send", async ({ page }) => {
    await page.keyboard.press("Control+k");
    const input = page.locator(PANEL_INPUT);
    await expect(input).toBeVisible();

    const send = page.getByRole("button", { name: /^send$/i });
    await expect(send).toBeDisabled();

    await input.fill("what is this run doing?");
    await expect(send).toBeEnabled();
  });

  test("Ctrl+K does not fire while a text field has focus", async ({ page }) => {
    // The handler ignores the shortcut when an input is focused, so typing a
    // literal ⌘K into a filter box does not open the panel over it.
    const filterInput = page.locator("input[placeholder^='filter']").first();
    await filterInput.click();
    await page.keyboard.press("Control+k");

    await expect(page.locator(PANEL_INPUT)).toBeHidden();
  });
});
