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
  { value: "default", label: "Local / offline", description: "qwen3:8b via Ollama" },
  { value: "qwen3:8b", label: "Local (fast)", description: "qwen3:8b" },
  { value: "llama3.1:8b", label: "Local (general)", description: "llama3.1:8b" },
  // Cloud top tier
  { value: "claude-sonnet-4.6", label: "Best quality", description: "Claude Sonnet 4.6" },
  { value: "claude-sonnet-4", label: "Best quality (prev)", description: "Claude Sonnet 4" },
  { value: "gemini-3-flash", label: "Best quality (Google)", description: "Gemini 3 Flash" },
  { value: "grok-4.1-fast", label: "Best quality (xAI)", description: "Grok 4.1 Fast" },
  // Cloud value
  { value: "default-cloud", label: "Fast & cheap", description: "DeepSeek V3.1 · $0.15/M" },
  { value: "deepseek-v3.1", label: "Fast & cheap", description: "DeepSeek V3.1 · $0.15/M" },
  { value: "deepseek-v3.2", label: "Fast & cheap (long ctx)", description: "DeepSeek V3.2 · 164K ctx" },
  { value: "gemini-flash", label: "Fast & cheap (huge ctx)", description: "Gemini 2.5 Flash · 1M ctx" },
  { value: "gpt-4.1-mini", label: "Fast & cheap (OpenAI)", description: "GPT-4.1 Mini · $0.40/M" },
  { value: "qwen3-coder", label: "Fast & cheap (coder)", description: "Qwen3 Coder · $0.22/M" },
  { value: "qwen3-235b", label: "Balanced (large)", description: "Qwen3 235B · $0.20/M" },
  { value: "kimi-k2.5", label: "Balanced (Moonshot)", description: "Kimi K2.5" },
  { value: "minimax-m2.5", label: "Balanced (MiniMax)", description: "MiniMax M2.5" },
  { value: "claude-haiku", label: "Fast & cheap (Anthropic)", description: "Claude Haiku 3.5 · $0.80/M" },
  { value: "mistral-medium", label: "Balanced (Mistral)", description: "Mistral Medium · $0.40/M" },
  // Free
  { value: "nemotron-3-super-free", label: "Free tier (large)", description: "Nemotron 3 Super · NVIDIA 120B" },
  { value: "step-flash-free", label: "Free tier", description: "Step 3.5 Flash · StepFun" },
  { value: "trinity-free", label: "Free tier (Arcee)", description: "Trinity Large" },
  { value: "qwen3-coder-free", label: "Free tier (coder)", description: "Qwen3 Coder · rate limited" },
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
