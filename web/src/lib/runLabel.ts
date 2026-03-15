import type { AgentRun } from "../types/agent-run";

/** Returns a human-readable label for a run, preferring displayName, then a
 *  truncated prompt (at word boundary), falling back to the K8s name. */
export function getRunLabel(run: AgentRun): string {
  if (run.spec.displayName) return run.spec.displayName;
  if (run.spec.prompt) {
    const truncated = run.spec.prompt.slice(0, 50);
    if (truncated.length < run.spec.prompt.length) {
      // Try to break at the last space for a clean word boundary
      const lastSpace = truncated.lastIndexOf(" ");
      return (lastSpace > 20 ? truncated.slice(0, lastSpace) : truncated) + "...";
    }
    return truncated;
  }
  return run.name;
}
