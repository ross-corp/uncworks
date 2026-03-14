import type { AgentRun } from "../types/agent-run";
import { PhaseBadge } from "./StatusBadge";

export default function EventsView({ runs }: { runs: AgentRun[] }) {
  const events = runs
    .flatMap((run) => {
      const items: { time: string; runName: string; runId: string; type: string; detail: string; phase: AgentRun["status"]["phase"] }[] = [];
      items.push({
        time: run.createdAt,
        runName: run.name,
        runId: run.id,
        type: "created",
        detail: `Agent run created for ${run.spec.repos.map((repo) => repo.url.split("/").pop()).join(", ") || "unknown repo"}`,
        phase: run.status.phase,
      });
      if (run.status.startedAt) {
        items.push({
          time: run.status.startedAt,
          runName: run.name,
          runId: run.id,
          type: "started",
          detail: `Pod ${run.status.podName} started`,
          phase: run.status.phase,
        });
      }
      if (run.status.completedAt) {
        items.push({
          time: run.status.completedAt,
          runName: run.name,
          runId: run.id,
          type: run.status.phase,
          detail: run.status.message || `Run ${run.status.phase}`,
          phase: run.status.phase,
        });
      }
      return items;
    })
    .sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());

  if (events.length === 0) {
    return (
      <div className="px-6 py-12 text-center text-sm text-txt-tertiary">
        No events yet.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          <col style={{ width: 160 }} />
          <col style={{ width: 180 }} />
          <col style={{ width: 120 }} />
          <col />
        </colgroup>
        <thead>
          <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
            <th className="px-4 py-2">Time</th>
            <th className="px-4 py-2">Run</th>
            <th className="px-4 py-2">Phase</th>
            <th className="px-4 py-2">Detail</th>
          </tr>
        </thead>
        <tbody>
          {events.map((ev, i) => (
            <tr
              key={`${ev.runId}-${ev.type}-${i}`}
              className="border-b border-edge transition-colors hover:bg-surface-1"
            >
              <td className="px-4 py-2.5 font-mono text-xs text-txt-tertiary whitespace-nowrap">
                {new Date(ev.time).toLocaleTimeString()}
              </td>
              <td className="px-4 py-2.5 overflow-hidden text-ellipsis whitespace-nowrap">
                <span className="font-medium">{ev.runName}</span>
              </td>
              <td className="px-4 py-2.5">
                <PhaseBadge phase={ev.phase} />
              </td>
              <td className="px-4 py-2.5 text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                {ev.detail}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
