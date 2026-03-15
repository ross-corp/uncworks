import type { AgentRunPhase } from "./agent-run";

/** A single node in the orchestration graph. */
export interface GraphNode {
  runId: string;
  parentRunId: string | null;
  agentType: "spec" | "senior" | "junior";
  phase: AgentRunPhase;
  currentActivity: string;
  startedAt: string;
  completedAt: string;
  filesChanged: number;
  linesAdded: number;
  linesRemoved: number;
}

/** An edge in the orchestration graph. */
export interface GraphEdge {
  parentRunId: string;
  childRunId: string;
}

/** Position in the tree layout. */
export interface NodePosition {
  x: number;
  y: number;
}

/** Graph event types from SSE stream. */
export type GraphEventType = "NODE_ADDED" | "NODE_STATUS_CHANGED" | "NODE_PROGRESS";

export interface GraphEvent {
  type: GraphEventType;
  runId: string;
  parentRunId?: string;
  agentType?: GraphNode["agentType"];
  phase?: AgentRunPhase;
  message?: string;
  currentActivity?: string;
  elapsedSeconds?: number;
}

/** Whether a phase is terminal (no further transitions). */
export function isTerminalPhase(phase: AgentRunPhase): boolean {
  return phase === "succeeded" || phase === "failed" || phase === "cancelled";
}
