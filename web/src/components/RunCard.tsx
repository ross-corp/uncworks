import type { AgentRun } from "../types/agent-run";
import { getStatusColors } from "../lib/statusColors";

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function repoName(url: string): string {
  const parts = url.split("/");
  return parts[parts.length - 1] || url;
}

interface RunCardProps {
  run: AgentRun;
  selected?: boolean;
  onClick?: (run: AgentRun) => void;
}

export function RunCard({ run, selected = false, onClick }: RunCardProps) {
  const statusColors = getStatusColors(run.status.phase);
  const isRunning = run.status.phase === "running";
  const repos = run.spec.repos;
  const repoLabel = repos.length > 0 ? repos.map((r) => repoName(r.url)).join(", ") : "";

  return (
    <div
      data-testid={`run-card-${run.id}`}
      onClick={() => onClick?.(run)}
      className="cursor-pointer transition-colors"
      style={{
        borderLeftWidth: selected ? "3px" : "0px",
        borderLeftColor: selected ? "var(--color-semantic-accent)" : "transparent",
        border: "1px solid var(--color-border-subtle)",
        marginBottom: "8px",
        backgroundColor: selected ? "var(--color-bg-elevated)" : "var(--color-bg-surface)",
      }}
      onMouseEnter={(e) => {
        if (!selected) {
          e.currentTarget.style.backgroundColor = "var(--color-bg-elevated)";
          e.currentTarget.style.borderColor = "var(--color-text-muted)";
        }
      }}
      onMouseLeave={(e) => {
        if (!selected) {
          e.currentTarget.style.backgroundColor = "var(--color-bg-surface)";
          e.currentTarget.style.borderColor = "var(--color-border-subtle)";
        }
      }}
    >
      <div className="flex items-start gap-3 px-4 py-3">
        {/* Status dot */}
        <div
          className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
          style={{
            backgroundColor: statusColors.color,
            animation: isRunning ? "status-pulse 2s ease-in-out infinite" : undefined,
          }}
        />

        {/* Content */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span
              className="truncate font-semibold text-sm"
              style={{ color: "var(--color-text-primary)" }}
            >
              {run.name}
            </span>
            <span
              className="shrink-0 text-xs"
              style={{ color: "var(--color-text-muted)" }}
            >
              {timeAgo(run.createdAt)}
            </span>
          </div>

          {repoLabel && (
            <p
              className="mt-0.5 truncate text-xs"
              style={{ color: "var(--color-text-muted)" }}
            >
              {repoLabel}
            </p>
          )}

          {run.spec.prompt && (
            <p
              className="mt-1 truncate text-xs"
              style={{ color: "var(--color-text-secondary)" }}
            >
              {run.spec.prompt}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
