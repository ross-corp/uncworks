/**
 * gRPC data binding — connects the TUI state to the AOT API server.
 *
 * Fetches the initial agent run list, subscribes to live updates for the
 * selected run, and sends human input for HITL.
 */

import type { AOTClient } from "../../shared/src/grpc/client.js";
import type { AgentRun, AgentRunEvent } from "../../shared/src/types/agent-run.js";
import type { AgentRunView } from "./views.js";
import type { AppState } from "./state.js";

/** Convert a domain AgentRun to the view model used by the TUI. */
function toView(run: AgentRun): AgentRunView {
  return {
    id: run.id,
    name: run.name,
    phase: run.status.phase,
    backend: run.spec.backend,
    prompt: run.spec.prompt,
  };
}

export class DataBinding {
  private client: AOTClient;
  private state: AppState;
  private watchAbort: AbortController | null = null;
  private refreshInterval: ReturnType<typeof setInterval> | null = null;

  constructor(client: AOTClient, state: AppState) {
    this.client = client;
    this.state = state;
  }

  /** Fetch the initial agent run list. */
  async fetchRuns(): Promise<void> {
    try {
      const runs = await this.client.listAgentRuns();
      this.state.setRuns(runs.map(toView));
      this.state.setError(null);
    } catch (err) {
      this.state.setError(`Failed to fetch runs: ${(err as Error).message}`);
    }
  }

  /** Start watching a specific agent run for live updates. */
  watchRun(id: string): void {
    this.stopWatch();

    this.watchAbort = this.client.watchAgentRun(
      id,
      (event: AgentRunEvent) => {
        if (event.type === "phase_changed") {
          // Update the run's phase in the list
          const runs = this.state.runs().map((r) =>
            r.id === event.agentRunId ? { ...r, phase: event.payload } : r
          );
          this.state.setRuns(runs);
        }
      },
      (err: Error) => {
        this.state.setError(`Watch error: ${err.message}`);
      }
    );
  }

  /** Stop watching the current run. */
  stopWatch(): void {
    if (this.watchAbort) {
      this.watchAbort.abort();
      this.watchAbort = null;
    }
  }

  /** Send human input for a HITL-waiting run. */
  async sendInput(agentRunId: string, input: string): Promise<boolean> {
    try {
      const accepted = await this.client.sendHumanInput(agentRunId, input);
      if (accepted) {
        this.state.setError(null);
      } else {
        this.state.setError("Input was not accepted");
      }
      return accepted;
    } catch (err) {
      this.state.setError(`Send failed: ${(err as Error).message}`);
      return false;
    }
  }

  /** Start periodic refresh of the run list. */
  startAutoRefresh(intervalMs = 5000): void {
    this.refreshInterval = setInterval(() => {
      void this.fetchRuns();
    }, intervalMs);
  }

  /** Stop all data bindings. */
  stop(): void {
    this.stopWatch();
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
      this.refreshInterval = null;
    }
  }
}
