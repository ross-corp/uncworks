import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { AOTExtension } from "./extension";
import { AskHumanTool } from "./tools/ask-human";

describe("HITL Signaling", () => {
  it("should pause execution and wait for human input", async () => {
    const ext = new AOTExtension({
      agentRunId: "hitl-test-1",
      controlPlaneAddress: "localhost:50051",
      enableTracing: false,
    });

    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);

    // Start waiting for input
    const inputPromise = ext.waitForHumanInput("What should I do?");

    assert.equal(ext.isPaused(), true);
    assert.equal(ext.isWaitingForInput(), true);

    // Simulate human providing input
    ext.provideHumanInput("Continue with option A");

    const result = await inputPromise;
    assert.equal(result, "Continue with option A");
    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);
  });

  it("should integrate ask_human tool with extension HITL flow", async () => {
    const ext = new AOTExtension({
      agentRunId: "hitl-test-2",
      controlPlaneAddress: "localhost:50051",
      enableTracing: false,
    });

    // Register ask_human that uses the extension's HITL mechanism
    const askHuman = AskHumanTool(async (question) => {
      // In real usage, this would send the question via gRPC to the control plane
      // and wait for the response. Here we simulate it.
      const inputPromise = ext.waitForHumanInput(question);

      // Simulate control plane forwarding human input after a brief delay
      setTimeout(() => {
        ext.provideHumanInput("Approved");
      }, 10);

      return inputPromise;
    });

    ext.registerTool(askHuman);

    const result = await ext.executeTool("ask_human", {
      question: "Should I merge this PR?",
    });

    assert.equal(result.success, true);
    assert.equal(result.output, "Approved");
    assert.equal(ext.isPaused(), false);
  });
});
