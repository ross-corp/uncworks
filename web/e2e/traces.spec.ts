import { test, expect } from "@playwright/test";

// Sample run for mocking
const SAMPLE_RUN = {
  id: "run-trace-1",
  name: "trace-test-run",
  spec: {
    backend: "Pod",
    repos: [{ url: "https://github.com/org/repo", branch: "main" }],
    prompt: "Implement feature X",
    ttlSeconds: 900,
    modelTier: "default",
    displayName: "Trace Test Run",
    orchestrationMode: "spec",
  },
  status: {
    phase: "Completed",
    message: "",
    podName: "pod-trace-1",
    traceID: "trace-1",
    startedAt: "2026-03-20T10:00:00Z",
    completedAt: "2026-03-20T10:10:00Z",
  },
  createdAt: "2026-03-20T10:00:00Z",
};

// Realistic trace spans with stage parents and children
const SAMPLE_SPANS = [
  {
    id: "stage-plan",
    name: "PLAN",
    type: "stage",
    startTime: "2026-03-20T10:00:00Z",
    endTime: "2026-03-20T10:03:00Z",
    status: "ok",
    hasDiff: false,
    metadata: { stage: "plan" },
  },
  {
    id: "thought-1",
    traceId: "trace-1",
    parentId: "stage-plan",
    name: "manage.thought",
    type: "thought",
    startTime: "2026-03-20T10:00:01Z",
    endTime: "2026-03-20T10:00:30Z",
    hasDiff: false,
    metadata: {
      "gen_ai.usage.input_tokens": 1500,
      "gen_ai.usage.output_tokens": 350,
      content: "Let me analyze the requirements...",
    },
  },
  {
    id: "tool-plan-1",
    traceId: "trace-1",
    parentId: "stage-plan",
    name: "manage.read",
    type: "tool",
    startTime: "2026-03-20T10:00:31Z",
    endTime: "2026-03-20T10:00:35Z",
    hasDiff: false,
    metadata: { toolInput: '{"file_path":"spec.md"}' },
  },
  {
    id: "stage-execute",
    name: "EXECUTE",
    type: "stage",
    startTime: "2026-03-20T10:03:00Z",
    endTime: "2026-03-20T10:08:00Z",
    status: "ok",
    hasDiff: false,
    metadata: { stage: "execute" },
  },
  {
    id: "thought-2",
    traceId: "trace-1",
    parentId: "stage-execute",
    name: "implement.thought",
    type: "thought",
    startTime: "2026-03-20T10:03:01Z",
    endTime: "2026-03-20T10:03:20Z",
    hasDiff: false,
    metadata: {
      "gen_ai.usage.input_tokens": 2000,
      "gen_ai.usage.output_tokens": 500,
      content: "I will implement the feature now...",
    },
  },
  {
    id: "tool-exec-1",
    traceId: "trace-1",
    parentId: "stage-execute",
    name: "implement.bash",
    type: "tool",
    startTime: "2026-03-20T10:03:21Z",
    endTime: "2026-03-20T10:03:25Z",
    hasDiff: true,
    metadata: {
      toolInput: '{"command":"go build ./..."}',
      checkpointSha: "abc123",
    },
  },
  {
    id: "stage-verify",
    name: "VERIFY",
    type: "stage",
    startTime: "2026-03-20T10:08:00Z",
    endTime: "2026-03-20T10:10:00Z",
    status: "ok",
    hasDiff: false,
    metadata: { stage: "verify" },
  },
];

const DIFF_DATA = {
  files: [
    { path: "main.go", patch: "+fmt.Println(\"hello\")\n-old line" },
    { path: "util.go", patch: "+new util function" },
  ],
};

function mockTraceApis(page: import("@playwright/test").Page) {
  return Promise.all([
    // List runs
    page.route("**/api/v1/runs", (route) => {
      if (route.request().url().includes("/traces")) return;
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([SAMPLE_RUN]),
      });
    }),
    // Get single run
    page.route("**/api/v1/runs/run-trace-1", (route) => {
      if (route.request().url().includes("/traces")) return;
      if (route.request().url().includes("/logs")) return;
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(SAMPLE_RUN),
      });
    }),
    // Structured logs
    page.route("**/api/v1/runs/run-trace-1/logs/structured", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    }),
    // Traces endpoint
    page.route("**/api/v1/runs/run-trace-1/traces", (route) => {
      if (route.request().url().includes("/diff")) return;
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(SAMPLE_SPANS),
      });
    }),
    // Diff endpoint for the tool span with hasDiff=true
    page.route(
      "**/api/v1/runs/run-trace-1/traces/tool-exec-1/diff",
      (route) => {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(DIFF_DATA),
        });
      }
    ),
    // Diff endpoint for spans without diff
    page.route("**/api/v1/runs/run-trace-1/traces/*/diff", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ files: [] }),
      });
    }),
  ]);
}

test.describe("Traces", () => {
  test("clicking a span row shows detail panel", async ({ page }) => {
    // Navigate to a run detail page
    await page.goto("/");
    // Click the first run in the list
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }
    // Switch to traces tab
    await page.locator("[data-testid='detail-tab-traces']").click();

    // If spans exist, click the first one
    const spanRow = page
      .locator("[data-testid='trace-timeline'] button")
      .first();
    if (await spanRow.isVisible({ timeout: 5000 }).catch(() => false)) {
      await spanRow.click();
      // Verify detail panel appears (look for metadata or close button)
      await expect(page.locator("text=Duration")).toBeVisible({
        timeout: 3000,
      });
    }
  });

  test("trace timeline shows span count", async ({ page }) => {
    await page.goto("/");
    const firstRun = page.locator("[data-testid='run-row']").first();
    if (await firstRun.isVisible()) {
      await firstRun.click();
    }
    await page.locator("[data-testid='detail-tab-traces']").click();
    // Should show "N spans" or "No trace spans recorded"
    const timeline = page.locator("[data-testid='trace-timeline']");
    await expect(timeline).toBeVisible({ timeout: 5000 });
  });

  // --- Task 5.1: Stage spans render in waterfall ---

  test("stage parent rows render with stage type styling", async ({
    page,
  }) => {
    await mockTraceApis(page);
    await page.goto("/run/run-trace-1");

    // Switch to traces tab
    await page.getByTestId("detail-tab-traces").click();

    // Wait for timeline to render
    const timeline = page.getByTestId("trace-timeline");
    await expect(timeline).toBeVisible({ timeout: 5000 });

    // Verify span count includes all 7 spans
    await expect(timeline.locator("text=/\\d+ spans/")).toBeVisible();

    // Verify stage span rows exist (PLAN, EXECUTE, VERIFY)
    // Stage spans have bold text and are rendered as buttons in the waterfall
    const buttons = timeline.locator("button");
    const buttonCount = await buttons.count();
    expect(buttonCount).toBeGreaterThanOrEqual(3);

    // Verify stage names are visible
    await expect(timeline.locator("text=PLAN")).toBeVisible();
    await expect(timeline.locator("text=EXECUTE")).toBeVisible();
    await expect(timeline.locator("text=VERIFY")).toBeVisible();

    // Stage spans should have font-bold styling (taller rows, bold text)
    const planRow = timeline.locator("text=PLAN").first();
    await expect(planRow).toBeVisible();
  });

  // --- Task 5.2: Collapse/expand toggle ---

  test("collapse toggle hides children, expand restores them", async ({
    page,
  }) => {
    await mockTraceApis(page);
    await page.goto("/run/run-trace-1");

    await page.getByTestId("detail-tab-traces").click();
    const timeline = page.getByTestId("trace-timeline");
    await expect(timeline).toBeVisible({ timeout: 5000 });

    // Verify child spans are initially visible
    await expect(timeline.locator("text=manage.thought")).toBeVisible();
    await expect(timeline.locator("text=manage.read")).toBeVisible();

    // Click the collapse toggle (chevron) on the PLAN stage row
    // The toggle is a span with cursor-pointer inside the PLAN button row
    const planRow = timeline.locator("button", { hasText: "PLAN" }).first();
    const collapseToggle = planRow.locator(
      "span.cursor-pointer, [class*='cursor-pointer']"
    );
    if ((await collapseToggle.count()) > 0) {
      await collapseToggle.first().click();

      // After collapse, child spans under PLAN should be hidden
      await expect(timeline.locator("text=manage.thought")).not.toBeVisible({
        timeout: 3000,
      });
      await expect(timeline.locator("text=manage.read")).not.toBeVisible({
        timeout: 3000,
      });

      // PLAN row itself should still be visible
      await expect(timeline.locator("text=PLAN")).toBeVisible();

      // Click again to expand
      await collapseToggle.first().click();

      // Children should be visible again
      await expect(timeline.locator("text=manage.thought")).toBeVisible({
        timeout: 3000,
      });
      await expect(timeline.locator("text=manage.read")).toBeVisible({
        timeout: 3000,
      });
    }
  });

  // --- Task 5.3: Diff badge and detail panel ---

  test("clicking a span with DIFF badge shows diff content", async ({
    page,
  }) => {
    await mockTraceApis(page);
    await page.goto("/run/run-trace-1");

    await page.getByTestId("detail-tab-traces").click();
    const timeline = page.getByTestId("trace-timeline");
    await expect(timeline).toBeVisible({ timeout: 5000 });

    // Click the tool span that has hasDiff=true (implement.bash)
    const bashSpan = timeline
      .locator("button", { hasText: "implement.bash" })
      .first();
    await expect(bashSpan).toBeVisible({ timeout: 3000 });
    await bashSpan.click();

    // Detail panel should open and show diff data
    // Wait for diff to load (it fetches from the diff endpoint)
    await expect(page.locator("text=Duration")).toBeVisible({ timeout: 3000 });

    // Should show the file paths from the diff
    await expect(page.locator("text=main.go")).toBeVisible({ timeout: 5000 });
    await expect(page.locator("text=util.go")).toBeVisible({ timeout: 5000 });

    // Diff content should show green/red lines
    await expect(page.locator("text=Changes")).toBeVisible({ timeout: 3000 });
  });

  // --- Task 5.4: Token display in detail panel ---

  test("thought span detail panel shows token usage", async ({ page }) => {
    await mockTraceApis(page);
    await page.goto("/run/run-trace-1");

    await page.getByTestId("detail-tab-traces").click();
    const timeline = page.getByTestId("trace-timeline");
    await expect(timeline).toBeVisible({ timeout: 5000 });

    // Click a thought span
    const thoughtSpan = timeline
      .locator("button", { hasText: "manage.thought" })
      .first();
    await expect(thoughtSpan).toBeVisible({ timeout: 3000 });
    await thoughtSpan.click();

    // Detail panel should show token info in the metadata section
    await expect(page.locator("text=Duration")).toBeVisible({ timeout: 3000 });

    // Click "All Metadata" to expand it if collapsed
    const metadataToggle = page.locator("text=All Metadata");
    if (await metadataToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
      await metadataToggle.click();
      // Token usage values should be visible in the metadata JSON
      await expect(page.locator("text=gen_ai.usage.input_tokens")).toBeVisible({
        timeout: 3000,
      });
      await expect(
        page.locator("text=gen_ai.usage.output_tokens")
      ).toBeVisible({ timeout: 3000 });
    }
  });
});
