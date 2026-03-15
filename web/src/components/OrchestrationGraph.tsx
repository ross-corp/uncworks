import { useMemo, useRef, useCallback } from "react";
import type { GraphNode, NodePosition, GraphEdge } from "../types/graph";
import { isTerminalPhase } from "../types/graph";
import { useGraphStore, graphStore } from "../stores/graph-store";
import OrchestrationNode from "./OrchestrationNode";

// Layout constants
const NODE_WIDTH = 180;
const NODE_HEIGHT = 80;
const HORIZONTAL_SPACING = 40;
const VERTICAL_SPACING = 60;

/**
 * layoutTree — recursive top-down tree layout.
 * Returns a map of node ID to {x, y} positions.
 * O(n) complexity.
 */
export function layoutTree(
  nodes: Map<string, GraphNode>,
  edges: GraphEdge[]
): Map<string, NodePosition> {
  const positions = new Map<string, NodePosition>();
  if (nodes.size === 0) return positions;

  // Build children map
  const childrenMap = new Map<string, string[]>();
  for (const [id] of nodes) {
    childrenMap.set(id, []);
  }
  for (const edge of edges) {
    const children = childrenMap.get(edge.parentRunId);
    if (children) {
      children.push(edge.childRunId);
    }
  }

  // Find root nodes (no parent)
  const childIds = new Set(edges.map((e) => e.childRunId));
  const roots: string[] = [];
  for (const [id] of nodes) {
    if (!childIds.has(id)) {
      roots.push(id);
    }
  }

  if (roots.length === 0) {
    // Fallback: use first node as root
    const firstId = nodes.keys().next().value;
    if (firstId !== undefined) {
      roots.push(firstId);
    }
  }

  // Compute subtree widths
  const widthCache = new Map<string, number>();

  function subtreeWidth(nodeId: string): number {
    const cached = widthCache.get(nodeId);
    if (cached !== undefined) return cached;

    const children = childrenMap.get(nodeId) ?? [];
    if (children.length === 0) {
      widthCache.set(nodeId, NODE_WIDTH);
      return NODE_WIDTH;
    }

    let totalWidth = 0;
    for (let i = 0; i < children.length; i++) {
      if (i > 0) totalWidth += HORIZONTAL_SPACING;
      totalWidth += subtreeWidth(children[i]);
    }
    const width = Math.max(NODE_WIDTH, totalWidth);
    widthCache.set(nodeId, width);
    return width;
  }

  // Place nodes recursively
  function placeNode(nodeId: string, x: number, y: number) {
    positions.set(nodeId, { x, y });

    const children = childrenMap.get(nodeId) ?? [];
    if (children.length === 0) return;

    const totalChildrenWidth = children.reduce(
      (acc, childId, i) => acc + subtreeWidth(childId) + (i > 0 ? HORIZONTAL_SPACING : 0),
      0
    );

    let childX = x - totalChildrenWidth / 2;
    const childY = y + NODE_HEIGHT + VERTICAL_SPACING;

    for (const childId of children) {
      const childWidth = subtreeWidth(childId);
      placeNode(childId, childX + childWidth / 2, childY);
      childX += childWidth + HORIZONTAL_SPACING;
    }
  }

  // Layout each root
  if (roots.length === 1) {
    placeNode(roots[0], 0, 0);
  } else {
    let totalWidth = 0;
    for (let i = 0; i < roots.length; i++) {
      if (i > 0) totalWidth += HORIZONTAL_SPACING;
      totalWidth += subtreeWidth(roots[i]);
    }
    let startX = -totalWidth / 2;
    for (const rootId of roots) {
      const w = subtreeWidth(rootId);
      placeNode(rootId, startX + w / 2, 0);
      startX += w + HORIZONTAL_SPACING;
    }
  }

  return positions;
}

/**
 * OrchestrationGraph — renders the tree layout with nodes and edges.
 */
export default function OrchestrationGraph() {
  const { nodes, edges, selectedRunId, reconnecting } = useGraphStore();
  const containerRef = useRef<HTMLDivElement>(null);

  const positions = useMemo(() => layoutTree(nodes, edges), [nodes, edges]);

  const handleNodeClick = useCallback((runId: string) => {
    graphStore.setSelectedRunId(
      selectedRunId === runId ? null : runId
    );
  }, [selectedRunId]);

  const handleFitToView = useCallback(() => {
    containerRef.current?.scrollTo({
      left: 0,
      top: 0,
      behavior: "smooth",
    });
  }, []);

  if (nodes.size === 0) {
    return (
      <div
        className="flex items-center justify-center h-full muthr-scanlines"
        style={{ background: "var(--muthr-bg)", fontFamily: "var(--muthr-font)" }}
      >
        <span className="text-sm text-muted-foreground/60 uppercase tracking-widest">
          Awaiting agent deployment...
        </span>
      </div>
    );
  }

  // Calculate bounds for the SVG/container
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const pos of positions.values()) {
    minX = Math.min(minX, pos.x);
    maxX = Math.max(maxX, pos.x);
    minY = Math.min(minY, pos.y);
    maxY = Math.max(maxY, pos.y);
  }

  const padding = 60;
  const contentWidth = maxX - minX + NODE_WIDTH + padding * 2;
  const contentHeight = maxY - minY + NODE_HEIGHT + padding * 2;
  const offsetX = -minX + padding;
  const offsetY = -minY + padding;

  // Determine which nodes are "active" (non-terminal children exist)
  const hasActiveChildrenSet = new Set<string>();
  for (const edge of edges) {
    const child = nodes.get(edge.childRunId);
    if (child && !isTerminalPhase(child.phase)) {
      hasActiveChildrenSet.add(edge.parentRunId);
    }
  }

  // Root nodes
  const childIds = new Set(edges.map((e) => e.childRunId));

  return (
    <div
      ref={containerRef}
      className="relative h-full overflow-auto muthr-scanlines"
      style={{ background: "var(--muthr-bg)" }}
    >
      {/* Reconnecting banner */}
      {reconnecting && (
        <div
          className="absolute top-2 left-1/2 -translate-x-1/2 z-20 px-3 py-1 text-[10px] uppercase tracking-widest"
          style={{
            background: "var(--muthr-amber)",
            color: "var(--muthr-bg)",
            fontFamily: "var(--muthr-font)",
          }}
        >
          Reconnecting...
        </div>
      )}

      {/* Fit to view button */}
      <button
        onClick={handleFitToView}
        className="absolute top-2 right-2 z-20 px-2 py-1 text-[10px] uppercase tracking-widest border border-[var(--muthr-dim-green)] hover:border-[var(--muthr-green)] transition-colors"
        style={{
          background: "var(--muthr-bg)",
          color: "var(--muthr-green)",
          fontFamily: "var(--muthr-font)",
        }}
      >
        Fit to View
      </button>

      {/* Graph content */}
      <div
        style={{
          position: "relative",
          width: contentWidth,
          height: contentHeight,
          minWidth: "100%",
          minHeight: "100%",
        }}
      >
        {/* SVG edges */}
        <svg
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            width: contentWidth,
            height: contentHeight,
            pointerEvents: "none",
          }}
        >
          {edges.map((edge) => {
            const parentPos = positions.get(edge.parentRunId);
            const childPos = positions.get(edge.childRunId);
            if (!parentPos || !childPos) return null;

            const x1 = parentPos.x + offsetX + NODE_WIDTH / 2;
            const y1 = parentPos.y + offsetY + NODE_HEIGHT;
            const x2 = childPos.x + offsetX + NODE_WIDTH / 2;
            const y2 = childPos.y + offsetY;

            const childNode = nodes.get(edge.childRunId);
            const isActive = childNode && !isTerminalPhase(childNode.phase);

            return (
              <line
                key={`${edge.parentRunId}-${edge.childRunId}`}
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke={isActive ? "var(--muthr-green)" : "var(--muthr-dim-green)"}
                strokeWidth={isActive ? 2 : 1}
                strokeOpacity={isActive ? 0.8 : 0.4}
              />
            );
          })}
        </svg>

        {/* Nodes */}
        {Array.from(nodes.entries()).map(([runId, node]) => {
          const pos = positions.get(runId);
          if (!pos) return null;

          const isRoot = !childIds.has(runId);

          return (
            <div
              key={runId}
              style={{
                position: "absolute",
                left: pos.x + offsetX,
                top: pos.y + offsetY,
                width: NODE_WIDTH,
              }}
            >
              <OrchestrationNode
                node={node}
                isSelected={selectedRunId === runId}
                isRoot={isRoot}
                hasActiveChildren={hasActiveChildrenSet.has(runId)}
                isReceivingOutput={node.phase === "running" && !!node.currentActivity}
                onClick={() => handleNodeClick(runId)}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
