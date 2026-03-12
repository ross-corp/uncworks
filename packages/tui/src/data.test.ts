import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { DataBinding } from "./data.js";
import { createAppState } from "./state.js";
import type { AgentRun, AgentRunEvent } from "../../shared/src/types/agent-run.js";

const mockRun: AgentRun = {
  id: "ar-1",
  name: "fix-auth",
  spec: {
    backend: "Pod",
    repoURL: "https://github.com/test/repo",
    prompt: "Fix authentication bug",
  },
  status: { phase: "Running" },
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

const mockRun2: AgentRun = {
  id: "ar-2",
  name: "add-tests",
  spec: {
    backend: "Pod",
    repoURL: "https://github.com/test/repo",
    prompt: "Add unit tests",
  },
  status: { phase: "WaitingForInput" },
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

function createMockClient(overrides: Record<string, unknown> = {}) {
  return {
    listAgentRuns: async () => [mockRun, mockRun2],
    watchAgentRun: (
      _id: string,
      _onEvent: (e: AgentRunEvent) => void,
      _onError?: (e: Error) => void
    ) => new AbortController(),
    sendHumanInput: async (_id: string, _input: string) => true,
    ...overrides,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } as any;
}

describe("DataBinding", () => {
  it("fetches runs and populates state", async () => {
    const state = createAppState();
    const client = createMockClient();
    const data = new DataBinding(client, state);

    await data.fetchRuns();

    const runs = state.runs();
    assert.equal(runs.length, 2);
    assert.equal(runs[0].id, "ar-1");
    assert.equal(runs[0].name, "fix-auth");
    assert.equal(runs[0].phase, "Running");
    assert.equal(runs[1].phase, "WaitingForInput");
  });

  it("sets error on fetch failure", async () => {
    const state = createAppState();
    const client = createMockClient({
      listAgentRuns: async () => {
        throw new Error("connection refused");
      },
    });
    const data = new DataBinding(client, state);

    await data.fetchRuns();

    assert.ok(state.error()?.includes("connection refused"));
    assert.equal(state.runs().length, 0);
  });

  it("watches a run and updates phase on event", async () => {
    const state = createAppState();
    let eventCallback: ((e: AgentRunEvent) => void) | null = null;

    const client = createMockClient({
      watchAgentRun: (
        _id: string,
        onEvent: (e: AgentRunEvent) => void
      ) => {
        eventCallback = onEvent;
        return new AbortController();
      },
    });

    const data = new DataBinding(client, state);
    await data.fetchRuns();
    data.watchRun("ar-1");

    assert.ok(eventCallback, "event callback should be set");

    // Simulate a phase change event
    eventCallback!({
      agentRunId: "ar-1",
      type: "phase_changed",
      payload: "Succeeded",
      timestamp: "2026-01-01T00:01:00Z",
    });

    const runs = state.runs();
    assert.equal(runs[0].phase, "Succeeded");
    assert.equal(runs[1].phase, "WaitingForInput"); // unchanged

    data.stop();
  });

  it("sends human input successfully", async () => {
    const state = createAppState();
    let sentId = "";
    let sentInput = "";

    const client = createMockClient({
      sendHumanInput: async (id: string, input: string) => {
        sentId = id;
        sentInput = input;
        return true;
      },
    });

    const data = new DataBinding(client, state);
    const accepted = await data.sendInput("ar-2", "approve");

    assert.equal(accepted, true);
    assert.equal(sentId, "ar-2");
    assert.equal(sentInput, "approve");
    assert.equal(state.error(), null);
  });

  it("sets error when send input fails", async () => {
    const state = createAppState();
    const client = createMockClient({
      sendHumanInput: async () => {
        throw new Error("timeout");
      },
    });

    const data = new DataBinding(client, state);
    const accepted = await data.sendInput("ar-2", "approve");

    assert.equal(accepted, false);
    assert.ok(state.error()?.includes("timeout"));
  });

  it("sets error when input not accepted", async () => {
    const state = createAppState();
    const client = createMockClient({
      sendHumanInput: async () => false,
    });

    const data = new DataBinding(client, state);
    const accepted = await data.sendInput("ar-2", "approve");

    assert.equal(accepted, false);
    assert.ok(state.error()?.includes("not accepted"));
  });

  it("stops watch on stopWatch", () => {
    const state = createAppState();
    let aborted = false;
    const ctrl = new AbortController();
    ctrl.signal.addEventListener("abort", () => {
      aborted = true;
    });

    const client = createMockClient({
      watchAgentRun: () => ctrl,
    });

    const data = new DataBinding(client, state);
    data.watchRun("ar-1");
    data.stopWatch();

    assert.ok(aborted, "watch should be aborted");
  });
});
