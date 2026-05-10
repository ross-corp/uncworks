import { createContext, useContext } from "react";
import { AOTClient } from "../../../packages/shared/src/grpc/client";
import type {
  AgentRun as SharedAgentRun,
  AgentRunEvent as SharedEvent,
} from "../../../packages/shared/src/types/agent-run";
import type { AgentRun, AgentRunEvent, AgentRunPhase, Backend, ModelTier, Repository } from "../types/agent-run";

const API_BASE_URL = import.meta.env.VITE_API_URL ?? "";
const API_KEY = import.meta.env.VITE_API_KEY ?? "";

const defaultClient = new AOTClient({
  baseUrl: API_BASE_URL,
  apiKey: API_KEY || undefined,
});

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
  pod: "pod",
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
      orchestrationMode: r.spec.orchestrationMode as AgentRun["spec"]["orchestrationMode"],
      displayName: r.spec.displayName,
      project: r.spec.project,
      projectRef: r.spec.projectRef,
      feature: r.spec.feature,
      tags: r.spec.tags,
      maxBudget: r.spec.maxBudget,
      autoPush: r.spec.autoPush,
      autoPR: r.spec.autoPR,
      prBaseBranch: r.spec.prBaseBranch,
    },
    status: {
      phase: phaseMap[r.status.phase] ?? "pending",
      message: r.status.message ?? "",
      podName: r.status.podName ?? "",
      traceID: r.status.traceID ?? "",
      startedAt: r.status.startedAt ?? "",
      completedAt: r.status.completedAt ?? "",
      logOutput: r.status.logOutput,
      deploymentName: r.status.deploymentName,
      stage: r.status.stage,
      retryCount: r.status.retryCount,
      verificationResult: r.status.verificationResult,
      debugActive: r.status.debugActive ?? false,
      prUrl: r.status.prUrl,
      archived: r.status.archived,
      totalCost: r.status.totalCost,
      totalAdditions: r.status.totalAdditions,
      totalDeletions: r.status.totalDeletions,
      ciFixAttempts: r.status.ciFixAttempts,
      lastCIStatus: r.status.lastCIStatus,
      parentPRUrl: r.status.parentPRUrl,
    },
    createdAt: r.createdAt,
    updatedAt: r.updatedAt ?? r.createdAt,
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
