import { useGraphStore, graphStore } from "../stores/graph-store";
import { useTraceSpans } from "../hooks/useTraceSpans";
import { PhaseBadge } from "./StatusBadge";
import TraceTimeline from "./TraceTimeline";
import DiffViewer from "./DiffViewer";
import { useState } from "react";
import type { TraceSpan } from "../types/agent-run";

// ============================================================
// DetailPanel — slides in from the right (40% width) when a
// graph node is selected. Shows run metadata, trace timeline,
// and diff viewer.
// ============================================================

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

const AGENT_TYPE_LABELS: Record<string, string> = {
  spec: "SPEC",
  senior: "SENIOR AGENT",
  junior: "JUNIOR AGENT",
};

export default function DetailPanel() {
  const { nodes, selectedRunId } = useGraphStore();
  const { spans, loading } = useTraceSpans(selectedRunId);
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | null>(null);

  if (!selectedRunId) return null;

  const node = nodes.get(selectedRunId);
  if (!node) return null;

  function handleClose() {
    graphStore.setSelectedRunId(null);
  }

  return (
    <div
      className="flex flex-col h-full border-l overflow-hidden"
      style={{
        width: "40%",
        minWidth: 360,
        borderColor: "var(--muthr-dim-green)",
        background: "var(--muthr-bg)",
        fontFamily: "var(--muthr-font)",
        animation: "muthr-node-enter 200ms ease-out",
      }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-3 border-b flex-shrink-0"
        style={{ borderColor: "var(--muthr-dim-green)" }}
      >
        <div className="flex items-center gap-3 min-w-0">
          <span
            className="text-[10px] uppercase tracking-widest"
            style={{ color: "var(--muthr-green)" }}
          >
            {AGENT_TYPE_LABELS[node.agentType] ?? node.agentType}
          </span>
          <PhaseBadge phase={node.phase} />
        </div>
        <button
          onClick={handleClose}
          className="text-lg leading-none px-2 transition-colors hover:opacity-80"
          style={{ color: "var(--muthr-dim-green)" }}
          aria-label="Close panel"
        >
          &times;
        </button>
      </div>

      {/* Run metadata */}
      <div
        className="flex flex-col gap-1 px-4 py-3 border-b flex-shrink-0"
        style={{ borderColor: "var(--muthr-dim-green)" }}
      >
        <div className="flex items-baseline justify-between gap-4">
          <span className="text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
            Run ID
          </span>
          <span className="text-[11px] truncate" style={{ color: "var(--muthr-green)" }}>
            {node.runId}
          </span>
        </div>
        <div className="flex items-baseline justify-between gap-4">
          <span className="text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
            Duration
          </span>
          <span className="text-[11px]" style={{ color: "var(--muthr-green)" }}>
            {formatDuration(node.startedAt, node.completedAt)}
          </span>
        </div>
        {node.currentActivity && (
          <div className="flex items-baseline justify-between gap-4">
            <span className="text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
              Activity
            </span>
            <span className="text-[11px] truncate" style={{ color: "var(--muthr-green)" }}>
              {node.currentActivity}
            </span>
          </div>
        )}
      </div>

      {/* Trace timeline */}
      <div className="flex-shrink-0 border-b overflow-y-auto max-h-[40%]" style={{ borderColor: "var(--muthr-dim-green)" }}>
        {loading ? (
          <div
            className="flex items-center justify-center py-8"
            style={{ background: "var(--muthr-bg)" }}
          >
            <span className="text-[11px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
              Loading traces...
            </span>
          </div>
        ) : (
          <TraceTimeline
            spans={spans}
            selectedSpanId={selectedSpan?.id}
            onSelectSpan={setSelectedSpan}
            runId={node.runId}
            agentType={node.agentType}
          />
        )}
      </div>

      {/* Span detail / diff viewer */}
      <div className="flex-1 overflow-y-auto">
        {selectedSpan === null && (
          <div
            className="flex items-center justify-center h-full"
            style={{ background: "var(--muthr-bg)" }}
          >
            <span className="text-[11px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
              Select a span to view details
            </span>
          </div>
        )}
        {selectedSpan !== null && selectedSpan.hasDiff && selectedSpan.diff && (
          <DiffViewer diff={selectedSpan.diff} />
        )}
        {selectedSpan !== null && !selectedSpan.hasDiff && (
          <div className="p-4">
            <div className="flex flex-col gap-2">
              <div className="flex items-baseline justify-between gap-4">
                <span className="text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Name
                </span>
                <span className="text-[11px]" style={{ color: "var(--muthr-green)" }}>
                  {selectedSpan.name}
                </span>
              </div>
              <div className="flex items-baseline justify-between gap-4">
                <span className="text-[10px] uppercase tracking-widest" style={{ color: "var(--muthr-dim-green)" }}>
                  Type
                </span>
                <span className="text-[11px] uppercase" style={{ color: "var(--muthr-green)" }}>
                  {selectedSpan.type}
                </span>
              </div>
              {selectedSpan.metadata && Object.keys(selectedSpan.metadata).length > 0 && (
                <div>
                  <span className="text-[10px] uppercase tracking-widest block mb-1" style={{ color: "var(--muthr-dim-green)" }}>
                    Metadata
                  </span>
                  <pre
                    className="p-2 text-[10px] overflow-x-auto whitespace-pre-wrap"
                    style={{
                      background: "rgba(0, 255, 65, 0.04)",
                      color: "var(--muthr-dim-green)",
                      border: "1px solid var(--muthr-dim-green)",
                    }}
                  >
                    {JSON.stringify(selectedSpan.metadata, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
