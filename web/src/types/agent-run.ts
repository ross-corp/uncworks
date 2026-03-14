export type AgentRunPhase =
  | "pending"
  | "running"
  | "waiting_for_input"
  | "succeeded"
  | "failed"
  | "cancelled";

export type Backend = "pod" | "kubevirt" | "external";

export type ModelTier = "default" | "default-cloud" | "premium";

export interface Repository {
  url: string;
  branch: string;
  path?: string;
}

export interface AgentRun {
  id: string;
  name: string;
  spec: AgentRunSpec;
  status: AgentRunStatus;
  createdAt: string;
}

export interface AgentRunSpec {
  backend: Backend;
  repos: Repository[];
  workspaceName?: string;
  prompt: string;
  devboxConfig: string;
  ttlSeconds: number;
  envVars: Record<string, string>;
  modelTier: ModelTier;
  specContent?: string;
  specSource?: string;
}

export interface AgentRunStatus {
  phase: AgentRunPhase;
  message: string;
  podName: string;
  traceID: string;
  startedAt: string;
  completedAt: string;
  logOutput?: string;
  deploymentName?: string;
  debugActive?: boolean;
}

export interface AgentRunEvent {
  agentRunId: string;
  type: string;
  payload: string;
  timestamp: string;
}

export const PHASE_OPTIONS: { value: AgentRunPhase; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "waiting_for_input", label: "Waiting" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

export const BACKEND_OPTIONS: { value: Backend; label: string }[] = [
  { value: "pod", label: "Pod" },
  { value: "kubevirt", label: "KubeVirt" },
  { value: "external", label: "External" },
];

export const MODEL_TIER_OPTIONS: { value: ModelTier; label: string }[] = [
  { value: "default", label: "Default (Local)" },
  { value: "default-cloud", label: "Default (Cloud)" },
  { value: "premium", label: "Premium" },
];

/** A single trace span from an agent run. */
export interface TraceSpan {
  id: string;
  parentId?: string;
  name: string;
  type: "llm" | "tool" | "thought" | "input";
  startTime: string;
  endTime: string;
  metadata?: Record<string, unknown>;
  hasDiff: boolean;
  diff?: SpanDiff;
}

/** Git diff captured for a trace span. */
export interface SpanDiff {
  files: FileDiff[];
}

/** A single file's patch within a span diff. */
export interface FileDiff {
  path: string;
  patch: string;
}
