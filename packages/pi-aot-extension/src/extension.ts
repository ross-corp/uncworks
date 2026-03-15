import { createInterface } from "node:readline";
import { trace, SpanStatusCode, type Tracer } from "@opentelemetry/api";
import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { create } from "@bufbuild/protobuf";
import {
  AgentNotificationService,
  NotifyEventRequestSchema,
  EventType,
} from "../../../gen/ts/aot/agent/v1/agent_pb";

/** Configuration for the AOT extension. */
export interface AOTExtensionConfig {
  agentRunId: string;
  controlPlaneAddress: string;
  sidecarAddress?: string;
  enableTracing: boolean;
  /** Override the stdin stream (for testing). Defaults to process.stdin. */
  stdin?: NodeJS.ReadableStream;
  /** Disable the sidecar notification client (for testing). */
  disableNotifications?: boolean;
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
  private stdinBuffer: string[] = [];
  private notifClient: Client<typeof AgentNotificationService> | null = null;

  constructor(private config: AOTExtensionConfig) {
    this.tracer = trace.getTracer("aot-extension", "0.1.0");
    this.initStdinReader();
    if (!config.disableNotifications) {
      this.initNotifClient();
    }
  }

  private initStdinReader(): void {
    const input = this.config.stdin ?? process.stdin;
    if (!input.readable) return;

    const rl = createInterface({ input });
    rl.on("line", (line: string) => {
      if (this.inputResolve) {
        const resolve = this.inputResolve;
        this.inputResolve = null;
        this.paused = false;
        this.waitingForInput = false;
        this.notifyStarted();
        resolve(line);
      } else {
        this.stdinBuffer.push(line);
      }
    });
  }

  private initNotifClient(): void {
    const address = this.config.sidecarAddress || "http://localhost:50052";
    const transport = createConnectTransport({
      baseUrl: address,
      httpVersion: "2",
    });
    this.notifClient = createClient(AgentNotificationService, transport);
  }

  private async notifyWaitingForInput(question: string): Promise<void> {
    if (!this.notifClient) return;
    try {
      await this.notifClient.notifyEvent(
        create(NotifyEventRequestSchema, {
          agentRunId: this.config.agentRunId,
          eventType: EventType.WAITING_FOR_INPUT,
          payload: question,
        })
      );
    } catch (err) {
      console.warn("Failed to send WAITING_FOR_INPUT notification:", err);
    }
  }

  private async notifyStarted(): Promise<void> {
    if (!this.notifClient) return;
    try {
      await this.notifClient.notifyEvent(
        create(NotifyEventRequestSchema, {
          agentRunId: this.config.agentRunId,
          eventType: EventType.STARTED,
        })
      );
    } catch (err) {
      console.warn("Failed to send STARTED notification:", err);
    }
  }

  private async notifyToolCall(
    name: string,
    params: Record<string, unknown>
  ): Promise<void> {
    if (!this.notifClient) return;
    try {
      await this.notifClient.notifyEvent(
        create(NotifyEventRequestSchema, {
          agentRunId: this.config.agentRunId,
          eventType: EventType.TOOL_CALL,
          payload: JSON.stringify({ name, params }),
        })
      );
    } catch (err) {
      console.warn("Failed to send TOOL_CALL notification:", err);
    }
  }

  private async notifyLog(label: string, detail: string): Promise<void> {
    if (!this.notifClient) return;
    try {
      await this.notifClient.notifyEvent(
        create(NotifyEventRequestSchema, {
          agentRunId: this.config.agentRunId,
          eventType: EventType.LOG,
          payload: JSON.stringify({ label, detail }),
        })
      );
    } catch (err) {
      console.warn("Failed to send LOG notification:", err);
    }
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

    // Notify sidecar of the tool call so it records a trace span
    this.notifyToolCall(name, params);

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
        // Notify sidecar with tool result as a log event
        this.notifyLog(
          `tool_result:${name}`,
          result.success ? "ok" : result.error ?? "error"
        );
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        span.setStatus({ code: SpanStatusCode.ERROR, message });
        this.notifyLog(`tool_error:${name}`, message);
        return { success: false, output: "", error: message };
      } finally {
        span.end();
      }
    });
  }

  /** Pause the execution loop and wait for human input. */
  async waitForHumanInput(question: string): Promise<string> {
    // Check buffer first — if input arrived before we started waiting, resolve immediately
    if (this.stdinBuffer.length > 0) {
      const buffered = this.stdinBuffer.shift()!;
      return buffered;
    }

    this.paused = true;
    this.waitingForInput = true;

    // Notify sidecar we're waiting (fire-and-forget, don't block on it)
    this.notifyWaitingForInput(question);

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
      this.notifyStarted();
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
