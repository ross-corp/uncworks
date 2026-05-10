// web/src/mocks/fixtures.ts — Typed fixture factories for MSW integration tests.
// Each factory returns a minimal valid object for its domain type.
// Override specific fields with the spread operator: agentRunFixture({ name: "my-run" })

import type { AgentRun, AgentRunPhase } from "../types/agent-run";
import type { AppSettings } from "../hooks/useSettings";

// ── AgentRun (the shared-package wire format, as returned by the API) ──────────

/** The raw API shape returned by GET /api/v1/runs — matches SharedAgentRun */
export interface RawAgentRun {
  id: string;
  name: string;
  spec: {
    backend: string;
    repos: { url: string; branch?: string; path?: string }[];
    workspaceName?: string;
    prompt: string;
    devboxConfig?: string;
    ttlSeconds?: number;
    envVars?: Record<string, string>;
    modelTier?: string;
    project?: string;
    projectRef?: string;
    displayName?: string;
    tags?: string[];
    orchestrationMode?: string;
  };
  status: {
    phase: string;
    message: string;
    podName: string;
    traceID: string;
    startedAt: string;
    completedAt: string;
  };
  createdAt: string;
}

export function rawAgentRunFixture(overrides: Partial<RawAgentRun> = {}): RawAgentRun {
  return {
    id: "run-001",
    name: "ar-test-run",
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/ross-corp/uncworks", branch: "main" }],
      prompt: "Fix the bug in auth.go",
      devboxConfig: "",
      ttlSeconds: 3600,
      envVars: {},
      modelTier: "default",
      projectRef: "my-project",
    },
    status: {
      phase: "Succeeded",
      message: "",
      podName: "ar-test-run-pod",
      traceID: "trace-001",
      startedAt: "2026-01-01T00:00:00Z",
      completedAt: "2026-01-01T00:05:00Z",
    },
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

/** The mapped web AgentRun (after mapRun) — use when testing components that expect the web type */
export function agentRunFixture(overrides: Partial<AgentRun> = {}): AgentRun {
  return {
    id: "run-001",
    name: "ar-test-run",
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/ross-corp/uncworks", branch: "main" }],
      prompt: "Fix the bug in auth.go",
      devboxConfig: "",
      ttlSeconds: 3600,
      envVars: {},
      modelTier: "default",
      projectRef: "my-project",
    },
    status: {
      phase: "succeeded" as AgentRunPhase,
      message: "",
      podName: "ar-test-run-pod",
      traceID: "trace-001",
      startedAt: "2026-01-01T00:00:00Z",
      completedAt: "2026-01-01T00:05:00Z",
    },
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:05:00Z",
    ...overrides,
  };
}

// ── Project ───────────────────────────────────────────────────────────────────

export interface ProjectSummary {
  name: string;
  displayName: string;
  description: string;
  repos: { url: string; branch: string }[];
  configRepoReady: boolean;
  configRepoMessage?: string;
  runCount: number;
  lastRunId: string;
  totalCost: string;
  createdAt: string;
}

export function projectFixture(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    name: "my-project",
    displayName: "My Project",
    description: "A test project",
    repos: [{ url: "https://github.com/ross-corp/uncworks", branch: "main" }],
    configRepoReady: true,
    runCount: 3,
    lastRunId: "ar-test-run",
    totalCost: "$0.05",
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

// ── AppSettings ───────────────────────────────────────────────────────────────

export function appSettingsFixture(overrides: Partial<AppSettings> = {}): AppSettings {
  return {
    githubToken: "",
    namespace: "uncworks",
    kubeContext: "colima-uncworks",
    portRangeStart: 50100,
    portRangeEnd: 50120,
    envOverrides: {},
    litellmURL: "https://openrouter.ai/api/v1",
    githubAuthed: false,
    updateChannel: "stable",
    autoUpdateEnabled: false,
    defaultManageModel: "google/gemini-2.5-pro",
    defaultImplementModel: "qwen/qwen3-coder",
    wizardComplete: true,
    apiserverURL: "http://localhost:50100",
    llmKeyConfigured: true,
    copilotModel: "google/gemini-2.5-flash-lite",
    ...overrides,
  };
}

// ── ServiceInfo ───────────────────────────────────────────────────────────────

export interface ServiceInfoFixture {
  name: string;
  displayName: string;
  clusterPort: number;
  localPort: number;
  ready: boolean;
  forwarding: boolean;
}

export function serviceInfoFixture(overrides: Partial<ServiceInfoFixture> = {}): ServiceInfoFixture {
  return {
    name: "apiserver",
    displayName: "API Server",
    clusterPort: 50055,
    localPort: 50100,
    ready: true,
    forwarding: true,
    ...overrides,
  };
}

// ── Chain ─────────────────────────────────────────────────────────────────────

export interface ChainFixture {
  name: string;
  displayName: string;
  description?: string;
  steps: { name: string; templateRef: string }[];
  createdAt: string;
}

export function chainFixture(overrides: Partial<ChainFixture> = {}): ChainFixture {
  return {
    name: "my-chain",
    displayName: "My Chain",
    steps: [{ name: "step-1", templateRef: "my-template" }],
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

// ── Template ──────────────────────────────────────────────────────────────────

export interface TemplateFixture {
  name: string;
  displayName: string;
  description?: string;
  projectRef: string;
  prompt: string;
  createdAt: string;
}

export function templateFixture(overrides: Partial<TemplateFixture> = {}): TemplateFixture {
  return {
    name: "my-template",
    displayName: "My Template",
    projectRef: "my-project",
    prompt: "Run the tests",
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

// ── Schedule ──────────────────────────────────────────────────────────────────

export interface ScheduleFixture {
  name: string;
  displayName: string;
  cron: string;
  chainRef: string;
  concurrencyPolicy: string;
  enabled: boolean;
  createdAt: string;
}

export function scheduleFixture(overrides: Partial<ScheduleFixture> = {}): ScheduleFixture {
  return {
    name: "my-schedule",
    displayName: "My Schedule",
    cron: "0 * * * *",
    chainRef: "my-chain",
    concurrencyPolicy: "Allow",
    enabled: true,
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}
