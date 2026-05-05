import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

export interface ChainStepStatus {
  name: string;
  phase: string;
  dependsOn: string[];
  runId?: string;
  elapsedSeconds?: number;
}

// Node dimensions
const NODE_W = 160;
const NODE_H = 60;
const COL_GAP = 220;
const ROW_GAP = 100;

function computeDepths(steps: ChainStepStatus[]): Map<string, number> {
  const depthMap = new Map<string, number>();

  function depth(name: string): number {
    if (depthMap.has(name)) return depthMap.get(name)!;
    const step = steps.find((s) => s.name === name);
    if (!step || step.dependsOn.length === 0) {
      depthMap.set(name, 0);
      return 0;
    }
    const d = Math.max(...step.dependsOn.map(depth)) + 1;
    depthMap.set(name, d);
    return d;
  }

  for (const s of steps) depth(s.name);
  return depthMap;
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

const PHASE_STYLE: Record<string, React.CSSProperties> = {
  running: { border: "2px solid #3b82f6", background: "#eff6ff", color: "#1d4ed8" },
  succeeded: { border: "2px solid #22c55e", background: "#f0fdf4", color: "#15803d" },
  failed: { border: "2px solid #ef4444", background: "#fef2f2", color: "#b91c1c" },
  cancelled: { border: "2px solid #ef4444", background: "#fef2f2", color: "#b91c1c" },
  pending: { border: "2px solid #9ca3af", background: "#f9fafb", color: "#6b7280" },
  skipped: { border: "2px solid #eab308", background: "#fefce8", color: "#a16207" },
};

interface StepNodeData extends Record<string, unknown> {
  label: string;
  phase: string;
  runId?: string;
  elapsedSeconds?: number;
}

function StepNode({ data }: NodeProps<Node<StepNodeData>>) {
  const style = PHASE_STYLE[data.phase] || PHASE_STYLE.pending;
  const showDuration =
    (data.phase === "succeeded" || data.phase === "failed") &&
    data.elapsedSeconds !== undefined;

  return (
    <div
      style={{
        ...style,
        width: NODE_W,
        height: NODE_H,
        borderRadius: 8,
        padding: "6px 10px",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        fontSize: 12,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <span style={{ fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", maxWidth: 110 }}>
          {data.label}
        </span>
        <span style={{ fontSize: 9, textTransform: "uppercase", letterSpacing: "0.05em", opacity: 0.7 }}>
          {data.phase}
        </span>
      </div>
      {showDuration && (
        <div style={{ fontSize: 10, opacity: 0.7, marginTop: 2 }}>
          {formatDuration(data.elapsedSeconds!)}
        </div>
      )}
      {!showDuration && data.runId && (
        <div style={{ fontSize: 9, opacity: 0.5, fontFamily: "monospace", marginTop: 2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {data.runId}
        </div>
      )}
    </div>
  );
}

const nodeTypes = { step: StepNode };

interface ChainDagVizProps {
  steps: ChainStepStatus[];
}

export default function ChainDagViz({ steps }: ChainDagVizProps) {
  const navigate = useNavigate();

  const depthMap = computeDepths(steps);

  // Group steps by depth column
  const byDepth = new Map<number, ChainStepStatus[]>();
  for (const s of steps) {
    const d = depthMap.get(s.name) ?? 0;
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(s);
  }

  // Build nodes
  const nodes: Node[] = steps.map((s) => {
    const col = depthMap.get(s.name) ?? 0;
    const colSteps = byDepth.get(col) ?? [];
    const row = colSteps.indexOf(s);
    return {
      id: s.name,
      type: "step",
      position: { x: col * COL_GAP, y: row * ROW_GAP },
      data: {
        label: s.name,
        phase: s.phase,
        runId: s.runId,
        elapsedSeconds: s.elapsedSeconds,
      },
    };
  });

  // Build edges
  const edges: Edge[] = [];
  for (const s of steps) {
    for (const dep of s.dependsOn) {
      edges.push({
        id: `${dep}->${s.name}`,
        source: dep,
        target: s.name,
        animated: s.phase === "running",
      });
    }
  }

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const data = node.data as StepNodeData;
      if (data.runId) navigate(`/run/${data.runId}`);
    },
    [navigate]
  );

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={onNodeClick}
        fitView
        fitViewOptions={{ padding: 0.3 }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
