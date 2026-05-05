// web/e2e-app/helpers.ts — Shared helpers for Wails app e2e tests.
import { chromium, type Browser, type Page } from "@playwright/test";

/** Default CDP endpoint Wails exposes when built with --devtools */
const CDP_ENDPOINT = process.env.UNCWORKS_CDP_URL ?? "http://localhost:34115";

/** Connect to the running UNCWORKS app via CDP and return the first page. */
export async function connectToApp(): Promise<{ browser: Browser; page: Page }> {
  const browser = await chromium.connectOverCDP(CDP_ENDPOINT);
  const contexts = browser.contexts();
  if (contexts.length === 0) {
    throw new Error(
      `No browser contexts found at ${CDP_ENDPOINT}. ` +
      "Make sure UNCWORKS is running (built with task build:app:devtools)."
    );
  }
  const pages = contexts[0].pages();
  const page = pages[0] ?? (await contexts[0].newPage());
  return { browser, page };
}
