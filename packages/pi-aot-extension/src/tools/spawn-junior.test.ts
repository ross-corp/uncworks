import { describe, it } from "vitest";
import assert from "node:assert/strict";
import { SpawnJuniorTool } from "./spawn-junior";

describe("SpawnJuniorTool – additional coverage", () => {
  it("should return error when onSpawn throws an Error", async () => {
    const tool = SpawnJuniorTool(async () => {
      throw new Error("spawn limit exceeded");
    });
    const result = await tool.execute({ task: "do-stuff", context: "ctx" });
    assert.equal(result.success, false);
    assert.equal(result.error, "spawn limit exceeded");
    assert.equal(result.output, "");
  });

  it("should return error when onSpawn throws a non-Error value", async () => {
    const tool = SpawnJuniorTool(async () => {
      // eslint-disable-next-line no-throw-literal
      throw 42;
    });
    const result = await tool.execute({ task: "do-stuff" });
    assert.equal(result.success, false);
    assert.equal(result.error, "42");
  });

  it("should pass empty string as context when context is undefined", async () => {
    let capturedCtx: string | undefined;
    const tool = SpawnJuniorTool(async (_task, ctx) => {
      capturedCtx = ctx;
      return "run-123";
    });
    const result = await tool.execute({ task: "build" });
    assert.equal(result.success, true);
    assert.equal(result.output, "run-123");
    assert.equal(capturedCtx, "");
  });

  it("should pass empty string as context when context is explicitly empty", async () => {
    let capturedCtx: string | undefined;
    const tool = SpawnJuniorTool(async (_task, ctx) => {
      capturedCtx = ctx;
      return "run-456";
    });
    const result = await tool.execute({ task: "lint", context: "" });
    assert.equal(result.success, true);
    assert.equal(result.output, "run-456");
    assert.equal(capturedCtx, "");
  });
});
