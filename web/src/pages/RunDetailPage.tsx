import { createSignal, createResource, onCleanup, Show } from "solid-js";
import { useParams, useNavigate } from "@solidjs/router";
import { AOTClient } from "../../../packages/shared/src/grpc/client";
import type { createAgentStore } from "../../../packages/shared/src/store/agent-store";
import AgentRunDetail from "../components/AgentRunDetail";
import EventLog from "../components/EventLog";
import HumanInputForm from "../components/HumanInputForm";

interface RunDetailPageProps {
  client: AOTClient;
  store: ReturnType<typeof createAgentStore>;
}

export default function RunDetailPage(props: RunDetailPageProps) {
  const params = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [streamError, setStreamError] = createSignal<string | null>(null);
  const [reconnecting, setReconnecting] = createSignal(false);
  const [cancelling, setCancelling] = createSignal(false);
  const [cancelError, setCancelError] = createSignal<string | null>(null);

  // Fetch run by ID
  const [run] = createResource(
    () => params.id,
    async (id) => {
      const result = await props.client.getAgentRun(id);
      props.store.updateRun(result);
      return result;
    }
  );

  // Start watch stream
  let abortController: AbortController | null = null;

  function startStream() {
    if (abortController) abortController.abort();
    setStreamError(null);
    setReconnecting(false);

    abortController = props.client.watchAgentRun(
      params.id,
      (event) => {
        props.store.addEvent(event);
        setReconnecting(false);
      },
      (err) => {
        if (!abortController?.signal.aborted) {
          setStreamError(err.message);
          setReconnecting(true);
          setTimeout(() => startStream(), 2000);
        }
      }
    );
  }

  startStream();

  onCleanup(() => {
    if (abortController) abortController.abort();
  });

  const currentRun = () => {
    const storeRun = props.store.state.runs[params.id];
    return storeRun ?? run() ?? null;
  };

  const runDetail = () => {
    const r = currentRun();
    if (!r) return null;
    return {
      id: r.id,
      name: r.name,
      backend: r.spec.backend,
      phase: r.status.phase,
      prompt: r.spec.prompt,
      createdAt: r.createdAt,
      message: r.status.message,
      podName: r.status.podName,
      traceID: r.status.traceID,
    };
  };

  const isTerminal = () => {
    const phase = currentRun()?.status.phase;
    return phase === "Succeeded" || phase === "Failed" || phase === "Cancelled";
  };

  const isWaiting = () => currentRun()?.status.phase === "WaitingForInput";

  async function handleCancel() {
    if (!confirm("Cancel this agent run?")) return;
    setCancelling(true);
    setCancelError(null);
    try {
      const updated = await props.client.cancelAgentRun(params.id);
      props.store.updateRun(updated);
    } catch (err) {
      setCancelError((err as Error).message);
    } finally {
      setCancelling(false);
    }
  }

  async function handleSendInput(input: string) {
    await props.client.sendHumanInput(params.id, input);
  }

  const events = () =>
    props.store.state.events.filter((e) => e.agentRunId === params.id);

  return (
    <div>
      <div style={{ display: "flex", "align-items": "center", gap: "12px", "margin-bottom": "16px" }}>
        <button
          onClick={() => navigate("/")}
          style={{ padding: "4px 12px", border: "1px solid #e5e7eb", "border-radius": "4px", background: "transparent", cursor: "pointer" }}
        >
          ← Back
        </button>
        <Show when={!isTerminal()}>
          <button
            onClick={handleCancel}
            disabled={cancelling()}
            style={{ padding: "4px 12px", border: "1px solid #ef4444", "border-radius": "4px", background: "transparent", color: "#ef4444", cursor: "pointer" }}
          >
            {cancelling() ? "Cancelling..." : "Cancel Run"}
          </button>
        </Show>
        <Show when={cancelError()}>
          <span style={{ color: "#d32f2f", "font-size": "0.85em" }}>{cancelError()}</span>
        </Show>
      </div>

      <Show when={reconnecting()}>
        <div style={{ padding: "8px", background: "#fff3cd", "border-radius": "4px", "margin-bottom": "12px", color: "#856404" }}>
          Reconnecting to event stream...
        </div>
      </Show>

      <Show when={run.loading} fallback={
        <div style={{ display: "grid", "grid-template-columns": "1fr 1fr", gap: "24px" }}>
          <div>
            <AgentRunDetail run={runDetail()} />
            <Show when={isWaiting()}>
              <HumanInputForm onSubmit={handleSendInput} />
            </Show>
          </div>
          <EventLog events={events()} />
        </div>
      }>
        <p>Loading run...</p>
      </Show>
    </div>
  );
}
