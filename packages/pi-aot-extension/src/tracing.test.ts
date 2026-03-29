import { describe, it } from "vitest";
import assert from "node:assert/strict";
import { SpanStatusCode } from "@opentelemetry/api";
import {
  InMemorySpanExporter,
  SimpleSpanProcessor,
  NodeTracerProvider,
} from "@opentelemetry/sdk-trace-node";
import { Resource } from "@opentelemetry/resources";
import { AOTExtension } from "./extension";

describe("OTel Tracing", () => {
  it("should emit spans for tool calls with correct status", async () => {
    const exporter = new InMemorySpanExporter();
    const provider = new NodeTracerProvider({
      resource: new Resource({ "service.name": "test" }),
    });
    provider.addSpanProcessor(new SimpleSpanProcessor(exporter));
    provider.register();

    const ext = new AOTExtension({
      agentRunId: "trace-test-1",
      controlPlaneAddress: "localhost:50051",
      enableTracing: true,
    });

    // Register a successful tool and a failing tool
    ext.registerTool({
      name: "traced_tool",
      description: "A traced tool",
      parameters: {},
      execute: async () => ({ success: true, output: "traced" }),
    });

    ext.registerTool({
      name: "failing_traced",
      description: "Fails",
      parameters: {},
      execute: async () => {
        throw new Error("boom");
      },
    });

    // Execute both
    await ext.executeTool("traced_tool", { key: "value" });
    await ext.executeTool("failing_traced", {});

    await provider.forceFlush();

    const spans = exporter.getFinishedSpans();
    assert.ok(spans.length >= 2, `Should have at least 2 spans, got ${spans.length}`);

    // Verify successful tool span
    const toolSpan = spans.find((s) => s.name === "tool.traced_tool");
    assert.ok(toolSpan, "Should have a span for the traced_tool call");
    assert.equal(toolSpan.status.code, SpanStatusCode.OK);
    assert.equal(toolSpan.attributes["tool.name"], "traced_tool");
    assert.equal(toolSpan.attributes["agent_run_id"], "trace-test-1");

    // Verify error tool span
    const errorSpan = spans.find((s) => s.name === "tool.failing_traced");
    assert.ok(errorSpan, "Should have error span");
    assert.equal(errorSpan.status.code, SpanStatusCode.ERROR);
    assert.ok(errorSpan.status.message?.includes("boom"));

    await provider.shutdown();
  });
});
