/** Backend type for an AgentRun. */
export type Backend = "pod";

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

/** Orchestration mode for an AgentRun. */
export type OrchestrationMode = "single" | "auto" | "manual" | "spec-driven";

/** A single task in a manual orchestration. */
export interface OrchestrationTask {
  name: string;
  prompt: string;
  repoUrls?: string[];
}

/** Orchestration configuration for manual mode. */
export interface Orchestration {
  tasks: OrchestrationTask[];
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
  parentRunId?: string;
  orchestrationMode?: OrchestrationMode;
  orchestration?: Orchestration;
  specRunId?: string;
  displayName?: string;
  project?: string;
  feature?: string;
  tags?: string[];
  projectRef?: string;
  specRef?: string;
  pipelineConfig?: {
    plan?: { model?: string; timeoutSeconds?: number; maxRetries?: number; onFailure?: string };
    execute?: { model?: string; timeoutSeconds?: number; maxRetries?: number; onFailure?: string };
    verify?: { model?: string; timeoutSeconds?: number; maxRetries?: number; onFailure?: string };
  };
  maxBudget?: number;
  autoPush?: boolean;
  autoPR?: boolean;
  prBaseBranch?: string;
}

/** Status of an AgentRun. */
export interface AgentRunStatus {
  phase: AgentRunPhase;
  message?: string;
  podName?: string;
  traceID?: string;
  startedAt?: string;
  completedAt?: string;
  logOutput?: string;
  retainUntil?: string;
  deploymentName?: string;
  debugActive?: boolean;
  stage?: string;
  retryCount?: number;
  verificationResult?: string;
  prUrl?: string;
}

/** Full AgentRun object. */
export interface AgentRun {
  id: string;
  name: string;
  spec: AgentRunSpec;
  status: AgentRunStatus;
  createdAt: string;
  updatedAt: string;
  children?: string[];
}

/** A node in the run graph. */
export interface RunGraphNode {
  name: string;
  phase: AgentRunPhase;
  role: string;
  startedAt?: string;
  completedAt?: string;
}

/** An edge in the run graph. */
export interface RunGraphEdge {
  parent: string;
  child: string;
}

/** The run graph for a spec execution. */
export interface RunGraph {
  nodes: RunGraphNode[];
  edges: RunGraphEdge[];
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
