import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { create } from "@bufbuild/protobuf";
import {
  AOTService,
  AgentRunSpecSchema,
  RepositorySchema,
  type AgentRun as PbAgentRun,
  type AgentRunEvent as PbAgentRunEvent,
  type AgentRunSpec as PbAgentRunSpec,
  Backend as PbBackend,
  AgentRunPhase as PbAgentRunPhase,
  AgentRunEventType as PbAgentRunEventType,
} from "../../../../gen/ts/aot/api/v1/api_pb.js";
import type {
  AgentRun,
  AgentRunEvent,
  AgentRunEventType,
  AgentRunPhase,
  AgentRunSpec,
  Backend,
} from "../types/agent-run";

export interface AOTClientOptions {
  /** Base URL of the AOT API server (e.g., "http://localhost:50051"). */
  baseUrl: string;
  /** Optional custom transport (for testing or Node.js usage). */
  transport?: Transport;
}

/** ConnectRPC client for the AOT API Service. */
export class AOTClient {
  private client: Client<typeof AOTService>;

  constructor(options: AOTClientOptions) {
    const transport =
      options.transport ??
      createConnectTransport({ baseUrl: options.baseUrl });
    this.client = createClient(AOTService, transport);
  }

  /** Create a new AgentRun. */
  async createAgentRun(spec: AgentRunSpec): Promise<AgentRun> {
    const pbSpec = create(AgentRunSpecSchema, {
      backend: backendToProto(spec.backend),
      repos: spec.repos.map((r) =>
        create(RepositorySchema, {
          url: r.url,
          branch: r.branch ?? "",
          path: r.path ?? "",
        })
      ),
      prompt: spec.prompt,
      devboxConfig: spec.devboxConfig ?? "",
      ttlSeconds: spec.ttlSeconds ?? 0,
      envVars: spec.envVars ?? {},
    });
    const resp = await this.client.createAgentRun({ spec: pbSpec });
    return toAgentRun(resp.agentRun!);
  }

  /** Get an AgentRun by ID. */
  async getAgentRun(id: string): Promise<AgentRun> {
    const resp = await this.client.getAgentRun({ id });
    return toAgentRun(resp);
  }

  /** List AgentRuns. */
  async listAgentRuns(
    phaseFilter?: AgentRunPhase,
    limit?: number
  ): Promise<AgentRun[]> {
    const resp = await this.client.listAgentRuns({
      phaseFilter: phaseFilter ? phaseToProto(phaseFilter) : PbAgentRunPhase.UNSPECIFIED,
      limit: limit ?? 0,
    });
    return (resp.agentRuns ?? []).map(toAgentRun);
  }

  /** Watch an AgentRun for real-time events via Connect server-streaming. */
  watchAgentRun(
    id: string,
    onEvent: (event: AgentRunEvent) => void,
    onError?: (err: Error) => void
  ): AbortController {
    const abort = new AbortController();
    (async () => {
      try {
        for await (const event of this.client.watchAgentRun(
          { id },
          { signal: abort.signal }
        )) {
          onEvent(toAgentRunEvent(event));
        }
      } catch (err) {
        if (!abort.signal.aborted) {
          onError?.(err as Error);
        }
      }
    })();
    return abort;
  }

  /** Cancel an AgentRun. */
  async cancelAgentRun(id: string): Promise<AgentRun> {
    const resp = await this.client.cancelAgentRun({ id });
    return toAgentRun(resp.agentRun!);
  }

  /** Send human input to a waiting AgentRun. */
  async sendHumanInput(agentRunId: string, input: string): Promise<boolean> {
    const resp = await this.client.sendHumanInput({ agentRunId, input });
    return resp.accepted;
  }
}

// --- Proto <-> Domain mappings ---

const backendFromProtoMap: Record<number, Backend> = {
  [PbBackend.POD]: "Pod",
  [PbBackend.KUBEVIRT]: "KubeVirt",
  [PbBackend.EXTERNAL]: "External",
};

const backendToProtoMap: Record<Backend, PbBackend> = {
  Pod: PbBackend.POD,
  KubeVirt: PbBackend.KUBEVIRT,
  External: PbBackend.EXTERNAL,
};

function backendToProto(b: Backend): PbBackend {
  return backendToProtoMap[b] ?? PbBackend.UNSPECIFIED;
}

function backendFromProto(b: PbBackend): Backend {
  return backendFromProtoMap[b] ?? "Pod";
}

const phaseFromProtoMap: Record<number, AgentRunPhase> = {
  [PbAgentRunPhase.PENDING]: "Pending",
  [PbAgentRunPhase.RUNNING]: "Running",
  [PbAgentRunPhase.WAITING_FOR_INPUT]: "WaitingForInput",
  [PbAgentRunPhase.SUCCEEDED]: "Succeeded",
  [PbAgentRunPhase.FAILED]: "Failed",
  [PbAgentRunPhase.CANCELLED]: "Cancelled",
};

const phaseToProtoMap: Record<AgentRunPhase, PbAgentRunPhase> = {
  Pending: PbAgentRunPhase.PENDING,
  Running: PbAgentRunPhase.RUNNING,
  WaitingForInput: PbAgentRunPhase.WAITING_FOR_INPUT,
  Succeeded: PbAgentRunPhase.SUCCEEDED,
  Failed: PbAgentRunPhase.FAILED,
  Cancelled: PbAgentRunPhase.CANCELLED,
};

function phaseToProto(p: AgentRunPhase): PbAgentRunPhase {
  return phaseToProtoMap[p] ?? PbAgentRunPhase.UNSPECIFIED;
}

function phaseFromProto(p: PbAgentRunPhase): AgentRunPhase {
  return phaseFromProtoMap[p] ?? "Pending";
}

const eventTypeFromProtoMap: Record<number, AgentRunEventType> = {
  [PbAgentRunEventType.PHASE_CHANGED]: "phase_changed",
  [PbAgentRunEventType.LOG]: "log",
  [PbAgentRunEventType.TOOL_CALL]: "tool_call",
  [PbAgentRunEventType.WAITING_FOR_INPUT]: "waiting_for_input",
  [PbAgentRunEventType.COMPLETED]: "completed",
};

function timestampToISO(ts?: { seconds: bigint; nanos: number }): string {
  if (!ts) return new Date().toISOString();
  return new Date(Number(ts.seconds) * 1000 + ts.nanos / 1_000_000).toISOString();
}

function toAgentRun(pb: PbAgentRun): AgentRun {
  return {
    id: pb.id,
    name: pb.name,
    spec: {
      backend: backendFromProto(pb.spec?.backend ?? PbBackend.UNSPECIFIED),
      repos: (pb.spec?.repos ?? []).map((r) => ({
        url: r.url,
        branch: r.branch || undefined,
        path: r.path || undefined,
      })),
      prompt: pb.spec?.prompt ?? "",
      devboxConfig: pb.spec?.devboxConfig,
      ttlSeconds: pb.spec?.ttlSeconds,
      envVars: pb.spec?.envVars ?? {},
    },
    status: {
      phase: phaseFromProto(pb.status?.phase ?? PbAgentRunPhase.UNSPECIFIED),
      message: pb.status?.message,
      podName: pb.status?.podName,
      traceID: pb.status?.traceId,
      startedAt: pb.status?.startedAt ? timestampToISO(pb.status.startedAt) : undefined,
      completedAt: pb.status?.completedAt ? timestampToISO(pb.status.completedAt) : undefined,
    },
    createdAt: timestampToISO(pb.createdAt),
    updatedAt: timestampToISO(pb.updatedAt),
  };
}

function toAgentRunEvent(pb: PbAgentRunEvent): AgentRunEvent {
  return {
    agentRunId: pb.agentRunId,
    type: eventTypeFromProtoMap[pb.type] ?? "log",
    payload: pb.payload,
    timestamp: timestampToISO(pb.timestamp),
  };
}
