import { test, expect } from "@playwright/test";

test("dashboard renders with title", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("title")).toHaveText("AOT Dashboard");
});

test("dashboard shows ready status", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("status")).toHaveText("Status: Ready");
});

test("counter increments on click", async ({ page }) => {
  await page.goto("/");
  const button = page.getByTestId("counter");
  await expect(button).toHaveText("Count: 0");
  await button.click();
  await expect(button).toHaveText("Count: 1");
  await button.click();
  await expect(button).toHaveText("Count: 2");
});
