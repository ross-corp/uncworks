import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { PassThrough } from "node:stream";
import { AOTExtension } from "./extension";
import { AskHumanTool } from "./tools/ask-human";

function createExtension(stdin: PassThrough): AOTExtension {
  return new AOTExtension({
    agentRunId: "hitl-test",
    controlPlaneAddress: "localhost:50051",
    enableTracing: false,
    stdin,
    disableNotifications: true,
  });
}

describe("HITL Signaling", () => {
  it("should pause execution and wait for human input via stdin", async () => {
    const stdin = new PassThrough();
    const ext = createExtension(stdin);

    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);

    const inputPromise = ext.waitForHumanInput("What should I do?");

    assert.equal(ext.isPaused(), true);
    assert.equal(ext.isWaitingForInput(), true);

    // Write to the injectable stdin stream
    stdin.write("Continue with option A\n");

    const result = await inputPromise;
    assert.equal(result, "Continue with option A");
    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);

    stdin.end();
  });

  it("should resolve immediately with buffered stdin input", async () => {
    const stdin = new PassThrough();
    const ext = createExtension(stdin);

    // Push stdin line BEFORE calling waitForHumanInput
    stdin.write("pre-buffered answer\n");

    // Give readline time to process the line
    await new Promise((r) => setTimeout(r, 50));

    // Should resolve immediately from buffer
    const result = await ext.waitForHumanInput("Any question?");
    assert.equal(result, "pre-buffered answer");
    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);

    stdin.end();
  });

  it("should work with provideHumanInput (programmatic path)", async () => {
    const stdin = new PassThrough();
    const ext = createExtension(stdin);

    const inputPromise = ext.waitForHumanInput("What should I do?");

    assert.equal(ext.isPaused(), true);
    assert.equal(ext.isWaitingForInput(), true);

    // Use provideHumanInput instead of stdin
    ext.provideHumanInput("Continue with option A");

    const result = await inputPromise;
    assert.equal(result, "Continue with option A");
    assert.equal(ext.isPaused(), false);
    assert.equal(ext.isWaitingForInput(), false);

    stdin.end();
  });

  it("should integrate ask_human tool with extension HITL flow", async () => {
    const stdin = new PassThrough();
    const ext = createExtension(stdin);

    const askHuman = AskHumanTool(async (question) => {
      const inputPromise = ext.waitForHumanInput(question);

      // Simulate stdin input after a brief delay
      setTimeout(() => {
        stdin.write("Approved\n");
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

    stdin.end();
  });
});
