import { useState, useEffect, useRef } from "react";
import type { GraphNode } from "../types/graph";
import type { GraphEdge } from "../types/graph";
import type { SpanDiff } from "../types/agent-run";
import DiffViewer from "./DiffViewer";

// ============================================================
// CompletionSummary — shown when all agents reach terminal phases.
// Displays status banner with typing animation, agent results
// table, aggregated diffs, and duration breakdown.
// ============================================================

const AGENT_TYPE_LABELS: Record<string, string> = {
  spec: "SPEC",
  senior: "SENIOR",
  junior: "JUNIOR",
};

function formatDuration(startedAt: string, completedAt: string): string {
  if (!startedAt) return "--";
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const secs = Math.floor((end - start) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ${secs % 60}s`;
  return `${Math.floor(mins / 60)}h ${mins % 60}m`;
}

/** TerminalBoot typing animation for the status banner */
function TerminalBootBanner({
  text,
  isFailed,
  onComplete,
}: {
  text: string;
  isFailed: boolean;
  onComplete: () => void;
}) {
  const [displayed, setDisplayed] = useState("");
  const [done, setDone] = useState(false);
  const indexRef = useRef(0);

  useEffect(() => {
    indexRef.current = 0;
    setDisplayed("");
    setDone(false);

    const interval = setInterval(() => {
      indexRef.current++;
      if (indexRef.current <= text.length) {
        setDisplayed(text.slice(0, indexRef.current));
      } else {
        clearInterval(interval);
        setDone(true);
        onComplete();
      }
    }, 40);

    return () => clearInterval(interval);
  }, [text, onComplete]);

  return (
    <div
      className="px-6 py-4 text-center"
      style={{
        background: "var(--muthr-bg)",
        fontFamily: "var(--muthr-font)",
      }}
    >
      <span
        className="text-lg uppercase tracking-[0.3em]"
        style={{
          color: isFailed ? "var(--muthr-amber)" : "var(--muthr-green)",
          textShadow: isFailed
            ? "0 0 12px rgba(255, 102, 0, 0.6)"
            : "0 0 12px rgba(0, 255, 65, 0.6)",
        }}
      >
        {displayed}
        {!done && <span className="muthr-cursor">_</span>}
      </span>
    </div>
  );
}

/** Agent results table row */
function AgentRow({ node }: { node: GraphNode }) {
  const isFailed = node.phase === "failed";
  const color = isFailed ? "var(--muthr-amber)" : "var(--muthr-green)";

  return (
    <tr style={{ borderBottom: "1px solid var(--muthr-dim-green)" }}>
      <td className="px-3 py-2 text-[11px] truncate max-w-[200px]" style={{ color }}>
        {node.runId}
      </td>
      <td className="px-3 py-2 text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
        {AGENT_TYPE_LABELS[node.agentType] ?? node.agentType}
      </td>
      <td className="px-3 py-2 text-[10px] uppercase tracking-widest" style={{ color }}>
        {node.phase}
      </td>
      <td className="px-3 py-2 text-[10px]" style={{ color: "var(--muthr-dim-green)" }}>
        {formatDuration(node.startedAt, node.completedAt)}
      </td>
      <td className="px-3 py-2 text-[10px]" style={{ color: "var(--muthr-dim-green)" }}>
        {node.filesChanged}
      </td>
    </tr>
  );
}

/** Duration breakdown bar chart */
function DurationBreakdown({ nodes }: { nodes: GraphNode[] }) {
  const withTimes = nodes.filter((n) => n.startedAt);
  if (withTimes.length === 0) return null;

  const earliest = Math.min(...withTimes.map((n) => new Date(n.startedAt).getTime()));
  const latest = Math.max(
    ...withTimes.map((n) => {
      const end = n.completedAt ? new Date(n.completedAt).getTime() : Date.now();
      return end;
    })
  );
  const totalRange = latest - earliest || 1;

  return (
    <div className="flex flex-col gap-1">
      {withTimes.map((node) => {
        const start = new Date(node.startedAt).getTime();
        const end = node.completedAt ? new Date(node.completedAt).getTime() : Date.now();
        const leftPercent = ((start - earliest) / totalRange) * 100;
        const widthPercent = Math.max(((end - start) / totalRange) * 100, 1);
        const isFailed = node.phase === "failed";

        return (
          <div key={node.runId} className="flex items-center gap-2">
            <span
              className="text-[9px] uppercase tracking-widest w-16 truncate text-right flex-shrink-0"
              style={{ color: "var(--muthr-dim-green)" }}
            >
              {AGENT_TYPE_LABELS[node.agentType]}
            </span>
            <div className="flex-1 h-4 relative" style={{ background: "rgba(0, 255, 65, 0.04)" }}>
              <div
                className="absolute h-full"
                style={{
                  left: `${leftPercent}%`,
                  width: `${widthPercent}%`,
                  background: isFailed ? "var(--muthr-amber)" : "var(--muthr-green)",
                  opacity: 0.6,
                }}
              />
            </div>
            <span
              className="text-[9px] w-12 flex-shrink-0"
              style={{ color: "var(--muthr-dim-green)" }}
            >
              {formatDuration(node.startedAt, node.completedAt)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

export default function CompletionSummary({
  nodes,
  onViewGraph,
}: {
  nodes: Map<string, GraphNode>;
  edges: readonly GraphEdge[];
  onViewGraph: () => void;
}) {
  const [bannerDone, setBannerDone] = useState(false);
  const [selectedDiff, setSelectedDiff] = useState<SpanDiff | null>(null);

  const nodeList = Array.from(nodes.values());
  const hasFailed = nodeList.some((n) => n.phase === "failed");
  const bannerText = hasFailed
    ? "SPEC COMPLETE \u2014 FAILURES DETECTED"
    : "SPEC COMPLETE \u2014 ALL SYSTEMS NOMINAL";

  // Sort: seniors first, then by start time
  const sortedNodes = [...nodeList].sort((a, b) => {
    const typeOrder: Record<string, number> = { spec: 0, senior: 1, junior: 2 };
    const ta = typeOrder[a.agentType] ?? 3;
    const tb = typeOrder[b.agentType] ?? 3;
    if (ta !== tb) return ta - tb;
    return new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime();
  });

  // Compute summary numbers
  const totalFilesChanged = nodeList.reduce((sum, n) => sum + n.filesChanged, 0);
  const totalLinesAdded = nodeList.reduce((sum, n) => sum + n.linesAdded, 0);
  const totalLinesRemoved = nodeList.reduce((sum, n) => sum + n.linesRemoved, 0);
  const wallClockStart = Math.min(...nodeList.filter((n) => n.startedAt).map((n) => new Date(n.startedAt).getTime()));
  const wallClockEnd = Math.max(
    ...nodeList
      .filter((n) => n.completedAt)
      .map((n) => new Date(n.completedAt).getTime())
  );
  const wallClockSecs = Math.floor((wallClockEnd - wallClockStart) / 1000);
  const totalAgentSecs = nodeList.reduce((sum, n) => {
    if (!n.startedAt) return sum;
    const s = new Date(n.startedAt).getTime();
    const e = n.completedAt ? new Date(n.completedAt).getTime() : Date.now();
    return sum + Math.floor((e - s) / 1000);
  }, 0);

  const handleBannerComplete = () => {
    setBannerDone(true);
  };

  return (
    <div
      className="flex flex-col h-full overflow-y-auto muthr-scanlines"
      style={{
        background: "var(--muthr-bg)",
        fontFamily: "var(--muthr-font)",
      }}
    >
      {/* Status banner with typing animation */}
      <TerminalBootBanner
        text={bannerText}
        isFailed={hasFailed}
        onComplete={handleBannerComplete}
      />

      {/* Content — fades in after banner */}
      <div
        className={bannerDone ? "muthr-fade-in" : "opacity-0"}
        style={{ animationDelay: "100ms" }}
      >
        {/* Summary numbers */}
        <div
          className="grid grid-cols-4 gap-4 px-6 py-4 border-b"
          style={{ borderColor: "var(--muthr-dim-green)" }}
        >
          <SummaryNumber label="Wall Clock" value={`${wallClockSecs}s`} />
          <SummaryNumber label="Agent Time" value={`${totalAgentSecs}s`} />
          <SummaryNumber label="Files Changed" value={`${totalFilesChanged}`} />
          <SummaryNumber
            label="Lines"
            value={`+${totalLinesAdded} / -${totalLinesRemoved}`}
          />
        </div>

        {/* Agent results table */}
        <div
          className={`px-6 py-4 border-b ${bannerDone ? "muthr-fade-in" : "opacity-0"}`}
          style={{ borderColor: "var(--muthr-dim-green)", animationDelay: "300ms" }}
        >
          <h3
            className="text-[10px] uppercase tracking-widest mb-2"
            style={{ color: "var(--muthr-dim-green)" }}
          >
            Agent Results
          </h3>
          <table className="w-full" style={{ borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--muthr-dim-green)" }}>
                <th className="px-3 py-1 text-left text-[9px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Run ID
                </th>
                <th className="px-3 py-1 text-left text-[9px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Type
                </th>
                <th className="px-3 py-1 text-left text-[9px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Status
                </th>
                <th className="px-3 py-1 text-left text-[9px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Duration
                </th>
                <th className="px-3 py-1 text-left text-[9px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Files
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedNodes.map((node) => (
                <AgentRow key={node.runId} node={node} />
              ))}
            </tbody>
          </table>
        </div>

        {/* Duration breakdown */}
        <div
          className={`px-6 py-4 border-b ${bannerDone ? "muthr-fade-in" : "opacity-0"}`}
          style={{ borderColor: "var(--muthr-dim-green)", animationDelay: "500ms" }}
        >
          <h3
            className="text-[10px] uppercase tracking-widest mb-2"
            style={{ color: "var(--muthr-dim-green)" }}
          >
            Duration Breakdown
          </h3>
          <DurationBreakdown nodes={sortedNodes} />
        </div>

        {/* View Graph button */}
        <div className="px-6 py-4 flex justify-center">
          <button
            onClick={onViewGraph}
            className="px-4 py-2 text-[10px] uppercase tracking-widest border transition-colors hover:border-[var(--muthr-green)]"
            style={{
              borderColor: "var(--muthr-dim-green)",
              color: "var(--muthr-green)",
              background: "transparent",
              fontFamily: "var(--muthr-font)",
            }}
          >
            View Graph
          </button>
        </div>
      </div>

      {/* Diff modal */}
      {selectedDiff && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/80"
            onClick={() => setSelectedDiff(null)}
          />
          <div className="relative w-[80vw] h-[80vh]">
            <button
              onClick={() => setSelectedDiff(null)}
              className="absolute top-2 right-2 z-10 px-2 py-1 text-lg"
              style={{ color: "var(--muthr-dim-green)" }}
            >
              &times;
            </button>
            <DiffViewer diff={selectedDiff} />
          </div>
        </div>
      )}
    </div>
  );
}

function SummaryNumber({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <span
        className="text-[10px] uppercase tracking-widest"
        style={{ color: "var(--muthr-dim-green)" }}
      >
        {label}
      </span>
      <span
        className="text-sm muthr-text-glow"
        style={{ color: "var(--muthr-green)" }}
      >
        {value}
      </span>
    </div>
  );
}
