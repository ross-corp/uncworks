import { createSignal, createResource, onCleanup, Show } from "solid-js";
import { AOTClient } from "../../packages/shared/src/grpc/client";
import type { AgentRun } from "../../packages/shared/src/types/agent-run";
import AgentRunList, { type AgentRunItem } from "./components/AgentRunList";
import AgentRunDetail from "./components/AgentRunDetail";

// In dev mode, vite proxies /aot.api.v1.AOTService/* to the API server (port 50055).
// In production, set VITE_API_URL to the API server URL.
const API_BASE_URL = import.meta.env.VITE_API_URL ?? "";

const client = new AOTClient({ baseUrl: API_BASE_URL });

function toListItem(run: AgentRun): AgentRunItem {
  return {
    id: run.id,
    name: run.name,
    backend: run.spec.backend,
    phase: run.status.phase,
    prompt: run.spec.prompt,
    createdAt: run.createdAt,
  };
}

function toDetail(run: AgentRun) {
  return {
    id: run.id,
    name: run.name,
    backend: run.spec.backend,
    phase: run.status.phase,
    prompt: run.spec.prompt,
    createdAt: run.createdAt,
    message: run.status.message,
    podName: run.status.podName,
    traceID: run.status.traceID,
  };
}

export default function App() {
  const [selectedId, setSelectedId] = createSignal<string | null>(null);
  const [error, setError] = createSignal<string | null>(null);
  const [previousRuns, setPreviousRuns] = createSignal<AgentRun[]>([]);

  const [runs, { refetch }] = createResource(async () => {
    try {
      const result = await client.listAgentRuns();
      setError(null);
      setPreviousRuns(result);
      return result;
    } catch (err) {
      setError((err as Error).message);
      return previousRuns();
    }
  });

  // Auto-refresh every 5 seconds
  const interval = setInterval(() => void refetch(), 5000);
  onCleanup(() => clearInterval(interval));

  const listItems = () => (runs() ?? []).map(toListItem);
  const selectedRun = () => {
    const id = selectedId();
    if (!id) return null;
    const run = (runs() ?? []).find((r) => r.id === id);
    return run ? toDetail(run) : null;
  };

  const statusText = () => {
    if (runs.loading && (runs() ?? []).length === 0) return "Loading...";
    if (error()) return `Error: ${error()}`;
    return `Connected (${(runs() ?? []).length} runs)`;
  };

  return (
    <main style={{ "max-width": "960px", margin: "0 auto", padding: "16px" }}>
      <h1 data-testid="title">AOT Dashboard</h1>
      <div data-testid="status" style={{ display: "flex", "align-items": "center", gap: "8px", "margin-bottom": "16px" }}>
        <Show when={runs.loading && (runs() ?? []).length === 0}>
          <span style={{ color: "#888" }}>Loading...</span>
        </Show>
        <Show when={error()}>
          <span style={{ color: "#d32f2f" }}>{statusText()}</span>
          <button
            data-testid="retry-button"
            onClick={() => void refetch()}
            style={{
              padding: "4px 12px",
              border: "1px solid #d32f2f",
              "border-radius": "4px",
              background: "transparent",
              color: "#d32f2f",
              cursor: "pointer",
            }}
          >
            Retry
          </button>
        </Show>
        <Show when={!error() && !(runs.loading && (runs() ?? []).length === 0)}>
          <span style={{ color: "#388e3c" }}>{statusText()}</span>
          <Show when={runs.loading}>
            <span style={{ color: "#888", "font-size": "0.85em" }}>(refreshing...)</span>
          </Show>
        </Show>
      </div>
      <div style={{ display: "grid", "grid-template-columns": "1fr 1fr", gap: "24px" }}>
        <AgentRunList
          runs={listItems()}
          selectedId={selectedId()}
          onSelect={setSelectedId}
        />
        <AgentRunDetail run={selectedRun()} />
      </div>
    </main>
  );
}
