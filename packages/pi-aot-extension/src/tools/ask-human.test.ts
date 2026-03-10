import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { AskHumanTool } from "./ask-human";

describe("AskHumanTool – additional coverage", () => {
  it("should return error when onAsk throws an Error", async () => {
    const tool = AskHumanTool(async () => {
      throw new Error("upstream service unavailable");
    });
    const result = await tool.execute({ question: "Are you there?" });
    assert.equal(result.success, false);
    assert.equal(result.error, "upstream service unavailable");
    assert.equal(result.output, "");
  });

  it("should return error when onAsk throws a string", async () => {
    const tool = AskHumanTool(async () => {
      // eslint-disable-next-line no-throw-literal
      throw "plain string error";
    });
    const result = await tool.execute({ question: "Hello?" });
    assert.equal(result.success, false);
    assert.equal(result.error, "plain string error");
    assert.equal(result.output, "");
  });

  it("should fail when question is an empty string", async () => {
    const tool = AskHumanTool(async () => "should not be called");
    const result = await tool.execute({ question: "" });
    assert.equal(result.success, false);
    assert.equal(result.error, "question is required");
  });
});
