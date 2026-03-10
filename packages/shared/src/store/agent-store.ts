import { createStore, reconcile } from "solid-js/store";
import type { AgentRun, AgentRunEvent, AgentRunPhase } from "../types/agent-run";

export interface AgentStoreState {
  runs: Record<string, AgentRun>;
  events: AgentRunEvent[];
  selectedRunId: string | null;
  filter: AgentRunPhase | null;
}

/** Creates a reactive Solid store for managing AgentRun state. */
export function createAgentStore() {
  const [state, setState] = createStore<AgentStoreState>({
    runs: {},
    events: [],
    selectedRunId: null,
    filter: null,
  });

  function setRuns(runs: AgentRun[]) {
    const map: Record<string, AgentRun> = {};
    for (const run of runs) {
      map[run.id] = run;
    }
    setState("runs", reconcile(map));
  }

  function updateRun(run: AgentRun) {
    setState("runs", run.id, run);
  }

  function removeRun(id: string) {
    setState("runs", (prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  }

  function addEvent(event: AgentRunEvent) {
    setState("events", (prev) => [...prev.slice(-999), event]);

    // Update run phase from event if applicable
    if (event.type === "phase_changed") {
      setState("runs", event.agentRunId, "status", "phase", event.payload as AgentRunPhase);
    }
  }

  function selectRun(id: string | null) {
    setState("selectedRunId", id);
  }

  function setFilter(phase: AgentRunPhase | null) {
    setState("filter", phase);
  }

  function getFilteredRuns(): AgentRun[] {
    const runs = Object.values(state.runs);
    if (!state.filter) return runs;
    return runs.filter((r) => r.status.phase === state.filter);
  }

  function getSelectedRun(): AgentRun | undefined {
    if (!state.selectedRunId) return undefined;
    return state.runs[state.selectedRunId];
  }

  return {
    state,
    setRuns,
    updateRun,
    removeRun,
    addEvent,
    selectRun,
    setFilter,
    getFilteredRuns,
    getSelectedRun,
  };
}
