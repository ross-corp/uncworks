import { createSignal } from "solid-js";
import AgentRunList, { type AgentRunItem } from "./components/AgentRunList";
import AgentRunDetail from "./components/AgentRunDetail";

const mockRuns: AgentRunItem[] = [
  {
    id: "ar-1",
    name: "fix-auth-bug",
    backend: "Pod",
    phase: "Running",
    prompt: "Fix the authentication bug in the login flow",
    createdAt: "2026-03-10T10:00:00Z",
  },
  {
    id: "ar-2",
    name: "add-tests",
    backend: "Pod",
    phase: "Succeeded",
    prompt: "Add unit tests for the user service",
    createdAt: "2026-03-10T09:00:00Z",
  },
  {
    id: "ar-3",
    name: "refactor-api",
    backend: "Pod",
    phase: "Pending",
    prompt: "Refactor the API endpoints to use the new schema",
    createdAt: "2026-03-10T11:00:00Z",
  },
];

const mockDetails: Record<string, any> = {
  "ar-1": {
    ...mockRuns[0],
    message: "Agent pod started, working on fix",
    podName: "agentrun-fix-auth-bug-pod",
    traceID: "abc123def456",
  },
  "ar-2": {
    ...mockRuns[1],
    message: "All tests passing",
    podName: "agentrun-add-tests-pod",
  },
  "ar-3": {
    ...mockRuns[2],
    message: "Queued, waiting for slot",
  },
};

export default function App() {
  const [selectedId, setSelectedId] = createSignal<string | null>(null);

  const selectedRun = () => (selectedId() ? mockDetails[selectedId()!] : null);

  return (
    <main style={{ "max-width": "960px", margin: "0 auto", padding: "16px" }}>
      <h1 data-testid="title">AOT Dashboard</h1>
      <p data-testid="status">Status: Ready</p>
      <div style={{ display: "grid", "grid-template-columns": "1fr 1fr", gap: "24px" }}>
        <AgentRunList
          runs={mockRuns}
          selectedId={selectedId()}
          onSelect={setSelectedId}
        />
        <AgentRunDetail run={selectedRun()} />
      </div>
    </main>
  );
}
