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
  updatedAt: string;
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
  approvalMode?: string;
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

export const MODEL_OPTIONS: { value: ModelTier; label: string; description: string }[] = [
  // Local
  { value: "default",       label: "default",              description: "qwen3:8b via Ollama" },
  { value: "qwen3:8b",      label: "qwen3:8b",             description: "Local · fast" },
  { value: "llama3.1:8b",   label: "llama3.1:8b",          description: "Local · general" },
  // Anthropic
  { value: "claude-sonnet-4.6", label: "claude-sonnet-4.6", description: "Anthropic · best" },
  { value: "claude-sonnet-4",   label: "claude-sonnet-4",   description: "Anthropic · prev" },
  { value: "claude-haiku",      label: "claude-haiku",       description: "Anthropic · $0.80/M" },
  // OpenAI
  { value: "gpt-4.1-mini",  label: "gpt-4.1-mini",         description: "OpenAI · $0.40/M" },
  // Google
  { value: "gemini-3-flash", label: "gemini-3-flash",       description: "Google · best" },
  { value: "gemini-flash",   label: "gemini-flash",          description: "Google · 1M ctx" },
  // DeepSeek
  { value: "deepseek-v3.1", label: "deepseek-v3.1",         description: "$0.15/M" },
  { value: "deepseek-v3.2", label: "deepseek-v3.2",         description: "164K ctx" },
  { value: "default-cloud", label: "default-cloud",          description: "DeepSeek V3.1 alias" },
  // Qwen
  { value: "qwen3-coder",   label: "qwen3-coder",           description: "$0.22/M" },
  { value: "qwen3-235b",    label: "qwen3-235b",            description: "$0.20/M · 235B" },
  // Others
  { value: "grok-4.1-fast", label: "grok-4.1-fast",         description: "xAI" },
  { value: "kimi-k2.5",              label: "kimi-k2.5",              description: "Moonshot" },
  { value: "moonshotai/kimi-k2.5",   label: "moonshotai/kimi-k2.5",   description: "Moonshot · OpenRouter" },
  { value: "z-ai/glm-4.7-flash",     label: "z-ai/glm-4.7-flash",     description: "ZhipuAI · OpenRouter" },
  { value: "minimax-m2.5",  label: "minimax-m2.5",          description: "MiniMax" },
  { value: "mistral-medium", label: "mistral-medium",        description: "$0.40/M" },
  // Free
  { value: "nemotron-3-super-free", label: "nemotron-3-super-free", description: "NVIDIA 120B · free" },
  { value: "step-flash-free",       label: "step-flash-free",       description: "StepFun · free" },
  { value: "trinity-free",          label: "trinity-free",           description: "Arcee · free" },
  { value: "qwen3-coder-free",      label: "qwen3-coder-free",      description: "rate limited · free" },
];

/** @deprecated use MODEL_OPTIONS */
export const MODEL_TIER_OPTIONS = MODEL_OPTIONS;

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
