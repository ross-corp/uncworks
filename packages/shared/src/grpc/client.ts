import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { create } from "@bufbuild/protobuf";
import {
  AOTService,
  AgentRunSpecSchema,
  OrchestrationSchema,
  OrchestrationTaskSchema,
  PipelineConfigSchema,
  StageConfigSchema,
  RepositorySchema,
  type AgentRun as PbAgentRun,
  type AgentRunEvent as PbAgentRunEvent,
  Backend as PbBackend,
  AgentRunPhase as PbAgentRunPhase,
  AgentRunEventType as PbAgentRunEventType,
  OrchestrationMode as PbOrchestrationMode,
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
  /** Optional API key for authenticating with the AOT API server. */
  apiKey?: string;
}

/** ConnectRPC client for the AOT API Service. */
export class AOTClient {
  private client: Client<typeof AOTService>;
  /** The API key, exposed so REST calls (file explorer, shell, etc.) can use it. */
  readonly apiKey?: string;

  constructor(options: AOTClientOptions) {
    this.apiKey = options.apiKey;
    const transport =
      options.transport ??
      createConnectTransport({
        baseUrl: options.baseUrl,
        interceptors: options.apiKey
          ? [
              (next) => async (req) => {
                req.header.set("Authorization", `Bearer ${options.apiKey}`);
                return next(req);
              },
            ]
          : [],
      });
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
      workspaceName: spec.workspaceName ?? "",
      devboxConfig: spec.devboxConfig ?? "",
      ttlSeconds: spec.ttlSeconds ?? 0,
      envVars: spec.envVars ?? {},
      modelTier: spec.modelTier ?? "",
      specContent: spec.specContent ?? "",
      specSource: spec.specSource ?? "",
      orchestrationMode: spec.orchestrationMode ? orchModeToProto(spec.orchestrationMode) : undefined,
      orchestration: spec.orchestration
        ? create(OrchestrationSchema, {
            tasks: (spec.orchestration.tasks ?? []).map((t) =>
              create(OrchestrationTaskSchema, {
                name: t.name,
                prompt: t.prompt,
                repoUrls: t.repoUrls ?? [],
              })
            ),
          })
        : undefined,
      pipelineConfig: spec.pipelineConfig
        ? create(PipelineConfigSchema, {
            plan: spec.pipelineConfig.plan
              ? create(StageConfigSchema, {
                  model: spec.pipelineConfig.plan.model ?? "",
                  timeoutSeconds: spec.pipelineConfig.plan.timeoutSeconds ?? 0,
                  maxRetries: spec.pipelineConfig.plan.maxRetries ?? 0,
                  onFailure: spec.pipelineConfig.plan.onFailure ?? "",
                })
              : undefined,
            execute: spec.pipelineConfig.execute
              ? create(StageConfigSchema, {
                  model: spec.pipelineConfig.execute.model ?? "",
                  timeoutSeconds: spec.pipelineConfig.execute.timeoutSeconds ?? 0,
                  maxRetries: spec.pipelineConfig.execute.maxRetries ?? 0,
                  onFailure: spec.pipelineConfig.execute.onFailure ?? "",
                })
              : undefined,
            verify: spec.pipelineConfig.verify
              ? create(StageConfigSchema, {
                  model: spec.pipelineConfig.verify.model ?? "",
                  timeoutSeconds: spec.pipelineConfig.verify.timeoutSeconds ?? 0,
                  maxRetries: spec.pipelineConfig.verify.maxRetries ?? 0,
                  onFailure: spec.pipelineConfig.verify.onFailure ?? "",
                })
              : undefined,
          })
        : undefined,
      parentRunId: spec.parentRunId ?? "",
      project: spec.project ?? "",
      feature: spec.feature ?? "",
      tags: spec.tags ?? [],
      projectRef: spec.projectRef ?? "",
      specRef: spec.specRef ?? "",
      maxBudget: spec.maxBudget ?? 0,
      autoPush: spec.autoPush ?? false,
      autoPr: spec.autoPR ?? false,
      prBaseBranch: spec.prBaseBranch ?? "",
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
  [PbBackend.POD]: "pod",
};

const backendToProtoMap: Record<Backend, PbBackend> = {
  pod: PbBackend.POD,
};

function backendToProto(b: Backend): PbBackend {
  return backendToProtoMap[b] ?? PbBackend.UNSPECIFIED;
}

function backendFromProto(b: PbBackend): Backend {
  return backendFromProtoMap[b] ?? "pod";
}

const orchModeToProtoMap: Record<string, PbOrchestrationMode> = {
  single: PbOrchestrationMode.SINGLE,
  auto: PbOrchestrationMode.AUTO,
  manual: PbOrchestrationMode.MANUAL,
  "spec-driven": PbOrchestrationMode.SPEC_DRIVEN,
};

function orchModeToProto(m: string): PbOrchestrationMode {
  return orchModeToProtoMap[m] ?? PbOrchestrationMode.UNSPECIFIED;
}

const orchModeFromProtoMap: Record<number, AgentRunSpec["orchestrationMode"]> = {
  [PbOrchestrationMode.SINGLE]: "single",
  [PbOrchestrationMode.AUTO]: "auto",
  [PbOrchestrationMode.MANUAL]: "manual",
  [PbOrchestrationMode.SPEC_DRIVEN]: "spec-driven",
};

function orchModeFromProto(m: PbOrchestrationMode): AgentRunSpec["orchestrationMode"] {
  return orchModeFromProtoMap[m] ?? undefined;
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
      modelTier: pb.spec?.modelTier || undefined,
      specContent: pb.spec?.specContent || undefined,
      specSource: pb.spec?.specSource || undefined,
      parentRunId: pb.spec?.parentRunId || undefined,
      orchestrationMode: pb.spec?.orchestrationMode
        ? orchModeFromProto(pb.spec.orchestrationMode)
        : undefined,
      orchestration: pb.spec?.orchestration
        ? {
            tasks: (pb.spec.orchestration.tasks ?? []).map((t) => ({
              name: t.name,
              prompt: t.prompt,
              repoUrls: t.repoUrls.length > 0 ? t.repoUrls : undefined,
            })),
          }
        : undefined,
      specRunId: pb.spec?.specRunId || undefined,
      displayName: pb.spec?.displayName || undefined,
      pipelineConfig: pb.spec?.pipelineConfig
        ? {
            plan: pb.spec.pipelineConfig.plan
              ? {
                  model: pb.spec.pipelineConfig.plan.model || undefined,
                  timeoutSeconds: pb.spec.pipelineConfig.plan.timeoutSeconds || undefined,
                  maxRetries: pb.spec.pipelineConfig.plan.maxRetries || undefined,
                  onFailure: pb.spec.pipelineConfig.plan.onFailure || undefined,
                }
              : undefined,
            execute: pb.spec.pipelineConfig.execute
              ? {
                  model: pb.spec.pipelineConfig.execute.model || undefined,
                  timeoutSeconds: pb.spec.pipelineConfig.execute.timeoutSeconds || undefined,
                  maxRetries: pb.spec.pipelineConfig.execute.maxRetries || undefined,
                  onFailure: pb.spec.pipelineConfig.execute.onFailure || undefined,
                }
              : undefined,
            verify: pb.spec.pipelineConfig.verify
              ? {
                  model: pb.spec.pipelineConfig.verify.model || undefined,
                  timeoutSeconds: pb.spec.pipelineConfig.verify.timeoutSeconds || undefined,
                  maxRetries: pb.spec.pipelineConfig.verify.maxRetries || undefined,
                  onFailure: pb.spec.pipelineConfig.verify.onFailure || undefined,
                }
              : undefined,
          }
        : undefined,
      project: pb.spec?.project || undefined,
      feature: pb.spec?.feature || undefined,
      tags: pb.spec?.tags.length ? pb.spec.tags : undefined,
      projectRef: pb.spec?.projectRef || undefined,
      specRef: pb.spec?.specRef || undefined,
      maxBudget: pb.spec?.maxBudget || undefined,
      autoPush: pb.spec?.autoPush || undefined,
      autoPR: pb.spec?.autoPr || undefined,
      prBaseBranch: pb.spec?.prBaseBranch || undefined,
    },
    status: {
      phase: phaseFromProto(pb.status?.phase ?? PbAgentRunPhase.UNSPECIFIED),
      message: pb.status?.message,
      podName: pb.status?.podName,
      traceID: pb.status?.traceId,
      startedAt: pb.status?.startedAt ? timestampToISO(pb.status.startedAt) : undefined,
      completedAt: pb.status?.completedAt ? timestampToISO(pb.status.completedAt) : undefined,
      logOutput: pb.status?.logOutput || undefined,
      retainUntil: pb.status?.retainUntil ? timestampToISO(pb.status.retainUntil) : undefined,
      deploymentName: pb.status?.deploymentName || undefined,
      stage: pb.status?.stage || undefined,
      debugActive: pb.status?.debugActive || false,
      retryCount: pb.status?.retryCount || undefined,
      verificationResult: pb.status?.verificationResult || undefined,
      prUrl: pb.status?.prUrl || undefined,
    },
    createdAt: timestampToISO(pb.createdAt),
    updatedAt: timestampToISO(pb.updatedAt),
    children: pb.children.length ? pb.children : undefined,
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
