import { useState, useMemo } from "react";
import { useOrchestrationGraph } from "../hooks/useOrchestrationGraph";
import { useGraphStore } from "../stores/graph-store";
import { isTerminalPhase } from "../types/graph";
import OrchestrationGraph from "../components/OrchestrationGraph";
import DetailPanel from "../components/DetailPanel";
import CompletionSummary from "../components/CompletionSummary";

// ============================================================
// SpecRunPage — top-level page combining OrchestrationGraph,
// DetailPanel, and CompletionSummary. Mounted at /specs/:specRunId.
// ============================================================

export default function SpecRunPage({ specRunId }: { specRunId: string }) {
  const [forceGraphView, setForceGraphView] = useState(false);

  // Connect to graph data and SSE stream
  useOrchestrationGraph(specRunId);
  const { nodes, edges, selectedRunId } = useGraphStore();

  // Determine if all agents are in terminal phases
  const allTerminal = useMemo(() => {
    if (nodes.size === 0) return false;
    for (const node of nodes.values()) {
      if (!isTerminalPhase(node.phase)) return false;
    }
    return true;
  }, [nodes]);

  const showCompletion = allTerminal && !forceGraphView;
  const hasPanelOpen = selectedRunId !== null;

  return (
    <div
      className="flex flex-col h-full"
      style={{ fontFamily: "var(--muthr-font)" }}
    >
      {/* Breadcrumb header */}
      <div
        className="flex items-center gap-2 px-4 py-2 border-b flex-shrink-0"
        style={{
          borderColor: "var(--muthr-dim-green)",
          background: "var(--muthr-bg)",
        }}
      >
        <span
          className="text-[10px] uppercase tracking-widest cursor-pointer hover:opacity-80 transition-opacity"
          style={{ color: "var(--muthr-dim-green)" }}
        >
          Runs
        </span>
        <span
          className="text-[10px]"
          style={{ color: "var(--muthr-dim-green)" }}
        >
          /
        </span>
        <span
          className="text-[11px] uppercase tracking-wider truncate muthr-text-glow"
          style={{ color: "var(--muthr-green)" }}
        >
          Spec Run {specRunId}
        </span>
      </div>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        {showCompletion ? (
          <CompletionSummary
            nodes={nodes}
            edges={edges}
            onViewGraph={() => setForceGraphView(true)}
          />
        ) : (
          <>
            {/* Graph area — shrinks to 60% when panel is open */}
            <div
              className="flex-1 overflow-hidden transition-all duration-200"
              style={{
                flexBasis: hasPanelOpen ? "60%" : "100%",
                maxWidth: hasPanelOpen ? "60%" : "100%",
              }}
            >
              <OrchestrationGraph />
            </div>

            {/* Detail panel — 40% width when open */}
            {hasPanelOpen && <DetailPanel />}
          </>
        )}
      </div>
    </div>
  );
}
