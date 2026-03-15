import { useState, useCallback } from "react";
import type { FileEntry } from "../hooks/useFiles";
import { useFiles } from "../hooks/useFiles";

interface FileTreeNode {
  entry: FileEntry;
  path: string;
  children: FileTreeNode[] | null;
  expanded: boolean;
  loading: boolean;
}

export default function FileTree({
  runId,
  onSelectFile,
}: {
  runId: string;
  onSelectFile: (path: string) => void;
}) {
  const { listDir } = useFiles();
  const [roots, setRoots] = useState<FileTreeNode[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingRoot, setLoadingRoot] = useState(false);

  // Load root directory on first render
  const loadRoot = useCallback(async () => {
    if (roots !== null || loadingRoot) return;
    setLoadingRoot(true);
    try {
      const result = await listDir(runId, "/workspace");
      setRoots(
        result.entries.map((entry) => ({
          entry,
          path: `/workspace/${entry.name}`,
          children: null,
          expanded: false,
          loading: false,
        }))
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoadingRoot(false);
    }
  }, [runId, roots, loadingRoot, listDir]);

  // Trigger root load
  if (roots === null && !loadingRoot && !error) {
    loadRoot();
  }

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

  if (error) {
    return (
      <div className="p-3 text-sm text-destructive">{error}</div>
    );
  }

  if (roots === null) {
    return (
      <div className="p-3 text-sm text-muted-foreground/60">Loading...</div>
    );
  }

  return (
    <div className="overflow-y-auto font-mono text-xs">
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
