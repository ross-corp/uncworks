import { For, Show, createEffect, createSignal } from "solid-js";
import type { AgentRunEvent } from "../../../packages/shared/src/types/agent-run";

interface EventLogProps {
  events: AgentRunEvent[];
}

const typeColors: Record<string, string> = {
  phase_changed: "#8b5cf6",
  log: "#6b7280",
  tool_call: "#3b82f6",
  waiting_for_input: "#f59e0b",
  completed: "#10b981",
};

export default function EventLog(props: EventLogProps) {
  let containerRef: HTMLDivElement | undefined;
  const [autoScroll, setAutoScroll] = createSignal(true);

  function handleScroll() {
    if (!containerRef) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef;
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 50);
  }

  createEffect(() => {
    // Access events to track changes
    const _ = props.events.length;
    if (autoScroll() && containerRef) {
      containerRef.scrollTop = containerRef.scrollHeight;
    }
  });

  return (
    <div>
      <h2>Events</h2>
      <div
        ref={containerRef}
        onScroll={handleScroll}
        style={{
          height: "400px",
          "overflow-y": "auto",
          border: "1px solid #e5e7eb",
          "border-radius": "6px",
          padding: "8px",
          "font-family": "monospace",
          "font-size": "0.85em",
          background: "#1e1e1e",
          color: "#d4d4d4",
        }}
      >
        <Show when={props.events.length > 0} fallback={<div style={{ color: "#6b7280", padding: "8px" }}>Waiting for events...</div>}>
          <For each={props.events}>
            {(event) => {
              const time = new Date(event.timestamp).toLocaleTimeString();
              const color = typeColors[event.type] || "#6b7280";
              return (
                <div style={{ padding: "2px 0", "border-bottom": "1px solid #333" }}>
                  <span style={{ color: "#6b7280" }}>{time}</span>{" "}
                  <span style={{ color, "font-weight": "bold", "text-transform": "uppercase", "font-size": "0.8em" }}>[{event.type}]</span>{" "}
                  <span>{event.payload}</span>
                </div>
              );
            }}
          </For>
        </Show>
      </div>
    </div>
  );
}
