import { Show } from "solid-js";

interface AgentRunDetailProps {
  run: {
    id: string;
    name: string;
    backend: string;
    phase: string;
    prompt: string;
    createdAt: string;
    message?: string;
    podName?: string;
    traceID?: string;
  } | null;
}

export default function AgentRunDetail(props: AgentRunDetailProps) {
  return (
    <div data-testid="agent-run-detail">
      <Show when={props.run} fallback={<p data-testid="no-selection">Select an agent run</p>}>
        {(run) => (
          <div>
            <h2 data-testid="detail-name">{run().name}</h2>
            <table style={{ width: "100%", "border-collapse": "collapse" }}>
              <tbody>
                <tr>
                  <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Phase</td>
                  <td data-testid="detail-phase" style={{ padding: "4px 8px" }}>{run().phase}</td>
                </tr>
                <tr>
                  <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Backend</td>
                  <td data-testid="detail-backend" style={{ padding: "4px 8px" }}>{run().backend}</td>
                </tr>
                <tr>
                  <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Prompt</td>
                  <td data-testid="detail-prompt" style={{ padding: "4px 8px" }}>{run().prompt}</td>
                </tr>
                <Show when={run().podName}>
                  <tr>
                    <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Pod</td>
                    <td data-testid="detail-pod" style={{ padding: "4px 8px" }}>{run().podName}</td>
                  </tr>
                </Show>
                <Show when={run().traceID}>
                  <tr>
                    <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Trace ID</td>
                    <td data-testid="detail-trace" style={{ padding: "4px 8px" }}>{run().traceID}</td>
                  </tr>
                </Show>
                <Show when={run().message}>
                  <tr>
                    <td style={{ padding: "4px 8px", "font-weight": "bold" }}>Message</td>
                    <td data-testid="detail-message" style={{ padding: "4px 8px" }}>{run().message}</td>
                  </tr>
                </Show>
              </tbody>
            </table>
          </div>
        )}
      </Show>
    </div>
  );
}
