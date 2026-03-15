import type { AgentRun } from "../types/agent-run";

/** Returns a short identifier for a run: displayName if set, otherwise the
 *  K8s name (e.g. "ar-o7ar3t"). Prompt text is intentionally omitted — the
 *  Prompt column handles that. */
export function getRunLabel(run: AgentRun): string {
  if (run.spec.displayName) return run.spec.displayName;
  return run.name;
}
