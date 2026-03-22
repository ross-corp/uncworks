import type { AgentRun } from "../../types/agent-run";

const now = new Date().toISOString();
const fiveMinAgo = new Date(Date.now() - 5 * 60000).toISOString();
const oneHourAgo = new Date(Date.now() - 60 * 60000).toISOString();

export const mockRuns: AgentRun[] = [
  {
    id: "run-abc-123",
    name: "fix-auth-middleware",
    createdAt: fiveMinAgo,
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/acme/backend.git", branch: "fix/auth" }],
      prompt: "Fix the JWT validation in the auth middleware. The token expiry check is off by one hour.",
      devboxConfig: "",
      ttlSeconds: 3600,
      envVars: { NODE_ENV: "development" },
      modelTier: "default-cloud",
    },
    status: {
      phase: "running",
      message: "Analyzing auth middleware...",
      podName: "agent-abc-123",
      traceID: "trace-xyz-789",
      startedAt: fiveMinAgo,
      completedAt: "",
    },
  },
  {
    id: "run-def-456",
    name: "add-user-search",
    createdAt: oneHourAgo,
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/acme/frontend.git", branch: "feat/search" }],
      prompt: "Add a user search component with autocomplete.",
      devboxConfig: "",
      ttlSeconds: 7200,
      envVars: {},
      modelTier: "premium",
    },
    status: {
      phase: "waiting_for_input",
      message: "Should the search results include inactive users?",
      podName: "agent-def-456",
      traceID: "trace-uvw-321",
      startedAt: oneHourAgo,
      completedAt: "",
    },
  },
  {
    id: "run-ghi-789",
    name: "refactor-db-layer",
    createdAt: oneHourAgo,
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/acme/backend.git", branch: "main" }],
      prompt: "Refactor the database layer to use connection pooling.",
      devboxConfig: "",
      ttlSeconds: 3600,
      envVars: {},
      modelTier: "default",
    },
    status: {
      phase: "succeeded",
      message: "Refactoring complete. 12 files changed, all tests passing.",
      podName: "agent-ghi-789",
      traceID: "trace-rst-654",
      startedAt: oneHourAgo,
      completedAt: now,
    },
  },
  {
    id: "run-jkl-012",
    name: "update-ci-pipeline",
    createdAt: now,
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/acme/infra.git", branch: "main" }],
      prompt: "Update the CI pipeline to use Node 20.",
      devboxConfig: "",
      ttlSeconds: 1800,
      envVars: {},
      modelTier: "default-cloud",
    },
    status: {
      phase: "pending",
      message: "",
      podName: "",
      traceID: "",
      startedAt: "",
      completedAt: "",
    },
  },
  {
    id: "run-mno-345",
    name: "fix-memory-leak",
    createdAt: oneHourAgo,
    spec: {
      backend: "pod",
      repos: [{ url: "https://github.com/acme/backend.git", branch: "fix/memory" }],
      prompt: "Investigate and fix the memory leak in the WebSocket handler.",
      devboxConfig: "",
      ttlSeconds: 3600,
      envVars: {},
      modelTier: "premium",
    },
    status: {
      phase: "failed",
      message: "Error: OOMKilled — pod exceeded memory limit",
      podName: "agent-mno-345",
      traceID: "trace-opq-987",
      startedAt: oneHourAgo,
      completedAt: now,
    },
  },
];
