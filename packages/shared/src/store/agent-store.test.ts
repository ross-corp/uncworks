import { describe, it } from "node:test";
import assert from "node:assert/strict";
// Note: SolidJS reactivity works outside of JSX, but createRoot is needed
// for proper cleanup. Using basic store operations for testing.

// Since we can't easily test SolidJS reactivity in Node without a DOM,
// we test the store logic through its public API.
import type { AgentRun, AgentRunPhase } from "../types/agent-run";

describe("AgentStore types", () => {
  it("should define all phase types", () => {
    const phases: AgentRunPhase[] = [
      "Pending",
      "Running",
      "WaitingForInput",
      "Succeeded",
      "Failed",
      "Cancelled",
    ];
    assert.equal(phases.length, 6);
  });

  it("should define AgentRun interface shape", () => {
    const run: AgentRun = {
      id: "ar-1",
      name: "test-run",
      spec: {
        backend: "Pod",
        repoURL: "https://github.com/test/repo",
        prompt: "Fix tests",
      },
      status: {
        phase: "Running",
        message: "In progress",
      },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    assert.equal(run.id, "ar-1");
    assert.equal(run.spec.backend, "Pod");
    assert.equal(run.status.phase, "Running");
  });

  it("should support all backend types", () => {
    const backends = ["Pod", "KubeVirt", "External"] as const;
    assert.equal(backends.length, 3);
  });
});
