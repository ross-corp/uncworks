import { For, Show } from "solid-js";

export interface AgentRunItem {
  id: string;
  name: string;
  backend: string;
  phase: string;
  prompt: string;
  createdAt: string;
}

interface AgentRunListProps {
  runs: AgentRunItem[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

const phaseColors: Record<string, string> = {
  Pending: "#f59e0b",
  Running: "#3b82f6",
  WaitingForInput: "#8b5cf6",
  Succeeded: "#10b981",
  Failed: "#ef4444",
  Cancelled: "#6b7280",
};

export default function AgentRunList(props: AgentRunListProps) {
  return (
    <div data-testid="agent-run-list">
      <h2>Agent Runs</h2>
      <Show
        when={props.runs.length > 0}
        fallback={<p data-testid="empty-state">No agent runs</p>}
      >
        <ul style={{ "list-style": "none", padding: 0 }}>
          <For each={props.runs}>
            {(run) => (
              <li
                data-testid={`run-${run.id}`}
                onClick={() => props.onSelect(run.id)}
                style={{
                  padding: "8px 12px",
                  margin: "4px 0",
                  border: `2px solid ${props.selectedId === run.id ? "#3b82f6" : "#e5e7eb"}`,
                  "border-radius": "6px",
                  cursor: "pointer",
                  background: props.selectedId === run.id ? "#eff6ff" : "white",
                }}
              >
                <div style={{ display: "flex", "justify-content": "space-between" }}>
                  <strong>{run.name}</strong>
                  <span
                    data-testid={`phase-${run.id}`}
                    style={{
                      color: phaseColors[run.phase] || "#6b7280",
                      "font-weight": "bold",
                    }}
                  >
                    {run.phase}
                  </span>
                </div>
                <div style={{ "font-size": "0.85em", color: "#6b7280" }}>
                  {run.backend} | {run.prompt.slice(0, 60)}
                  {run.prompt.length > 60 ? "..." : ""}
                </div>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </div>
  );
}
