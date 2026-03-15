import type { GraphNode } from "../types/graph";
import { isTerminalPhase } from "../types/graph";
import { PhaseBadge } from "./StatusBadge";
import { PhosphorPulse, RadarSweep, DataStreamBackground } from "./LiveIndicators";

const AGENT_TYPE_LABELS: Record<GraphNode["agentType"], string> = {
  spec: "SPEC",
  senior: "SENIOR",
  junior: "JUNIOR",
};

function formatElapsed(startedAt: string, completedAt: string): string {
  if (!startedAt) return "--";
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const secs = Math.floor((end - start) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ${secs % 60}s`;
  return `${Math.floor(mins / 60)}h ${mins % 60}m`;
}

function truncate(text: string, maxLen: number): string {
  if (text.length <= maxLen) return text;
  return text.slice(0, maxLen - 1) + "\u2026";
}

export default function OrchestrationNode({
  node,
  isSelected,
  isRoot,
  hasActiveChildren,
  isReceivingOutput,
  onClick,
}: {
  node: GraphNode;
  isSelected: boolean;
  isRoot: boolean;
  hasActiveChildren: boolean;
  isReceivingOutput: boolean;
  onClick: () => void;
}) {
  const isRunning = node.phase === "running";
  const isFailed = node.phase === "failed";
  const isTerminal = isTerminalPhase(node.phase);

  // Border color based on state
  let borderClass = "border-[var(--muthr-dim-green)]";
  if (isSelected) {
    borderClass = "border-[var(--muthr-green)]";
  } else if (isFailed) {
    borderClass = "border-[var(--muthr-amber)]";
  } else if (isRunning) {
    borderClass = "border-[var(--muthr-green)]";
  }

  // Background based on state
  let bgClass = "bg-[var(--muthr-bg)]";
  if (isTerminal && !isFailed) {
    bgClass = "bg-[var(--muthr-dim-green)]";
  }

  const nodeContent = (
    <button
      onClick={onClick}
      className={`
        relative flex flex-col items-start gap-1 px-3 py-2 border transition-all duration-200
        min-w-[160px] max-w-[220px] text-left muthr-node-enter
        ${borderClass} ${bgClass}
        ${isSelected ? "muthr-glow" : ""}
        hover:border-[var(--muthr-green)] cursor-pointer
      `}
      style={{ fontFamily: "var(--muthr-font)" }}
    >
      {/* Radar sweep behind root node */}
      {isRoot && <RadarSweep active={hasActiveChildren} />}

      {/* Agent type label */}
      <div className="flex items-center justify-between w-full relative z-10">
        <span
          className="text-[10px] tracking-widest uppercase"
          style={{ color: "var(--muthr-green)" }}
        >
          {AGENT_TYPE_LABELS[node.agentType]}
        </span>
        <PhaseBadge phase={node.phase} />
      </div>

      {/* Run ID (truncated) */}
      <span className="text-xs text-muted-foreground font-mono truncate w-full relative z-10">
        {truncate(node.runId, 20)}
      </span>

      {/* Current activity (running only) */}
      {isRunning && node.currentActivity && (
        <span
          className="text-[10px] uppercase tracking-wider truncate w-full relative z-10"
          style={{ color: "var(--muthr-green)" }}
        >
          {truncate(node.currentActivity, 30)}
        </span>
      )}

      {/* Elapsed time */}
      <span className="text-[10px] text-muted-foreground/60 relative z-10">
        {formatElapsed(node.startedAt, node.completedAt)}
      </span>
    </button>
  );

  // Wrap with data stream if receiving output
  const withDataStream = (
    <DataStreamBackground active={isReceivingOutput}>
      {nodeContent}
    </DataStreamBackground>
  );

  // Wrap with pulse if running
  if (isRunning) {
    return <PhosphorPulse active>{withDataStream}</PhosphorPulse>;
  }

  // Wrap with amber pulse if failed
  if (isFailed) {
    return <div className="muthr-amber-pulse">{withDataStream}</div>;
  }

  return withDataStream;
}
