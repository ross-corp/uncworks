import { createResource, onCleanup, createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { AOTClient } from "../../../packages/shared/src/grpc/client";
import type { AgentRun } from "../../../packages/shared/src/types/agent-run";
import type { createAgentStore } from "../../../packages/shared/src/store/agent-store";
import AgentRunList from "../components/AgentRunList";
import CreateRunForm from "../components/CreateRunForm";

interface RunListPageProps {
  client: AOTClient;
  store: ReturnType<typeof createAgentStore>;
}

export default function RunListPage(props: RunListPageProps) {
  const navigate = useNavigate();
  const [error, setError] = createSignal<string | null>(null);
  const [showCreate, setShowCreate] = createSignal(false);
  const [previousRuns, setPreviousRuns] = createSignal<AgentRun[]>([]);

  const [runs, { refetch }] = createResource(async () => {
    try {
      const result = await props.client.listAgentRuns();
      setError(null);
      setPreviousRuns(result);
      props.store.setRuns(result);
      return result;
    } catch (err) {
      setError((err as Error).message);
      return previousRuns();
    }
  });

  const interval = setInterval(() => void refetch(), 5000);
  onCleanup(() => clearInterval(interval));

  const listItems = () =>
    (runs() ?? []).map((r) => ({
      id: r.id,
      name: r.name,
      backend: r.spec.backend,
      phase: r.status.phase,
      prompt: r.spec.prompt,
      createdAt: r.createdAt,
    }));

  const statusText = () => {
    if (runs.loading && (runs() ?? []).length === 0) return "Loading...";
    if (error()) return `Error: ${error()}`;
    return `${(runs() ?? []).length} runs`;
  };

  return (
    <div>
      <div style={{ display: "flex", "justify-content": "space-between", "align-items": "center", "margin-bottom": "16px" }}>
        <div style={{ display: "flex", "align-items": "center", gap: "8px" }}>
          <Show when={error()}>
            <span style={{ color: "#d32f2f" }}>{statusText()}</span>
            <button onClick={() => void refetch()} style={{ padding: "4px 12px", border: "1px solid #d32f2f", "border-radius": "4px", background: "transparent", color: "#d32f2f", cursor: "pointer" }}>Retry</button>
          </Show>
          <Show when={!error()}>
            <span style={{ color: "#388e3c" }}>{statusText()}</span>
          </Show>
        </div>
        <button
          onClick={() => setShowCreate(!showCreate())}
          style={{ padding: "8px 16px", background: "#3b82f6", color: "white", border: "none", "border-radius": "6px", cursor: "pointer", "font-weight": "bold" }}
        >
          {showCreate() ? "Cancel" : "New Run"}
        </button>
      </div>

      <Show when={showCreate()}>
        <CreateRunForm
          client={props.client}
          onCreated={(run) => {
            setShowCreate(false);
            void refetch();
            navigate(`/runs/${run.id}`);
          }}
        />
      </Show>

      <AgentRunList runs={listItems()} />
    </div>
  );
}
