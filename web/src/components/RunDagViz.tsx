// RunDagViz.tsx — ReactFlow DAG for agent run parent/child graphs.
import {
  ReactFlow,
  Background,
  Controls,
  MarkerType,
  type Node,
  type Edge,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useNavigate } from "react-router-dom";
import type { RunGraphNode, RunGraphEdge } from "../types/agent-run";

const NODE_W = 180;
const NODE_H = 64;
const COL_GAP = 240;
const ROW_GAP = 90;

const PHASE_STYLE: Record<string, React.CSSProperties> = {
  running: { border: "2px solid #3b82f6", background: "#eff6ff", color: "#1d4ed8" },
  succeeded: { border: "2px solid #22c55e", background: "#f0fdf4", color: "#15803d" },
  failed: { border: "2px solid #ef4444", background: "#fef2f2", color: "#b91c1c" },
  cancelled: { border: "2px solid #9ca3af", background: "#f9fafb", color: "#6b7280" },
  pending: { border: "2px solid #9ca3af", background: "#f9fafb", color: "#6b7280" },
  waiting_for_input: { border: "2px solid #f59e0b", background: "#fffbeb", color: "#d97706" },
};

interface RunNodeData extends Record<string, unknown> {
  name: string;
  phase: string;
  role: string;
  isCurrent: boolean;
}

function RunNode({ data }: NodeProps<Node<RunNodeData>>) {
  const style = PHASE_STYLE[data.phase] ?? PHASE_STYLE.pending;
  const outerStyle: React.CSSProperties = {
    ...style,
    width: NODE_W,
    height: NODE_H,
    borderRadius: 8,
    padding: "6px 10px",
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    cursor: data.isCurrent ? "default" : "pointer",
    boxShadow: data.isCurrent ? "0 0 0 3px rgba(59,130,246,0.5)" : undefined,
    fontWeight: data.isCurrent ? 700 : 400,
  };

  return (
    <div style={outerStyle}>
      <div style={{ fontSize: 11, fontFamily: "monospace", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {data.name}
      </div>
      {data.role && (
        <div style={{ fontSize: 9, opacity: 0.6, textTransform: "uppercase", letterSpacing: "0.05em", marginTop: 2 }}>
          {data.role}
        </div>
      )}
      <div style={{ fontSize: 9, textTransform: "uppercase", opacity: 0.7, marginTop: 2 }}>
        {data.phase.replace(/_/g, " ")}
      </div>
    </div>
  );
}

const nodeTypes = { run: RunNode };

interface RunDagVizProps {
  nodes: RunGraphNode[];
  edges: RunGraphEdge[];
  currentRunName: string;
}

export default function RunDagViz({ nodes, edges, currentRunName }: RunDagVizProps) {
  const navigate = useNavigate();
  // Compute depth (column) for each node based on edges
  const depthMap = new Map<string, number>();
  function depth(name: string): number {
    if (depthMap.has(name)) return depthMap.get(name)!;
    const parents = edges.filter((e) => e.child === name).map((e) => e.parent);
    const d = parents.length === 0 ? 0 : Math.max(...parents.map(depth)) + 1;
    depthMap.set(name, d);
    return d;
  }
  for (const n of nodes) depth(n.name);

  // Group nodes by depth column for row placement
  const byDepth = new Map<number, RunGraphNode[]>();
  for (const n of nodes) {
    const d = depthMap.get(n.name) ?? 0;
    if (!byDepth.has(d)) byDepth.set(d, []);
    byDepth.get(d)!.push(n);
  }

  const flowNodes: Node[] = nodes.map((n) => {
    const col = depthMap.get(n.name) ?? 0;
    const colNodes = byDepth.get(col) ?? [];
    const row = colNodes.indexOf(n);
    return {
      id: n.name,
      type: "run",
      position: { x: col * COL_GAP, y: row * ROW_GAP },
      data: {
        name: n.name,
        phase: n.phase,
        role: n.role ?? "",
        isCurrent: n.name === currentRunName,
      },
    };
  });

  const flowEdges: Edge[] = edges.map((e) => {
    const targetNode = nodes.find((n) => n.name === e.child);
    return {
      id: `${e.parent}->${e.child}`,
      source: e.parent,
      target: e.child,
      type: "smoothstep",
      animated: targetNode?.phase === "running",
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
      style: { strokeWidth: 1.5 },
    };
  });

  function onNodeClick(_: React.MouseEvent, node: Node) {
    const d = node.data as RunNodeData;
    if (!d.isCurrent && d.name) navigate(`/run/${d.name}`);
  }

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
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
