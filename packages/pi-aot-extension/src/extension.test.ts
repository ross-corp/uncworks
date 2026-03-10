import { describe, it, before } from "node:test";
import assert from "node:assert/strict";
import { AOTExtension } from "./extension";
import { AskHumanTool } from "./tools/ask-human";
import { SpawnJuniorTool } from "./tools/spawn-junior";

describe("AOTExtension", () => {
  let ext: AOTExtension;

  before(() => {
    ext = new AOTExtension({
      agentRunId: "test-run-1",
      controlPlaneAddress: "localhost:50051",
      enableTracing: false,
    });
  });

  it("should register and retrieve tools", () => {
    ext.registerTool({
      name: "test_tool",
      description: "A test tool",
      parameters: {},
      execute: async () => ({ success: true, output: "ok" }),
    });

    const tools = ext.getTools();
    assert.ok(tools.some((t) => t.name === "test_tool"));
  });

  it("should execute a registered tool", async () => {
    ext.registerTool({
      name: "echo",
      description: "Echo input",
      parameters: {},
      execute: async (params) => ({
        success: true,
        output: String(params.message),
      }),
    });

    const result = await ext.executeTool("echo", { message: "hello" });
    assert.equal(result.success, true);
    assert.equal(result.output, "hello");
  });

  it("should return error for unknown tool", async () => {
    const result = await ext.executeTool("nonexistent", {});
    assert.equal(result.success, false);
    assert.ok(result.error?.includes("Unknown tool"));
  });

  it("should handle tool execution errors", async () => {
    ext.registerTool({
      name: "failing_tool",
      description: "Always fails",
      parameters: {},
      execute: async () => {
        throw new Error("intentional failure");
      },
    });

    const result = await ext.executeTool("failing_tool", {});
    assert.equal(result.success, false);
    assert.ok(result.error?.includes("intentional failure"));
  });

  it("should return agent run ID", () => {
    assert.equal(ext.getAgentRunId(), "test-run-1");
  });
});

describe("AskHumanTool", () => {
  it("should call onAsk with the question", async () => {
    const tool = AskHumanTool(async (q) => `Answer to: ${q}`);
    const result = await tool.execute({ question: "What color?" });
    assert.equal(result.success, true);
    assert.equal(result.output, "Answer to: What color?");
  });

  it("should fail without question", async () => {
    const tool = AskHumanTool(async () => "");
    const result = await tool.execute({});
    assert.equal(result.success, false);
  });
});

describe("SpawnJuniorTool", () => {
  it("should call onSpawn with task and context", async () => {
    const tool = SpawnJuniorTool(async (task, ctx) => `junior-${task}-${ctx}`);
    const result = await tool.execute({
      task: "fix-tests",
      context: "branch-main",
    });
    assert.equal(result.success, true);
    assert.equal(result.output, "junior-fix-tests-branch-main");
  });

  it("should fail without task", async () => {
    const tool = SpawnJuniorTool(async () => "");
    const result = await tool.execute({});
    assert.equal(result.success, false);
  });
});
