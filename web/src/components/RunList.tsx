import type { AgentRun } from "../types/agent-run";

interface RunListProps {
  runs: AgentRun[];
  selectedId: string | null;
  onSelect: (run: AgentRun) => void;
  onDoubleClick?: (run: AgentRun) => void;
  loading?: boolean;
}

const PHASE_COLORS: Record<string, string> = {
  running: "var(--unc-active, #3b82f6)",
  waiting_for_input: "var(--unc-warning, #f59e0b)",
  pending: "var(--unc-neutral, #6b7280)",
  succeeded: "var(--unc-success, #22c55e)",
  failed: "var(--unc-error, #ef4444)",
  cancelled: "var(--unc-neutral, #6b7280)",
};

function formatAge(iso: string): string {
  if (!iso) return "-";
  const secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

function repoName(url: string): string {
  const parts = url.replace(/\.git$/, "").split("/");
  return parts.pop() ?? url;
}

export function RunList({ runs, selectedId, onSelect, onDoubleClick, loading }: RunListProps) {
  if (loading) {
    return (
      <div data-testid="run-list" className="flex h-full items-center justify-center text-sm" style={{ color: "var(--unc-muted)" }}>
        Loading...
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div data-testid="run-list" className="flex h-full items-center justify-center text-sm" style={{ color: "var(--unc-muted)" }}>
        No runs
      </div>
    );
  }

  return (
    <div data-testid="run-list" className="h-full overflow-y-auto">
      <table
        role="grid"
        style={{ tableLayout: "fixed", width: "100%", borderCollapse: "collapse" }}
      >
        <thead>
          <tr
            style={{
              height: "28px",
              fontSize: "11px",
              color: "var(--unc-muted)",
              borderBottom: "1px solid var(--unc-border)",
            }}
          >
            <th style={{ width: "20px", padding: "0 4px" }}></th>
            <th style={{ width: "160px", padding: "0 8px", textAlign: "left", fontWeight: 500 }}>Name</th>
            <th style={{ padding: "0 8px", textAlign: "left", fontWeight: 500 }}>Prompt</th>
            <th style={{ width: "120px", padding: "0 8px", textAlign: "left", fontWeight: 500 }}>Repo</th>
            <th style={{ width: "80px", padding: "0 8px", textAlign: "left", fontWeight: 500 }}>Phase</th>
            <th style={{ width: "60px", padding: "0 8px", textAlign: "right", fontWeight: 500 }}>Age</th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run, i) => {
            const isSelected = run.id === selectedId;
            const isEven = i % 2 === 1;
            const isRunning = run.status.phase === "running";

            return (
              <tr
                key={run.id}
                data-run-id={run.id}
                data-testid={`run-row-${run.id}`}
                aria-selected={isSelected}
                onClick={() => onSelect(run)}
                onDoubleClick={() => onDoubleClick?.(run)}
                style={{
                  height: "32px",
                  fontSize: "13px",
                  lineHeight: "1.4",
                  cursor: "pointer",
                  backgroundColor: isSelected
                    ? "color-mix(in srgb, var(--unc-accent) 10%, transparent)"
                    : isEven
                    ? "color-mix(in srgb, var(--unc-muted) 5%, transparent)"
                    : "transparent",
                  borderLeft: isSelected ? "2px solid var(--unc-accent)" : "2px solid transparent",
                  color: "var(--unc-fg)",
                }}
              >
                <td style={{ padding: "0 4px", textAlign: "center" }}>
                  <span
                    style={{
                      display: "inline-block",
                      width: "8px",
                      height: "8px",
                      borderRadius: "50%",
                      backgroundColor: PHASE_COLORS[run.status.phase] ?? "var(--unc-neutral)",
                      animation: isRunning ? "pulse 2s infinite" : "none",
                    }}
                  />
                </td>
                <td
                  style={{
                    padding: "0 8px",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    fontFamily: "monospace",
                  }}
                >
                  <span style={{ fontSize: "13px" }}>{run.name}</span>
                  {run.spec.displayName && (
                    <span style={{ fontSize: "11px", color: "var(--unc-muted)", marginLeft: "6px" }}>
                      {run.spec.displayName}
                    </span>
                  )}
                </td>
                <td
                  style={{
                    padding: "0 8px",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                  title={run.spec.prompt}
                >
                  {run.spec.prompt.slice(0, 60)}{run.spec.prompt.length > 60 ? "..." : ""}
                </td>
                <td
                  style={{
                    padding: "0 8px",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    fontSize: "12px",
                  }}
                >
                  {run.spec.repos.length > 0 ? repoName(run.spec.repos[0].url) : "-"}
                </td>
                <td
                  style={{
                    padding: "0 8px",
                    fontSize: "12px",
                    color: PHASE_COLORS[run.status.phase] ?? "var(--unc-muted)",
                  }}
                >
                  {run.status.phase}
                  {run.status.stage && (
                    <span style={{ marginLeft: "4px", opacity: 0.6, fontSize: "10px" }}>
                      ({run.status.stage})
                    </span>
                  )}
                </td>
                <td
                  style={{
                    padding: "0 8px",
                    textAlign: "right",
                    fontSize: "11px",
                    color: "var(--unc-muted)",
                  }}
                >
                  {formatAge(run.createdAt)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
