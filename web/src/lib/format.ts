import type { AgentRunPhase } from "../types/agent-run";

/** Compute aggregate status for a group of runs. */
export function aggregatePhase(runs: { status: { phase: AgentRunPhase } }[]): AgentRunPhase {
  if (runs.some((r) => r.status.phase === "succeeded")) return "succeeded";
  if (runs.some((r) => r.status.phase === "running")) return "running";
  if (runs.some((r) => r.status.phase === "waiting_for_input")) return "waiting_for_input";
  if (runs.some((r) => r.status.phase === "pending")) return "pending";
  if (runs.every((r) => r.status.phase === "failed")) return "failed";
  if (runs.every((r) => r.status.phase === "cancelled")) return "cancelled";
  return "pending";
}

/** Format an ISO timestamp as a relative age string (e.g., "5m", "2h", "3d"). */
export function formatAge(iso: string): string {
  if (!iso) return "";
  const secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  return `${Math.floor(hrs / 24)}d`;
}
