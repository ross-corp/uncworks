// helpers.ts — shared locators and actions for the e2e suite.
//
// The prompt and spec fields are Monaco editors rather than plain textareas.
// Monaco renders its placeholder as a widget and leaves no placeholder
// attribute on the hidden textarea it uses for input, so a test cannot select
// either field the way it selected the textarea Monaco replaced. Everything
// here goes through the `data-testid` the view sets instead.
import { expect, type Page, type Locator } from "@playwright/test";

/** editor returns the Monaco wrapper for a field, such as "prompt-editor". */
export function editor(page: Page, testId: string): Locator {
  return page.getByTestId(testId);
}

/** editorInput returns the hidden textarea Monaco types into. */
export function editorInput(page: Page, testId: string): Locator {
  return page.getByTestId(testId).locator("textarea");
}

/** waitForEditor waits for Monaco to finish loading, which is lazy. */
export async function waitForEditor(page: Page, testId: string): Promise<void> {
  await expect(page.getByTestId(testId)).toBeVisible();
  await expect(editorInput(page, testId)).toBeAttached({ timeout: 15_000 });
}

/** fillEditor types into a Monaco editor and waits for the value to land. */
export async function fillEditor(page: Page, testId: string, value: string): Promise<void> {
  await waitForEditor(page, testId);
  const input = editorInput(page, testId);
  await input.click({ force: true });
  await page.keyboard.press("ControlOrMeta+A");
  // insertText, not keyboard.type. Monaco's autocomplete swallows and reorders
  // individual keystrokes, which produced values like "Do something sful".
  await page.keyboard.insertText(value);
  await expect(editorText(page, testId)).toContainText(value.split("\n")[0]);
}

/** editorText returns the rendered content of a Monaco editor. */
export function editorText(page: Page, testId: string): Locator {
  return page.getByTestId(testId).locator(".view-lines");
}

/**
 * heading selects a page heading by its accessible role.
 *
 * A bare `text=Runs` matches the sidebar link, the heading, and any row that
 * happens to say Runs, which is a strict-mode violation rather than a failure
 * worth reading.
 */
export function heading(page: Page, name: string | RegExp): Locator {
  return page.getByRole("heading", { name }).first();
}

/**
 * sanitize drops the empty-string timestamp fields a fixture tends to carry.
 *
 * The list arrives over ConnectRPC, so the response is decoded as protobuf
 * JSON, where a Timestamp cannot be "". One such field makes the whole message
 * fail to decode, and the view renders "No runs yet" with no error anywhere.
 * That silence is what made this suite look like a UI regression.
 */
const PHASE_ENUM: Record<string, string> = {
  Pending: "AGENT_RUN_PHASE_PENDING",
  Running: "AGENT_RUN_PHASE_RUNNING",
  WaitingForInput: "AGENT_RUN_PHASE_WAITING_FOR_INPUT",
  Succeeded: "AGENT_RUN_PHASE_SUCCEEDED",
  Failed: "AGENT_RUN_PHASE_FAILED",
  Cancelled: "AGENT_RUN_PHASE_CANCELLED",
};

const BACKEND_ENUM: Record<string, string> = { Pod: "BACKEND_POD" };

const ORCH_ENUM: Record<string, string> = {
  single: "ORCHESTRATION_MODE_SINGLE",
  auto: "ORCHESTRATION_MODE_AUTO",
  manual: "ORCHESTRATION_MODE_MANUAL",
  "spec-driven": "ORCHESTRATION_MODE_SPEC_DRIVEN",
};

/**
 * sanitize turns a friendly fixture into valid protobuf JSON.
 *
 * The list and the detail view both arrive over ConnectRPC, so the response is
 * decoded as protobuf JSON. That decoder is strict in two ways a fixture author
 * would not expect, and either one empties the view with no error anywhere:
 *
 *  - A Timestamp cannot be "". One `startedAt: ""` fails the whole message.
 *  - An enum must be its declared constant. `phase: "Running"` is not a value;
 *    `AGENT_RUN_PHASE_RUNNING` is. A fixture using the friendly name decoded to
 *    the zero value, which is why runs showed as queued.
 *
 * Fixtures stay readable and this does the translation.
 */
function sanitize(runs: unknown[]): unknown[] {
  const timestamps = ["startedAt", "completedAt", "retainUntil", "lastRunAt"];
  return runs.map((run) => {
    if (typeof run !== "object" || run === null) return run;
    const copy = { ...(run as Record<string, unknown>) };

    const status = copy.status;
    if (typeof status === "object" && status !== null) {
      const s = { ...(status as Record<string, unknown>) };
      for (const field of timestamps) {
        if (s[field] === "") delete s[field];
      }
      if (typeof s.phase === "string" && PHASE_ENUM[s.phase]) s.phase = PHASE_ENUM[s.phase];
      copy.status = s;
    }

    const spec = copy.spec;
    if (typeof spec === "object" && spec !== null) {
      const sp = { ...(spec as Record<string, unknown>) };
      if (typeof sp.backend === "string" && BACKEND_ENUM[sp.backend]) sp.backend = BACKEND_ENUM[sp.backend];
      if (typeof sp.orchestrationMode === "string" && ORCH_ENUM[sp.orchestrationMode]) {
        sp.orchestrationMode = ORCH_ENUM[sp.orchestrationMode];
      }
      copy.spec = sp;
    }

    for (const field of timestamps) {
      if (copy[field] === "") delete copy[field];
    }
    return copy;
  });
}

/** mockRuns stubs both the REST and the ConnectRPC list endpoints. */
export async function mockRuns(page: Page, runs: unknown[] = []): Promise<void> {
  runs = sanitize(runs);
  await page.route("**/api/v1/runs", (route) => {
    if (route.request().method() === "GET") {
      route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(runs) });
      return;
    }
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ id: "new-run-1", name: "created-run" }),
    });
  });
  // The app lists runs over ConnectRPC, not REST. Without this the call reaches
  // the dev server's proxy and returns 502, which is what left the views in an
  // error state through most of this suite.
  await page.route("**/aot.api.v1.AOTService/**", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ agentRuns: runs, nextCursor: "" }),
    }),
  );
}

/**
 * mockRun stubs one run's detail view.
 *
 * The detail view fetches over ConnectRPC (GetAgentRun) and only the traces
 * endpoint is REST, so stubbing /api/v1/runs/:id alone left the view empty.
 */
export async function mockRun(page: Page, run: Record<string, unknown>): Promise<void> {
  const [clean] = sanitize([run]) as Array<Record<string, unknown>>;
  const id = String(clean.id ?? "");

  await page.route("**/aot.api.v1.AOTService/**", (route) => {
    const url = route.request().url();
    const body = url.endsWith("/GetAgentRun")
      ? clean
      : { agentRuns: [clean], nextCursor: "" };
    route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(body) });
  });
  await page.route(`**/api/v1/runs/${id}/**`, (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: "[]" }),
  );
  await page.route("**/api/v1/runs", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify([clean]) }),
  );
}
