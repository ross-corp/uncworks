import type { AgentRunPhase } from "../types/agent-run";

/** Compute aggregate status for a group of runs.
 * Priority: running > waiting_for_input > pending > succeeded > failed(all) > cancelled(all) > pending
 */
export function aggregatePhase(runs: { status: { phase: AgentRunPhase } }[]): AgentRunPhase {
  if (runs.length === 0) return "pending";
  if (runs.some((r) => r.status.phase === "running")) return "running";
  if (runs.some((r) => r.status.phase === "waiting_for_input")) return "waiting_for_input";
  if (runs.some((r) => r.status.phase === "pending")) return "pending";
  if (runs.some((r) => r.status.phase === "succeeded")) return "succeeded";
  if (runs.every((r) => r.status.phase === "failed")) return "failed";
  if (runs.every((r) => r.status.phase === "cancelled")) return "cancelled";
  return "pending";
}

/** Format an ISO timestamp as a future-relative string (e.g., "in 5m", "in 2h", "overdue"). */
export function formatRelative(iso: string): string {
  if (!iso) return "";
  const ts = new Date(iso).getTime();
  if (isNaN(ts)) return "";
  const secs = Math.floor((ts - Date.now()) / 1000);
  if (secs < 0) return "overdue";
  if (secs < 60) return `in ${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `in ${hrs}h`;
  return `in ${Math.floor(hrs / 24)}d`;
}

/** Format the elapsed duration between two ISO timestamps (e.g., "2m30s", "1h5m"). */
export function formatDuration(startIso: string, endIso: string): string {
  if (!startIso || !endIso) return "";
  const start = new Date(startIso).getTime();
  const end = new Date(endIso).getTime();
  if (isNaN(start) || isNaN(end) || end <= start) return "";
  const secs = Math.floor((end - start) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  const remSecs = secs % 60;
  if (mins < 60) return remSecs > 0 ? `${mins}m${remSecs}s` : `${mins}m`;
  const hrs = Math.floor(mins / 60);
  const remMins = mins % 60;
  return remMins > 0 ? `${hrs}h${remMins}m` : `${hrs}h`;
}

/** Format an ISO timestamp as a relative age string (e.g., "5m", "2h", "3d"). */
export function formatAge(iso: string): string {
  if (!iso) return "";
  const ts = new Date(iso).getTime();
  if (isNaN(ts)) return "";
  const raw = Math.floor((Date.now() - ts) / 1000);
  const secs = Math.max(0, raw); // clamp future timestamps (clock skew) to 0
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  return `${Math.floor(hrs / 24)}d`;
}
