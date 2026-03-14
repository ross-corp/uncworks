import { createContext, useContext } from "react";
import { AOTClient } from "../../../packages/shared/src/grpc/client";
import type {
  AgentRun as SharedAgentRun,
  AgentRunEvent as SharedEvent,
} from "../../../packages/shared/src/types/agent-run";
import type { AgentRun, AgentRunEvent, AgentRunPhase, Backend, ModelTier, Repository } from "../types/agent-run";

const API_BASE_URL = import.meta.env.VITE_API_URL ?? "";

const defaultClient = new AOTClient({ baseUrl: API_BASE_URL });

export const ClientContext = createContext<AOTClient>(defaultClient);

export function useClient(): AOTClient {
  return useContext(ClientContext);
}

const phaseMap: Record<string, AgentRunPhase> = {
  Pending: "pending",
  Running: "running",
  WaitingForInput: "waiting_for_input",
  Succeeded: "succeeded",
  Failed: "failed",
  Cancelled: "cancelled",
};

const backendMap: Record<string, Backend> = {
  Pod: "pod",
  KubeVirt: "kubevirt",
  External: "external",
};

/** Map shared AgentRun → web UI AgentRun */
export function mapRun(r: SharedAgentRun): AgentRun {
  return {
    id: r.id,
    name: r.name,
    spec: {
      backend: backendMap[r.spec.backend] ?? "pod",
      repos: (r.spec.repos ?? []).map((repo): Repository => ({
        url: repo.url,
        branch: repo.branch ?? "main",
        path: repo.path,
      })),
      workspaceName: r.spec.workspaceName,
      prompt: r.spec.prompt,
      devboxConfig: r.spec.devboxConfig ?? "",
      ttlSeconds: r.spec.ttlSeconds ?? 3600,
      envVars: r.spec.envVars ?? {},
      modelTier: (r.spec.modelTier as ModelTier) ?? "default",
      specContent: r.spec.specContent,
      specSource: r.spec.specSource,
    },
    status: {
      phase: phaseMap[r.status.phase] ?? "pending",
      message: r.status.message ?? "",
      podName: r.status.podName ?? "",
      traceID: r.status.traceID ?? "",
      startedAt: r.status.startedAt ?? "",
      completedAt: r.status.completedAt ?? "",
    },
    createdAt: r.createdAt,
  };
}

/** Map shared AgentRunEvent → web UI AgentRunEvent */
export function mapEvent(e: SharedEvent): AgentRunEvent {
  return {
    agentRunId: e.agentRunId,
    type: e.type,
    payload: e.payload,
    timestamp: e.timestamp,
  };
}
