import { useState, useRef, useEffect, useCallback } from "react";
import type { AgentRun } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";
import { SkeletonRow } from "./Skeleton";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";

function ActionMenu({
  run,
  onCancel,
  onClone,
  onDelete,
}: {
  run: AgentRun;
  onCancel: (id: string) => void;
  onClone: (run: AgentRun) => void;
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
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onCancel(run.id)}
          className="whitespace-nowrap text-xs text-destructive opacity-0 group-hover:opacity-100"
        >
          Cancel
        </Button>
      )}
      <div className="relative" ref={menuRef}>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setMenuOpen(!menuOpen)}
          className="text-xs opacity-0 group-hover:opacity-100"
          aria-label="More options"
        >
          &middot;&middot;&middot;
        </Button>
        {menuOpen && (
          <div className="absolute right-0 bottom-full z-10 mb-1 w-36 border border-border bg-card py-1 shadow-lg">
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
              onClick={() => {
                navigator.clipboard.writeText(run.id);
                setMenuOpen(false);
              }}
            >
              Copy ID
            </button>
            {run.status.traceID && (
              <button
                className="w-full px-3 py-1.5 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                onClick={() => {
                  navigator.clipboard.writeText(run.status.traceID);
                  setMenuOpen(false);
                }}
              >
                Copy Trace ID
              </button>
            )}
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
              onClick={() => {
                onClone(run);
                setMenuOpen(false);
              }}
            >
              Clone Run
            </button>
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-destructive hover:bg-destructive/10"
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
  { key: "repo",    label: "Repos",   defaultWidth: 180, minWidth: 80 },
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
  onClone,
  onDelete,
  loading,
  onNewRun,
}: {
  runs: AgentRun[];
  selectedRunId?: string | null;
  onSelect?: (run: AgentRun) => void;
  onCancel: (id: string) => void;
  onClone: (run: AgentRun) => void;
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
        className="absolute top-0 -right-px bottom-0 z-10 w-[3px] cursor-col-resize bg-border hover:bg-muted-foreground transition-colors"
      />
    );
  }

  function repoName(url: string): string {
    const parts = url.split("/");
    return parts[parts.length - 1] || url;
  }

  function reposSummary(run: AgentRun): { text: string; title: string } {
    const repos = run.spec.repos;
    if (!repos || repos.length === 0) return { text: "\u2014", title: "" };
    const names = repos.map((r) => repoName(r.url));
    const title = repos.map((r) => `${r.url}:${r.branch}`).join("\n");
    if (names.length <= 2) return { text: names.join(", "), title };
    return { text: `${names[0]}, ${names[1]} +${names.length - 2}`, title };
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
            <tr className="border-b border-border text-left text-xs font-medium text-muted-foreground/60">
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
        <p className="text-sm text-muted-foreground/60">No agent runs match the current filters.</p>
        {onNewRun && (
          <Button onClick={onNewRun} className="mt-3 text-sm">
            + Create Agent Run
          </Button>
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
          <tr className="border-b border-border text-left text-xs font-medium text-muted-foreground/60">
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
                data-testid={`table-row-${run.id}`}
                onClick={() => onSelect?.(run)}
                className={`group border-b border-border transition-colors cursor-pointer ${
                  isSelected
                    ? "bg-muted"
                    : "hover:bg-card"
                }`}
              >
                <td className="px-4 py-2.5 overflow-hidden text-ellipsis whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <span className="font-medium">{run.name}</span>
                    {run.spec.specContent && (
                      <Badge data-testid={`table-row-${run.id}-spec`} variant="outline" className="border-secondary/30 text-secondary text-[10px] px-1.5 py-0.5">
                        spec
                      </Badge>
                    )}
                    <span className="text-xs text-muted-foreground/60">{timeAgo(run.createdAt)}</span>
                  </div>
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap" data-testid={`table-row-${run.id}-phase`}>
                  <PhaseBadge phase={run.status.phase} />
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <BackendBadge backend={run.spec.backend} />
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <ModelTierBadge tier={run.spec.modelTier} />
                </td>
                <td
                  className="px-4 py-2.5 font-mono text-xs text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap"
                  title={reposSummary(run).title}
                >
                  {reposSummary(run).text}
                </td>
                <td className="px-4 py-2.5 text-xs text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap">
                  {run.status.message || <span className="text-muted-foreground/60">&mdash;</span>}
                </td>
                <td className="px-4 py-2.5 overflow-hidden whitespace-nowrap">
                  <ActionMenu
                    run={run}
                    onCancel={onCancel}
                    onClone={onClone}
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
