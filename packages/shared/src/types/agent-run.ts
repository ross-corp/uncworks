/** Backend type for an AgentRun. */
export type Backend = "Pod" | "KubeVirt" | "External";

/** Lifecycle phase of an AgentRun. */
export type AgentRunPhase =
  | "Pending"
  | "Running"
  | "WaitingForInput"
  | "Succeeded"
  | "Failed"
  | "Cancelled";

/** Repository to clone into the agent workspace. */
export interface Repository {
  url: string;
  branch?: string;
  path?: string;
}

/** Spec for creating an AgentRun. */
export interface AgentRunSpec {
  backend: Backend;
  repos: Repository[];
  prompt: string;
  devboxConfig?: string;
  ttlSeconds?: number;
  envVars?: Record<string, string>;
  modelTier?: string;
  image?: string;
  specContent?: string;
  specSource?: string;
  workspaceName?: string;
}

/** Status of an AgentRun. */
export interface AgentRunStatus {
  phase: AgentRunPhase;
  message?: string;
  podName?: string;
  traceID?: string;
  startedAt?: string;
  completedAt?: string;
}

/** Full AgentRun object. */
export interface AgentRun {
  id: string;
  name: string;
  spec: AgentRunSpec;
  status: AgentRunStatus;
  createdAt: string;
  updatedAt: string;
}

/** Event emitted for an AgentRun. */
export type AgentRunEventType =
  | "phase_changed"
  | "log"
  | "tool_call"
  | "waiting_for_input"
  | "completed";

export interface AgentRunEvent {
  agentRunId: string;
  type: AgentRunEventType;
  payload: string;
  timestamp: string;
}
