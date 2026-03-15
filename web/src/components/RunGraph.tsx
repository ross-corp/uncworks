import type { ReactElement } from "react";
import type { RunGraphNode, RunGraphEdge, AgentRunPhase } from "../types/agent-run";

interface RunGraphProps {
  nodes: RunGraphNode[];
  edges: RunGraphEdge[];
  onSelectNode?: (name: string) => void;
}

const phaseColors: Record<AgentRunPhase, string> = {
  pending: "bg-gray-400",
  running: "bg-blue-500",
  waiting_for_input: "bg-yellow-500",
  succeeded: "bg-green-500",
  failed: "bg-red-500",
  cancelled: "bg-orange-500",
};

const phaseLabels: Record<AgentRunPhase, string> = {
  pending: "Pending",
  running: "Running",
  waiting_for_input: "Waiting",
  succeeded: "Succeeded",
  failed: "Failed",
  cancelled: "Cancelled",
};

function formatDuration(startedAt?: string, completedAt?: string): string {
  if (!startedAt) return "";
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const secs = Math.floor((end - start) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ${secs % 60}s`;
  return `${Math.floor(mins / 60)}h ${mins % 60}m`;
}

export default function RunGraph({ nodes, edges, onSelectNode }: RunGraphProps) {
  // Build the tree structure
  const childrenMap = new Map<string, string[]>();
  const childSet = new Set<string>();
  for (const edge of edges) {
    const children = childrenMap.get(edge.parent) || [];
    children.push(edge.child);
    childrenMap.set(edge.parent, children);
    childSet.add(edge.child);
  }

  // Find root nodes (no parent)
  const roots = nodes.filter((n) => !childSet.has(n.name));
  const nodeMap = new Map(nodes.map((n) => [n.name, n]));

  // Calculate progress
  const juniorNodes = nodes.filter((n) => n.role === "junior");
  const completedJuniors = juniorNodes.filter(
    (n) => n.phase === "succeeded" || n.phase === "failed" || n.phase === "cancelled"
  );

  function renderNode(name: string, depth: number): ReactElement | null {
    const node = nodeMap.get(name);
    if (!node) return null;
    const children = childrenMap.get(name) || [];
    const colorClass = phaseColors[node.phase] || "bg-gray-400";

    return (
      <div key={name} style={{ marginLeft: depth * 24 }} className="my-1">
        <button
          onClick={() => onSelectNode?.(name)}
          className="flex items-center gap-2 px-3 py-1.5 rounded border border-border bg-card hover:bg-accent/50 transition-colors text-left w-full max-w-md"
          data-testid={`run-graph-node-${name}`}
        >
          <span className={`inline-block w-2 h-2 rounded-full flex-shrink-0 ${colorClass}`} />
          <span className="font-mono text-xs text-foreground truncate">{name}</span>
          <span className="text-xs text-muted-foreground/60 whitespace-nowrap">
            {phaseLabels[node.phase] || node.phase}
          </span>
          {node.startedAt && (
            <span className="text-xs text-muted-foreground/40 whitespace-nowrap">
              {formatDuration(node.startedAt, node.completedAt)}
            </span>
          )}
          <span className="text-xs text-muted-foreground/40 capitalize">{node.role}</span>
        </button>
        {children.map((child) => renderNode(child, depth + 1))}
      </div>
    );
  }

  return (
    <div data-testid="run-graph" className="p-4 space-y-1">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
          Run Graph
        </h3>
        {juniorNodes.length > 0 && (
          <span className="text-xs text-muted-foreground" data-testid="run-graph-progress">
            {completedJuniors.length}/{juniorNodes.length} tasks complete
          </span>
        )}
      </div>
      {roots.map((root) => renderNode(root.name, 0))}
    </div>
  );
}
