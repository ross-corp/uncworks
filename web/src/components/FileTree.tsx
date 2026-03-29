import { useState, useCallback, useEffect, useRef } from "react";
import type { FileEntry } from "../hooks/useFiles";
import { useFiles } from "../hooks/useFiles";
import type { AgentRunPhase } from "../types/agent-run";

interface FileTreeNode {
  entry: FileEntry;
  path: string;
  children: FileTreeNode[] | null;
  expanded: boolean;
  loading: boolean;
}

const TERMINAL_PHASES = new Set<AgentRunPhase>(["succeeded", "failed", "cancelled"]);

export default function FileTree({
  runId,
  phase,
  onSelectFile,
}: {
  runId: string;
  phase?: AgentRunPhase;
  onSelectFile: (path: string) => void;
}) {
  const { listDir } = useFiles();
  const [roots, setRoots] = useState<FileTreeNode[] | null>(null);
  const [hydrating, setHydrating] = useState(false);
  const loadingRootRef = useRef(false);
  const rootsRef = useRef<FileTreeNode[] | null>(null);

  // Keep refs in sync so interval callbacks see current values
  useEffect(() => {
    rootsRef.current = roots;
  }, [roots]);

  // Collect the set of expanded directory paths from the current tree
  function collectExpandedPaths(nodes: FileTreeNode[], into?: Set<string>): Set<string> {
    const paths = into ?? new Set<string>();
    for (const node of nodes) {
      if (node.expanded) {
        paths.add(node.path);
      }
      if (node.children) {
        collectExpandedPaths(node.children, paths);
      }
    }
    return paths;
  }

  // Build new root nodes from entries, preserving expanded state for dirs
  // that were previously expanded. Also auto-expands single-child directories.
  function buildRootNodes(
    entries: FileEntry[],
    expandedPaths: Set<string>
  ): FileTreeNode[] {
    const nodes = entries.map((entry) => {
      const path = `/workspace/${entry.name}`;
      const wasExpanded = expandedPaths.has(path);
      const prev = rootsRef.current?.find((n) => n.path === path);
      return {
        entry,
        path,
        // Preserve children if the directory was already loaded
        children: prev?.children ?? null,
        expanded: wasExpanded,
        loading: false,
      };
    });

    // Auto-expand: if there is exactly one child and it is a directory,
    // expand it (only on initial load, i.e. when expandedPaths is empty)
    if (expandedPaths.size === 0 && nodes.length === 1 && nodes[0].entry.type === "directory") {
      nodes[0].expanded = true;
    }

    return nodes;
  }

  // Load root directory entries. Returns the entries on success, or null on failure.
  const loadRoot = useCallback(
    async ({ isRetry }: { isRetry?: boolean } = {}): Promise<FileTreeNode[] | null> => {
      if (loadingRootRef.current) return null;
      loadingRootRef.current = true;

      try {
        const result = await listDir(runId, "/workspace");
        const expandedPaths = rootsRef.current
          ? collectExpandedPaths(rootsRef.current)
          : new Set<string>();
        const newRoots = buildRootNodes(result.entries, expandedPaths);

        // If auto-expanding a single child dir, eagerly load its children
        if (
          expandedPaths.size === 0 &&
          newRoots.length === 1 &&
          newRoots[0].expanded &&
          newRoots[0].children === null
        ) {
          try {
            const childResult = await listDir(runId, newRoots[0].path);
            newRoots[0].children = childResult.entries.map((entry) => ({
              entry,
              path: `${newRoots[0].path}/${entry.name}`,
              children: null,
              expanded: false,
              loading: false,
            }));
            // Recurse: if the auto-expanded dir also has a single child dir, expand that too
            let current = newRoots[0];
            while (
              current.children &&
              current.children.length === 1 &&
              current.children[0].entry.type === "directory"
            ) {
              current.children[0].expanded = true;
              try {
                const deeper = await listDir(runId, current.children[0].path);
                current.children[0].children = deeper.entries.map((entry) => ({
                  entry,
                  path: `${current.children![0].path}/${entry.name}`,
                  children: null,
                  expanded: false,
                  loading: false,
                }));
                current = current.children[0];
              } catch {
                break;
              }
            }
          } catch {
            // Fine -- the dir just won't be pre-loaded
          }
        }

        setRoots(newRoots);
        setHydrating(false);
        return newRoots;
      } catch {
        // If we have never loaded successfully, enter hydrating state
        if (rootsRef.current === null && !isRetry) {
          setHydrating(true);
        }
        return null;
      } finally {
        loadingRootRef.current = false;
      }
    },
    // listDir is stable (from useFiles) and runId is the only external dep
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [runId, listDir]
  );

  // Initial load + hydration retry loop
  useEffect(() => {
    let cancelled = false;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    async function init() {
      const result = await loadRoot();
      if (cancelled) return;
      if (result === null) {
        // Failed -- start hydration retry loop (every 3s)
        scheduleRetry();
      }
    }

    function scheduleRetry() {
      retryTimer = setTimeout(async () => {
        if (cancelled) return;
        const result = await loadRoot({ isRetry: true });
        if (cancelled) return;
        if (result === null) {
          scheduleRetry();
        }
      }, 3000);
    }

    init();

    return () => {
      cancelled = true;
      if (retryTimer !== null) clearTimeout(retryTimer);
    };
  }, [loadRoot]);

  // Auto-refresh: poll root directory every 5 seconds to pick up new files.
  // Skip polling once the run has reached a terminal phase — the workspace
  // won't change after succeeded/failed/cancelled.
  const isTerminalPhase = phase ? TERMINAL_PHASES.has(phase) : false;
  useEffect(() => {
    if (isTerminalPhase) return;
    const interval = setInterval(() => {
      // Only refresh if we have already loaded successfully at least once
      if (rootsRef.current !== null) {
        loadRoot();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [loadRoot, isTerminalPhase]);

  async function toggleDir(node: FileTreeNode, updateFn: (updated: FileTreeNode) => void) {
    if (node.expanded) {
      updateFn({ ...node, expanded: false });
      return;
    }

    if (node.children !== null) {
      updateFn({ ...node, expanded: true });
      return;
    }

    updateFn({ ...node, loading: true });
    try {
      const result = await listDir(runId, node.path);
      const children = result.entries.map((entry) => ({
        entry,
        path: `${node.path}/${entry.name}`,
        children: null,
        expanded: false,
        loading: false,
      }));
      updateFn({ ...node, children, expanded: true, loading: false });
    } catch {
      updateFn({ ...node, loading: false });
    }
  }

  function updateNodeInList(
    list: FileTreeNode[],
    targetPath: string,
    updated: FileTreeNode
  ): FileTreeNode[] {
    return list.map((node) => {
      if (node.path === targetPath) return updated;
      if (node.children) {
        return {
          ...node,
          children: updateNodeInList(node.children, targetPath, updated),
        };
      }
      return node;
    });
  }

  function handleClick(node: FileTreeNode) {
    if (node.entry.type === "directory") {
      toggleDir(node, (updated) => {
        setRoots((prev) =>
          prev ? updateNodeInList(prev, node.path, updated) : prev
        );
      });
    } else {
      onSelectFile(node.path);
    }
  }

  if (hydrating) {
    // For completed runs the workspace pod is scaled down.
    // Reading directly from the PVC host path requires the apiserver to have
    // access to the node filesystem. If that fails, show a more accurate message
    // and guide the user to start a debug session instead.
    if (phase && TERMINAL_PHASES.has(phase)) {
      return (
        <div className="p-3 text-sm text-muted-foreground/60 space-y-2">
          <div>Workspace unavailable.</div>
          <div className="text-xs">
            The run has {phase}. Use the <span className="font-medium text-foreground">Shell</span> tab
            to start a debug session and browse files interactively.
          </div>
        </div>
      );
    }
    return (
      <div className="p-3 text-sm text-muted-foreground/60">
        Workspace hydrating...
      </div>
    );
  }

  if (roots === null) {
    return (
      <div className="p-3 text-sm text-muted-foreground/60">Loading...</div>
    );
  }

  return (
    <div data-testid="file-tree" className="overflow-y-auto overscroll-none font-mono text-xs">
      {roots.map((node) => (
        <TreeNodeRow
          key={node.path}
          node={node}
          depth={0}
          onClick={handleClick}
        />
      ))}
    </div>
  );
}

function TreeNodeRow({
  node,
  depth,
  onClick,
}: {
  node: FileTreeNode;
  depth: number;
  onClick: (node: FileTreeNode) => void;
}) {
  const isDir = node.entry.type === "directory";
  const icon = isDir
    ? node.expanded
      ? "\u25BE"
      : "\u25B8"
    : "\u00A0\u00A0";
  const folderIcon = isDir ? "\uD83D\uDCC1" : "\uD83D\uDCC4";

  return (
    <>
      <button
        data-testid={`file-entry-${node.entry.name}`}
        className="flex w-full items-center gap-1 px-2 py-0.5 text-left text-muted-foreground hover:bg-muted transition-colors"
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => onClick(node)}
      >
        <span className="w-3 text-center text-muted-foreground/60">{icon}</span>
        <span>{folderIcon}</span>
        <span className="truncate">{node.entry.name}</span>
        {!isDir && node.entry.size > 0 && (
          <span className="ml-auto text-muted-foreground/60">
            {formatSize(node.entry.size)}
          </span>
        )}
        {node.loading && (
          <span className="ml-auto text-muted-foreground/60">...</span>
        )}
      </button>
      {node.expanded && node.children?.map((child) => (
        <TreeNodeRow
          key={child.path}
          node={child}
          depth={depth + 1}
          onClick={onClick}
        />
      ))}
    </>
  );
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}K`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}M`;
}
