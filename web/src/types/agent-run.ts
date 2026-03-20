export type AgentRunPhase =
  | "pending"
  | "running"
  | "waiting_for_input"
  | "succeeded"
  | "failed"
  | "cancelled";

export type Backend = "pod" | "kubevirt" | "external";

export type ModelTier = "default" | "default-cloud" | "premium" | string;

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
  children?: string[];
}

export interface RunGraphNode {
  name: string;
  phase: AgentRunPhase;
  role: string;
  startedAt?: string;
  completedAt?: string;
}

export interface RunGraphEdge {
  parent: string;
  child: string;
}

export interface RunGraph {
  nodes: RunGraphNode[];
  edges: RunGraphEdge[];
}

export type OrchestrationMode = "single" | "auto" | "manual" | "spec-driven";

export interface OrchestrationTask {
  name: string;
  prompt: string;
  repoUrls?: string[];
}

export interface Orchestration {
  tasks: OrchestrationTask[];
}

export interface StageConfig {
  model?: string;
  timeoutSeconds?: number;
  maxRetries?: number;
  onFailure?: "retry" | "fail" | "skip";
}

export interface PipelineConfig {
  plan?: StageConfig;
  execute?: StageConfig;
  verify?: StageConfig;
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
  parentRunId?: string;
  orchestrationMode?: OrchestrationMode;
  orchestration?: Orchestration;
  specRunId?: string;
  displayName?: string;
  pipelineConfig?: PipelineConfig;
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
  stage?: string;
  retryCount?: number;
  verificationResult?: string;
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

export const MODEL_TIER_OPTIONS: { value: ModelTier; label: string; description: string }[] = [
  { value: "default", label: "Default (Local)", description: "Ollama · qwen3:8b · No rate limits" },
  { value: "qwen3:8b", label: "qwen3:8b", description: "Local · Ollama · Best 8B coder" },
  { value: "llama3.1:8b", label: "llama3.1:8b", description: "Local · Ollama · All-rounder" },
  { value: "default-cloud", label: "Cloud", description: "OpenRouter · qwen3-coder · Rate limited" },
  { value: "qwen3-coder", label: "qwen3-coder", description: "Cloud · OpenRouter · Free" },
  { value: "mistral-small", label: "mistral-small-3.1-24b", description: "Cloud · OpenRouter · Free" },
  { value: "qwen2.5:0.5b", label: "qwen2.5:0.5b", description: "Local · Ollama · CI only" },
];

export const ORCHESTRATION_MODE_OPTIONS: { value: OrchestrationMode; label: string }[] = [
  { value: "single", label: "Single" },
  { value: "spec-driven", label: "Spec-Driven" },
  { value: "auto", label: "Auto" },
  { value: "manual", label: "Manual" },
];

/** A single trace span from an agent run. */
export interface TraceSpan {
  id: string;
  parentId?: string;
  name: string;
  type: "llm" | "tool" | "thought" | "input" | "delegate";
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
