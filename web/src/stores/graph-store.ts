import { useSyncExternalStore } from "react";
import type { GraphNode, GraphEdge } from "../types/graph";
import type { AgentRunPhase } from "../types/agent-run";

export interface GraphState {
  nodes: Map<string, GraphNode>;
  edges: GraphEdge[];
  selectedRunId: string | null;
  reconnecting: boolean;
}

type Listener = () => void;

function createGraphStore() {
  let state: GraphState = {
    nodes: new Map(),
    edges: [],
    selectedRunId: null,
    reconnecting: false,
  };

  const listeners = new Set<Listener>();

  function notify() {
    for (const l of listeners) l();
  }

  function getSnapshot(): GraphState {
    return state;
  }

  function subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }

  function setGraph(nodes: GraphNode[], edges: GraphEdge[]) {
    const nodeMap = new Map<string, GraphNode>();
    for (const n of nodes) nodeMap.set(n.runId, n);
    state = { ...state, nodes: nodeMap, edges };
    notify();
  }

  function addNode(node: GraphNode) {
    const newNodes = new Map(state.nodes);
    newNodes.set(node.runId, node);
    const parentEdge: GraphEdge | null = node.parentRunId
      ? { parentRunId: node.parentRunId, childRunId: node.runId }
      : null;
    state = {
      ...state,
      nodes: newNodes,
      edges: parentEdge ? [...state.edges, parentEdge] : state.edges,
    };
    notify();
  }

  function updateNodeStatus(runId: string, phase: AgentRunPhase, _message?: string) {
    const existing = state.nodes.get(runId);
    if (!existing) return;
    const newNodes = new Map(state.nodes);
    newNodes.set(runId, {
      ...existing,
      phase,
      completedAt:
        phase === "succeeded" || phase === "failed" || phase === "cancelled"
          ? new Date().toISOString()
          : existing.completedAt,
    });
    state = { ...state, nodes: newNodes };
    notify();
  }

  function updateNodeProgress(runId: string, currentActivity: string) {
    const existing = state.nodes.get(runId);
    if (!existing) return;
    const newNodes = new Map(state.nodes);
    newNodes.set(runId, { ...existing, currentActivity });
    state = { ...state, nodes: newNodes };
    notify();
  }

  function setSelectedRunId(runId: string | null) {
    state = { ...state, selectedRunId: runId };
    notify();
  }

  function setReconnecting(reconnecting: boolean) {
    state = { ...state, reconnecting };
    notify();
  }

  function reset() {
    state = {
      nodes: new Map(),
      edges: [],
      selectedRunId: null,
      reconnecting: false,
    };
    notify();
  }

  return {
    getSnapshot,
    subscribe,
    setGraph,
    addNode,
    updateNodeStatus,
    updateNodeProgress,
    setSelectedRunId,
    setReconnecting,
    reset,
  };
}

// Singleton graph store
export const graphStore = createGraphStore();

/** React hook to subscribe to the graph store. */
export function useGraphStore(): GraphState {
  return useSyncExternalStore(graphStore.subscribe, graphStore.getSnapshot);
}
