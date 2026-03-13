import { useState, useRef, useEffect, useCallback } from "react";
import type { AgentRun } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";
import { SkeletonRow } from "./Skeleton";

function ActionMenu({
  run,
  onCancel,
  onDelete,
}: {
  run: AgentRun;
  onCancel: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node))
        setMenuOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const isActive = run.status.phase === "running" || run.status.phase === "waiting_for_input";

  return (
    <div
      className="flex items-center gap-1"
      onClick={(e) => e.stopPropagation()}
    >
      {isActive && (
        <button
          onClick={() => onCancel(run.id)}
          className="btn-ghost whitespace-nowrap px-2 py-1 text-xs text-danger opacity-0 group-hover:opacity-100"
        >
          Cancel
        </button>
      )}
      <div className="relative" ref={menuRef}>
        <button
          onClick={() => setMenuOpen(!menuOpen)}
          className="btn-ghost px-2 py-1 text-xs opacity-0 group-hover:opacity-100"
        >
          &middot;&middot;&middot;
        </button>
        {menuOpen && (
          <div className="absolute right-0 bottom-full z-10 mb-1 w-36 rounded border border-edge bg-surface-1 py-1 shadow-lg">
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-txt-secondary hover:bg-surface-2 hover:text-txt-primary"
              onClick={() => {
                navigator.clipboard.writeText(run.id);
                setMenuOpen(false);
              }}
            >
              Copy ID
            </button>
            {run.status.traceID && (
              <button
                className="w-full px-3 py-1.5 text-left text-sm text-txt-secondary hover:bg-surface-2 hover:text-txt-primary"
                onClick={() => {
                  navigator.clipboard.writeText(run.status.traceID);
                  setMenuOpen(false);
                }}
              >
                Copy Trace ID
              </button>
            )}
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-danger hover:bg-danger/10"
              onClick={() => {
                onDelete(run.id);
                setMenuOpen(false);
              }}
            >
              Delete
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

const COLUMNS = [
  { key: "name",    label: "Name",    defaultWidth: 200, minWidth: 100 },
  { key: "phase",   label: "Phase",   defaultWidth: 160, minWidth: 100 },
  { key: "backend", label: "Backend", defaultWidth: 100, minWidth: 70 },
  { key: "model",   label: "Model",   defaultWidth: 90,  minWidth: 70 },
  { key: "repo",    label: "Repo",    defaultWidth: 180, minWidth: 80 },
  { key: "message", label: "Message", defaultWidth: 280, minWidth: 100 },
  { key: "actions", label: "",        defaultWidth: 140, minWidth: 80 },
];

const MIN_WIDTHS: Record<string, number> = Object.fromEntries(
  COLUMNS.map((c) => [c.key, c.minWidth])
);

export default function AgentRunTable({
  runs,
  selectedRunId,
  onSelect,
  onCancel,
  onDelete,
  loading,
  onNewRun,
}: {
  runs: AgentRun[];
  selectedRunId?: string | null;
  onSelect?: (run: AgentRun) => void;
  onCancel: (id: string) => void;
  onDelete: (id: string) => void;
  loading?: boolean;
  onNewRun?: () => void;
}) {
  // (loading used for skeleton rendering below)
  const [colWidths, setColWidths] = useState<Record<string, number>>(() =>
    Object.fromEntries(COLUMNS.map((c) => [c.key, c.defaultWidth]))
  );
  const dragRef = useRef<{ col: string; startX: number; startWidth: number } | null>(null);

  const onMouseMove = useCallback((e: MouseEvent) => {
    if (!dragRef.current) return;
    const { col, startX, startWidth } = dragRef.current;
    const delta = e.clientX - startX;
    const newWidth = Math.max(MIN_WIDTHS[col] ?? 40, startWidth + delta);
    setColWidths((prev) => ({ ...prev, [col]: newWidth }));
  }, []);

  const onMouseUp = useCallback(() => {
    dragRef.current = null;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }, []);

  useEffect(() => {
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [onMouseMove, onMouseUp]);

  function startResize(col: string, e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    dragRef.current = { col, startX: e.clientX, startWidth: colWidths[col] };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }

  function ResizeHandle({ col }: { col: string }) {
    return (
      <div
        onMouseDown={(e) => startResize(col, e)}
        className="absolute top-0 -right-px bottom-0 z-10 w-[3px] cursor-col-resize bg-edge hover:bg-txt-secondary transition-colors"
      />
    );
  }

  function repoName(url: string): string {
    const parts = url.split("/");
    return parts[parts.length - 1] || url;
  }

  function timeAgo(iso: string): string {
    if (!iso) return "";
    const diff = Date.now() - new Date(iso).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  }

  if (loading && runs.length === 0) {
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
          <colgroup>
            {COLUMNS.map((c) => (
              <col key={c.key} style={{ width: colWidths[c.key] }} />
            ))}
            <col />
          </colgroup>
          <thead>
            <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
              {COLUMNS.map((c) => (
                <th key={c.key} className="relative px-4 py-2">{c.label}</th>
              ))}
              <th />
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 5 }).map((_, i) => (
              <SkeletonRow key={i} />
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div className="px-6 py-12 text-center">
        <p className="text-sm text-txt-tertiary">No agent runs match the current filters.</p>
        {onNewRun && (
          <button onClick={onNewRun} className="btn-primary mt-3 text-sm">
            + Create Agent Run
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          {COLUMNS.map((c) => (
            <col key={c.key} style={{ width: colWidths[c.key] }} />
          ))}
          <col />
        </colgroup>
        <thead>
          <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
            {COLUMNS.map((c) => (
              <th key={c.key} className="relative px-4 py-2">
                {c.label}
                <ResizeHandle col={c.key} />
              </th>
            ))}
            <th />
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => {
            const isSelected = selectedRunId === run.id;
            return (
              <tr
                key={run.id}
                onClick={() => onSelect?.(run)}
                className={`group border-b border-edge transition-colors cursor-pointer ${
                  isSelected
                    ? "bg-surface-2"
                    : "hover:bg-surface-1"
                }`}
              >
                <td className="px-4 py-2.5 overflow-hidden text-ellipsis whitespace-nowrap">
                  <div>
                    <span className="font-medium">{run.name}</span>
                    <span className="ml-2 text-xs text-txt-tertiary">{timeAgo(run.createdAt)}</span>
                  </div>
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <PhaseBadge phase={run.status.phase} />
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <BackendBadge backend={run.spec.backend} />
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <ModelTierBadge tier={run.spec.modelTier} />
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                  {repoName(run.spec.repoURL)}
                  {run.spec.branch !== "main" && (
                    <span className="ml-1 text-txt-tertiary">:{run.spec.branch}</span>
                  )}
                </td>
                <td className="px-4 py-2.5 text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                  {run.status.message || <span className="text-txt-tertiary">&mdash;</span>}
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <ActionMenu
                    run={run}
                    onCancel={onCancel}
                    onDelete={onDelete}
                  />
                </td>
                <td />
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
