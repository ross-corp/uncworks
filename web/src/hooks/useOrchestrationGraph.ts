import { useEffect, useRef } from "react";
import { graphStore } from "../stores/graph-store";
import type { GraphNode, GraphEdge, GraphEvent } from "../types/graph";

/**
 * Hook that fetches the orchestration graph on mount and subscribes
 * to the WatchSpecRunGraph SSE stream for real-time updates.
 */
export function useOrchestrationGraph(specRunId: string | null) {
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!specRunId) {
      graphStore.reset();
      return;
    }

    const abort = new AbortController();
    abortRef.current = abort;

    // Fetch initial graph
    async function fetchGraph() {
      try {
        const res = await fetch(`/api/v1/specs/${specRunId}/graph`, {
          signal: abort.signal,
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = (await res.json()) as { nodes: GraphNode[]; edges: GraphEdge[] };
        graphStore.setGraph(data.nodes ?? [], data.edges ?? []);
      } catch (err) {
        if (!abort.signal.aborted) {
          console.error("Failed to fetch orchestration graph:", err);
        }
      }
    }

    // Subscribe to SSE stream
    function connectSSE() {
      const eventSource = new EventSource(`/api/v1/specs/${specRunId}/graph/watch`);

      eventSource.onopen = () => {
        graphStore.setReconnecting(false);
      };

      eventSource.onmessage = (msg) => {
        try {
          const event = JSON.parse(msg.data) as GraphEvent;
          switch (event.type) {
            case "NODE_ADDED":
              graphStore.addNode({
                runId: event.runId,
                parentRunId: event.parentRunId ?? null,
                agentType: event.agentType ?? "junior",
                phase: "pending",
                currentActivity: "",
                startedAt: new Date().toISOString(),
                completedAt: "",
                filesChanged: 0,
                linesAdded: 0,
                linesRemoved: 0,
              });
              break;
            case "NODE_STATUS_CHANGED":
              if (event.phase) {
                graphStore.updateNodeStatus(event.runId, event.phase, event.message);
              }
              break;
            case "NODE_PROGRESS":
              if (event.currentActivity !== undefined) {
                graphStore.updateNodeProgress(event.runId, event.currentActivity);
              }
              break;
          }
        } catch {
          // Ignore malformed events
        }
      };

      eventSource.onerror = () => {
        graphStore.setReconnecting(true);
        eventSource.close();
        // Auto-reconnect after 3s
        if (!abort.signal.aborted) {
          setTimeout(() => {
            if (!abort.signal.aborted) {
              fetchGraph();
              connectSSE();
            }
          }, 3000);
        }
      };

      // Clean up SSE on abort
      abort.signal.addEventListener("abort", () => {
        eventSource.close();
      });
    }

    fetchGraph();
    connectSSE();

    return () => {
      abort.abort();
      abortRef.current = null;
    };
  }, [specRunId]);
}
