import { For, Show } from "solid-js";
import { A } from "@solidjs/router";

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
              <li style={{ margin: "4px 0" }}>
                <A
                  href={`/runs/${run.id}`}
                  style={{
                    display: "block",
                    padding: "8px 12px",
                    border: "2px solid #e5e7eb",
                    "border-radius": "6px",
                    cursor: "pointer",
                    background: "white",
                    "text-decoration": "none",
                    color: "inherit",
                  }}
                  data-testid={`run-${run.id}`}
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
                </A>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </div>
  );
}
