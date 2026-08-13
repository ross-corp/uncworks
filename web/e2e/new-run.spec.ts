import { test, expect } from "@playwright/test";
import { editorText, fillEditor, heading, mockRuns, waitForEditor } from "./helpers";

// The prompt and spec fields are Monaco editors. See helpers.ts for why a test
// cannot reach them through a placeholder attribute.

test.describe("New Run View", () => {
  test.beforeEach(async ({ page }) => {
    await mockRuns(page);
  });

  test("/new loads with the prompt editor", async ({ page }) => {
    await page.goto("/new");

    await expect(heading(page, "New Run")).toBeVisible();
    await waitForEditor(page, "prompt-editor");
  });

  test("can type a prompt", async ({ page }) => {
    await page.goto("/new");

    await fillEditor(page, "prompt-editor", "Fix all the broken tests in the repo");
    await expect(editorText(page, "prompt-editor")).toContainText("Fix all the broken tests");
  });

  test("the Prompt and Spec toggle switches mode", async ({ page }) => {
    await page.goto("/new");
    await waitForEditor(page, "prompt-editor");

    const promptTab = page.getByRole("button", { name: /^prompt$/i });
    const specTab = page.getByRole("button", { name: /^spec$/i });

    // Prompt mode shows only the prompt editor.
    await expect(page.getByTestId("spec-editor")).toBeHidden();

    await specTab.click();
    await waitForEditor(page, "spec-editor");

    await promptTab.click();
    await expect(page.getByTestId("spec-editor")).toBeHidden();
  });

  test("spec mode shows both editors and both accept text", async ({ page }) => {
    await page.goto("/new");
    await waitForEditor(page, "prompt-editor");

    await page.getByRole("button", { name: /^spec$/i }).click();
    await waitForEditor(page, "spec-editor");

    await fillEditor(page, "prompt-editor", "Run the spec");
    await fillEditor(page, "spec-editor", "## Task");

    await expect(editorText(page, "prompt-editor")).toContainText("Run the spec");
    await expect(editorText(page, "spec-editor")).toContainText("## Task");
  });

  test("Cancel navigates back to the run list", async ({ page }) => {
    await page.goto("/new");

    // The header and the footer both offer Cancel, so take the footer's.
    await page.getByRole("button", { name: /^cancel$/i }).last().click();
    await expect(page).toHaveURL(/\/$/);
  });

  test("Run is disabled until the prompt has text", async ({ page }) => {
    await page.goto("/new");
    await waitForEditor(page, "prompt-editor");

    const runBtn = page.getByRole("button", { name: /^run$/i });
    await expect(runBtn).toBeDisabled();

    await fillEditor(page, "prompt-editor", "Do something useful");
    await expect(runBtn).toBeEnabled();
  });

  test("the run list is reachable from the header", async ({ page }) => {
    await page.goto("/new");
    await expect(heading(page, "New Run")).toBeVisible();

    await page.goto("/");
    await expect(heading(page, "Runs")).toBeVisible();
  });
});
