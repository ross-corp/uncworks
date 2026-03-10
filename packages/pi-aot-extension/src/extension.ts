import { trace, SpanStatusCode, type Tracer } from "@opentelemetry/api";

/** Configuration for the AOT extension. */
export interface AOTExtensionConfig {
  agentRunId: string;
  controlPlaneAddress: string;
  enableTracing: boolean;
}

/** Definition for a tool available to the agent. */
export interface ToolDefinition {
  name: string;
  description: string;
  parameters: Record<string, unknown>;
  execute: (params: Record<string, unknown>) => Promise<ToolResult>;
}

/** Result of a tool execution. */
export interface ToolResult {
  success: boolean;
  output: string;
  error?: string;
}

/**
 * AOTExtension is the agent harness extension that bridges the agent
 * with the AOT control plane. It manages tool registration, OTel tracing,
 * and the execution loop.
 */
export class AOTExtension {
  private tools: Map<string, ToolDefinition> = new Map();
  private tracer: Tracer;
  private paused = false;
  private waitingForInput = false;
  private inputResolve: ((input: string) => void) | null = null;

  constructor(private config: AOTExtensionConfig) {
    this.tracer = trace.getTracer("aot-extension", "0.1.0");
  }

  /** Register a tool with the extension. */
  registerTool(tool: ToolDefinition): void {
    this.tools.set(tool.name, tool);
  }

  /** Get all registered tools. */
  getTools(): ToolDefinition[] {
    return Array.from(this.tools.values());
  }

  /** Execute a tool by name with tracing. */
  async executeTool(
    name: string,
    params: Record<string, unknown>
  ): Promise<ToolResult> {
    const tool = this.tools.get(name);
    if (!tool) {
      return { success: false, output: "", error: `Unknown tool: ${name}` };
    }

    return this.tracer.startActiveSpan(`tool.${name}`, async (span) => {
      span.setAttribute("tool.name", name);
      span.setAttribute("agent_run_id", this.config.agentRunId);

      try {
        const result = await tool.execute(params);
        if (result.success) {
          span.setStatus({ code: SpanStatusCode.OK });
        } else {
          span.setStatus({
            code: SpanStatusCode.ERROR,
            message: result.error,
          });
        }
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        span.setStatus({ code: SpanStatusCode.ERROR, message });
        return { success: false, output: "", error: message };
      } finally {
        span.end();
      }
    });
  }

  /** Pause the execution loop and wait for human input. */
  async waitForHumanInput(question: string): Promise<string> {
    this.paused = true;
    this.waitingForInput = true;

    return new Promise<string>((resolve) => {
      this.inputResolve = resolve;
    });
  }

  /** Provide human input to resume the execution loop. */
  provideHumanInput(input: string): void {
    if (this.inputResolve) {
      this.inputResolve(input);
      this.inputResolve = null;
      this.paused = false;
      this.waitingForInput = false;
    }
  }

  /** Check if the extension is waiting for human input. */
  isWaitingForInput(): boolean {
    return this.waitingForInput;
  }

  /** Check if the extension is paused. */
  isPaused(): boolean {
    return this.paused;
  }

  /** Get the agent run ID. */
  getAgentRunId(): string {
    return this.config.agentRunId;
  }
}
