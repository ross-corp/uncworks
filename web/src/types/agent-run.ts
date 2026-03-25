export type AgentRunPhase =
  | "pending"
  | "running"
  | "waiting_for_input"
  | "succeeded"
  | "failed"
  | "cancelled";

export type Backend = "pod";

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
  project?: string;
  feature?: string;
  tags?: string[];
  projectRef?: string;
  specRef?: string;
  maxBudget?: number;
  autoPush?: boolean;
  autoPR?: boolean;
  prBaseBranch?: string;
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
  prUrl?: string;
  archived?: boolean;
  totalCost?: string;
  totalAdditions?: number;
  totalDeletions?: number;
  ciFixAttempts?: number;
  lastCIStatus?: string;
  parentPRUrl?: string;
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
];

export const MODEL_TIER_OPTIONS: { value: ModelTier; label: string; description: string }[] = [
  // Local
  { value: "default", label: "Default (Local)", description: "Ollama qwen3:8b" },
  { value: "qwen3:8b", label: "qwen3:8b", description: "Local 8B coder" },
  { value: "llama3.1:8b", label: "llama3.1:8b", description: "Local all-rounder" },
  // Cloud top tier
  { value: "claude-sonnet-4.6", label: "Claude Sonnet 4.6", description: "Latest Anthropic" },
  { value: "claude-sonnet-4", label: "Claude Sonnet 4", description: "$3/M in" },
  { value: "gemini-3-flash", label: "Gemini 3 Flash", description: "Newest Google" },
  { value: "grok-4.1-fast", label: "Grok 4.1 Fast", description: "xAI" },
  // Cloud value
  { value: "default-cloud", label: "Cloud Default", description: "DeepSeek V3.1 $0.15/M" },
  { value: "deepseek-v3.1", label: "DeepSeek V3.1", description: "$0.15/M cheapest" },
  { value: "deepseek-v3.2", label: "DeepSeek V3.2", description: "164K ctx $0.26/M" },
  { value: "gemini-flash", label: "Gemini 2.5 Flash", description: "1M ctx $0.15/M" },
  { value: "gpt-4.1-mini", label: "GPT-4.1 Mini", description: "1M ctx $0.40/M" },
  { value: "qwen3-coder", label: "Qwen3 Coder", description: "262K ctx $0.22/M" },
  { value: "qwen3-235b", label: "Qwen3 235B", description: "$0.20/M" },
  { value: "kimi-k2.5", label: "Kimi K2.5", description: "Moonshot" },
  { value: "minimax-m2.5", label: "MiniMax M2.5", description: "MiniMax" },
  { value: "claude-haiku", label: "Claude Haiku 3.5", description: "$0.80/M" },
  { value: "mistral-medium", label: "Mistral Medium", description: "$0.40/M" },
  // Free
  { value: "nemotron-3-super-free", label: "Nemotron 3 Super", description: "Free NVIDIA 120B" },
  { value: "step-flash-free", label: "Step 3.5 Flash", description: "Free StepFun" },
  { value: "trinity-free", label: "Trinity Large", description: "Free Arcee" },
  { value: "qwen3-coder-free", label: "Qwen3 Coder", description: "Free rate limited" },
];

export const ORCHESTRATION_MODE_OPTIONS: { value: OrchestrationMode; label: string; description: string }[] = [
  { value: "single", label: "Greedy", description: "Single-pass execution" },
  { value: "spec-driven", label: "Progressive", description: "Plan, execute, verify loop" },
];

/** A single trace span from an agent run. */
export interface TraceSpan {
  id: string;
  traceId?: string;
  parentId?: string;
  name: string;
  type: "llm" | "tool" | "thought" | "input" | "delegate" | "lifecycle" | "stage" | "compaction";
  startTime: string;
  endTime: string;
  status?: "ok" | "error" | "unset";
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
