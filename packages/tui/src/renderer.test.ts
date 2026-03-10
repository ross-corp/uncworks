import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { renderToString } from "./renderer";
import { dashboardView, agentRunListView, headerView } from "./views";
import type { AgentRunView } from "./views";

const mockRuns: AgentRunView[] = [
  {
    id: "ar-1",
    name: "fix-auth",
    phase: "Running",
    backend: "Pod",
    prompt: "Fix authentication bug",
  },
  {
    id: "ar-2",
    name: "add-tests",
    phase: "Succeeded",
    backend: "Pod",
    prompt: "Add unit tests",
  },
];

describe("TUI Renderer", () => {
  it("should render header", () => {
    const output = renderToString(headerView());
    assert.ok(output.includes("AOT Dashboard"));
  });

  it("should render agent run list", () => {
    const output = renderToString(agentRunListView(mockRuns, 0));
    assert.ok(output.includes("fix-auth"));
    assert.ok(output.includes("add-tests"));
    assert.ok(output.includes("Running"));
    assert.ok(output.includes("Succeeded"));
  });

  it("should highlight selected run", () => {
    const output = renderToString(agentRunListView(mockRuns, 0));
    assert.ok(output.includes("▸"));
  });

  it("should render empty state", () => {
    const output = renderToString(agentRunListView([], 0));
    assert.ok(output.includes("No agent runs"));
  });

  it("should render full dashboard", () => {
    const output = renderToString(dashboardView(mockRuns, 0, mockRuns[0]));
    assert.ok(output.includes("AOT Dashboard"));
    assert.ok(output.includes("fix-auth"));
    assert.ok(output.includes("Detail"));
    assert.ok(output.includes("Pod"));
  });

  it("should render dashboard with no selection", () => {
    const output = renderToString(dashboardView(mockRuns, -1, null));
    assert.ok(output.includes("Select an agent run"));
  });
});
