import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import type { AgentRun, AgentRunEvent, AgentRunSpec } from "../types/agent-run";

export interface AOTClientOptions {
  address: string;
  protoPath: string;
}

/** gRPC client for the AOT API Service. */
export class AOTClient {
  private client: any;

  constructor(private options: AOTClientOptions) {}

  /** Connect to the gRPC server. */
  async connect(): Promise<void> {
    const packageDef = await protoLoader.load(this.options.protoPath, {
      keepCase: false,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true,
    });

    const proto = grpc.loadPackageDefinition(packageDef) as any;
    this.client = new proto.aot.api.v1.AOTService(
      this.options.address,
      grpc.credentials.createInsecure()
    );
  }

  /** Create a new AgentRun. */
  createAgentRun(spec: AgentRunSpec): Promise<AgentRun> {
    return new Promise((resolve, reject) => {
      this.client.CreateAgentRun({ spec }, (err: any, res: any) => {
        if (err) return reject(err);
        resolve(res.agentRun);
      });
    });
  }

  /** Get an AgentRun by ID. */
  getAgentRun(id: string): Promise<AgentRun> {
    return new Promise((resolve, reject) => {
      this.client.GetAgentRun({ id }, (err: any, res: any) => {
        if (err) return reject(err);
        resolve(res);
      });
    });
  }

  /** List AgentRuns. */
  listAgentRuns(phaseFilter?: string, limit?: number): Promise<AgentRun[]> {
    return new Promise((resolve, reject) => {
      this.client.ListAgentRuns(
        { phaseFilter, limit },
        (err: any, res: any) => {
          if (err) return reject(err);
          resolve(res.agentRuns || []);
        }
      );
    });
  }

  /** Watch an AgentRun for real-time events. Returns an event stream. */
  watchAgentRun(
    id: string,
    onEvent: (event: AgentRunEvent) => void,
    onError?: (err: Error) => void
  ): () => void {
    const call = this.client.WatchAgentRun({ id });
    call.on("data", (event: AgentRunEvent) => onEvent(event));
    call.on("error", (err: Error) => onError?.(err));
    return () => call.cancel();
  }

  /** Cancel an AgentRun. */
  cancelAgentRun(id: string): Promise<AgentRun> {
    return new Promise((resolve, reject) => {
      this.client.CancelAgentRun({ id }, (err: any, res: any) => {
        if (err) return reject(err);
        resolve(res.agentRun);
      });
    });
  }

  /** Send human input to a waiting AgentRun. */
  sendHumanInput(agentRunId: string, input: string): Promise<boolean> {
    return new Promise((resolve, reject) => {
      this.client.SendHumanInput(
        { agentRunId, input },
        (err: any, res: any) => {
          if (err) return reject(err);
          resolve(res.accepted);
        }
      );
    });
  }
}
