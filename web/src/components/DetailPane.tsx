import { useState, useEffect } from "react";
import type { AgentRun, TraceSpan } from "../types/agent-run";
import { getRunLabel } from "../lib/runLabel";
import { PhaseBadge } from "./StatusBadge";
import LogViewer from "./LogViewer";
import FileExplorer from "./FileExplorer";
import ShellTerminal from "./ShellTerminal";
import TraceTimeline from "./TraceTimeline";
import DiffViewer from "./DiffViewer";
import { useWatchRun } from "../hooks/useWatchRun";
import { useTraces } from "../hooks/useTraces";
import { Button } from "./ui/button";

type TabId = "info" | "logs" | "files" | "shell" | "traces";

interface Tab {
  id: TabId;
  label: string;
  testId: string;
}

const TABS: Tab[] = [
  { id: "info", label: "Info", testId: "detail-tab-info" },
  { id: "logs", label: "Logs", testId: "detail-tab-logs" },
  { id: "files", label: "Files", testId: "detail-tab-files" },
  { id: "shell", label: "Shell", testId: "detail-tab-shell" },
  { id: "traces", label: "Traces", testId: "detail-tab-traces" },
];

interface DetailPaneProps {
  run: AgentRun;
  onClose: () => void;
  onCancel: (id: string) => void;
  onClone: (run: AgentRun) => void;
  onSendInput: (id: string, input: string) => void;
  /** Controlled active tab — when provided, the parent drives tab state (e.g. via keyboard Tab cycling). */
  activeTab?: TabId;
  onTabChange?: (tab: TabId) => void;
  /** Called when the watch stream receives a phase_changed event, so the parent can re-fetch runs. */
  onRefresh?: () => void;
}

/**
 * DetailPane — simplified detail view for use inside a split-pane layout.
 *
 * Reuses existing LogViewer, FileExplorer, ShellTerminal, TraceTimeline, and
 * DiffViewer components.  No breadcrumbs (the run list is visible beside it).
 * Minimal 16px padding.  Clone and Cancel buttons in the header as small ghost
 * buttons.
 */
export default function DetailPane({
  run,
  onClose,
  onCancel,
  onClone,
  onSendInput,
  activeTab: controlledTab,
  onTabChange,
  onRefresh,
}: DetailPaneProps) {
  const [internalTab, setInternalTab] = useState<TabId>("info");
  const activeTab = controlledTab ?? internalTab;

  function handleTabChange(tab: TabId) {
    setInternalTab(tab);
    onTabChange?.(tab);
  }

  // Reset to Info tab when run changes
  const runId = run.id;
  useEffect(() => {
    setInternalTab("info");
    onTabChange?.("info");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [runId]);

  const isActive = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  const hasPod = !!run.status.podName;

  const isWatchable = run.status.phase === "pending" || isActive;
  const streamRunId = isWatchable ? run.id : null;
  const { logLines, isStreaming } = useWatchRun(streamRunId, onRefresh ? () => onRefresh() : undefined);

  // Phase color mapping using semantic tokens
  const phaseColorMap: Record<string, string> = {
    running: "var(--color-accent, #3b82f6)",
    pending: "var(--color-muted, #6b7280)",
    waiting_for_input: "var(--color-warning, #f59e0b)",
    succeeded: "var(--color-success, #22c55e)",
    failed: "var(--color-error, #ef4444)",
    cancelled: "var(--color-neutral, #6b7280)",
  };

  return (
    <div data-testid="run-detail" className="flex h-full flex-col" style={{ padding: 0 }}>
      {/* Header: run name, phase, close X, clone/cancel */}
      <div
        className="flex items-center justify-between border-b px-4 py-2"
        style={{ borderColor: "var(--color-border, hsl(var(--border)))" }}
      >
        <div className="flex items-center gap-3 min-w-0">
          <span className="text-sm font-bold truncate">{getRunLabel(run)}</span>
          {run.spec.displayName && (
            <span className="text-xs font-mono truncate" style={{ color: "var(--color-muted)" }}>{run.name}</span>
          )}
          <span
            className="text-xs font-medium"
            data-testid="detail-phase"
            style={{ color: phaseColorMap[run.status.phase] ?? "inherit" }}
          >
            <PhaseBadge phase={run.status.phase} />
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onClone(run)}
            className="text-xs h-7 px-2"
          >
            Clone
          </Button>
          {isActive && (
            <Button
              data-testid="detail-cancel"
              variant="ghost"
              size="sm"
              onClick={() => onCancel(run.id)}
              className="text-xs h-7 px-2 text-destructive hover:text-destructive"
            >
              Cancel
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label="Close"
            className="text-lg leading-none h-7 w-7 p-0"
          >
            &times;
          </Button>
        </div>
      </div>

      {/* Tab bar — simple text tabs with accent bottom border on active */}
      <div
        className="flex items-center gap-0 border-b px-2"
        style={{ borderColor: "var(--color-border, hsl(var(--border)))" }}
      >
        {TABS.map((tab) => {
          const active = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              data-testid={tab.testId}
              onClick={() => handleTabChange(tab.id)}
              className={`relative px-3 py-2 text-xs font-medium transition-colors ${
                active
                  ? "text-foreground"
                  : "text-muted-foreground/60 hover:text-muted-foreground"
              }`}
            >
              {tab.label}
              {active && (
                <span
                  className="absolute bottom-0 left-0 right-0 h-0.5"
                  style={{ backgroundColor: "var(--color-accent, hsl(var(--primary)))" }}
                />
              )}
            </button>
          );
        })}
      </div>

      {/* Tab content — 16px padding */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "info" && (
          <InfoTab run={run} onSendInput={onSendInput} />
        )}
        {activeTab === "logs" && (
          <LogsTab run={run} logLines={logLines} isStreaming={isStreaming} hasPod={hasPod} />
        )}
        {activeTab === "files" && (
          <FileExplorer runId={run.id} />
        )}
        {activeTab === "shell" && (
          <ShellTab run={run} isActive={isActive} hasPod={hasPod} />
        )}
        {activeTab === "traces" && (
          <TracesTab runId={run.id} />
        )}
      </div>
    </div>
  );
}

/* -- InfoTab -- */

function InfoTab({
  run,
  onSendInput,
}: {
  run: AgentRun;
  onSendInput: (id: string, input: string) => void;
}) {
  const [humanInput, setHumanInput] = useState("");
  const isWaiting = run.status.phase === "waiting_for_input";

  function repoName(url: string): string {
    const parts = url.split("/");
    return parts.slice(-2).join("/");
  }

  function formatTime(iso: string): string {
    if (!iso) return "\u2014";
    return new Date(iso).toLocaleString();
  }

  function duration(): string {
    if (!run.status.startedAt) return "\u2014";
    const start = new Date(run.status.startedAt).getTime();
    const end = run.status.completedAt
      ? new Date(run.status.completedAt).getTime()
      : Date.now();
    const secs = Math.floor((end - start) / 1000);
    if (secs < 60) return `${secs}s`;
    const mins = Math.floor(secs / 60);
    const remSecs = secs % 60;
    if (mins < 60) return `${mins}m ${remSecs}s`;
    return `${Math.floor(mins / 60)}h ${mins % 60}m`;
  }

  return (
    <div className="overflow-y-auto p-4 space-y-4">
      {/* Run ID + full prompt + repo + phase + timestamps */}
      <div data-testid="detail-name" className="text-sm font-semibold">{getRunLabel(run)}</div>

      <div className="space-y-2 text-xs">
        <MetaRow label="Run ID" value={run.id} mono />
        <MetaRow label="Phase" value={run.status.phase} />
        <MetaRow label="Duration" value={duration()} />
        <MetaRow label="Created" value={formatTime(run.createdAt)} />
        {run.status.startedAt && <MetaRow label="Started" value={formatTime(run.status.startedAt)} />}
        {run.status.completedAt && <MetaRow label="Completed" value={formatTime(run.status.completedAt)} />}
      </div>

      {/* Repositories */}
      {run.spec.repos.length > 0 && (
        <div data-testid="detail-repos">
          <h3 className="mb-1 text-xs font-medium text-muted-foreground/60">Repositories</h3>
          <div className="space-y-1">
            {run.spec.repos.map((repo, i) => (
              <div key={i} className="flex items-baseline justify-between gap-4">
                <span className="font-mono text-xs text-muted-foreground truncate">
                  {repoName(repo.url)}
                </span>
                <span className="font-mono text-xs text-muted-foreground/60 whitespace-nowrap">
                  :{repo.branch}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Prompt */}
      <div>
        <h3 className="mb-1 text-xs font-medium text-muted-foreground/60">Prompt</h3>
        <div className="border border-border bg-muted p-3 text-xs text-muted-foreground whitespace-pre-wrap max-h-48 overflow-y-auto">
          {run.spec.prompt}
        </div>
      </div>

      {/* Status message */}
      {run.status.message && (
        <div>
          <h3 className="mb-1 text-xs font-medium text-muted-foreground/60">Status Message</h3>
          <div
            className={`border p-3 text-xs whitespace-pre-wrap ${
              run.status.phase === "failed"
                ? "border-destructive/30 bg-destructive/5 text-destructive"
                : "border-border bg-muted text-muted-foreground"
            }`}
          >
            {run.status.message}
          </div>
        </div>
      )}

      {/* HITL Input */}
      {isWaiting && (
        <div>
          <h3 className="mb-1 text-xs font-medium text-muted-foreground/60">Human Input</h3>
          <textarea
            data-testid="detail-hitl-input"
            className="w-full border border-input bg-background px-3 py-2 text-xs text-foreground placeholder-muted-foreground/60 outline-none transition-colors focus:border-primary min-h-[60px] resize-y"
            value={humanInput}
            onChange={(e) => setHumanInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                if (humanInput.trim()) {
                  onSendInput(run.id, humanInput.trim());
                  setHumanInput("");
                }
              }
            }}
            placeholder="Type your response to the agent..."
          />
          <Button
            data-testid="detail-hitl-send"
            size="sm"
            onClick={() => {
              if (humanInput.trim()) {
                onSendInput(run.id, humanInput.trim());
                setHumanInput("");
              }
            }}
            disabled={!humanInput.trim()}
            className="mt-1 text-xs disabled:opacity-40"
          >
            Send Input
          </Button>
        </div>
      )}
    </div>
  );
}

/* -- LogsTab -- */

function LogsTab({
  run,
  logLines,
  isStreaming,
  hasPod,
}: {
  run: AgentRun;
  logLines: string[];
  isStreaming: boolean;
  hasPod: boolean;
}) {
  const [fetchedLines, setFetchedLines] = useState<string[]>([]);
  const [fetching, setFetching] = useState(false);

  useEffect(() => {
    if (!hasPod || isStreaming || logLines.length > 0) return;
    let cancelled = false;
    setFetching(true);
    fetch(`/api/v1/runs/${run.id}/logs`)
      .then((r) => (r.ok ? r.text() : ""))
      .then((text) => {
        if (!cancelled && text) setFetchedLines(text.split("\n"));
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setFetching(false);
      });
    return () => {
      cancelled = true;
    };
  }, [run.id, hasPod, isStreaming, logLines.length]);

  const persistedLines = run.status.logOutput ? run.status.logOutput.split("\n") : [];
  const lines =
    logLines.length > 0 ? logLines : fetchedLines.length > 0 ? fetchedLines : persistedLines;
  const streaming = isStreaming && logLines.length > 0;

  if (lines.length === 0 && !isStreaming && !fetching) {
    return (
      <div className="flex h-full items-center justify-center text-xs text-muted-foreground/60">
        No logs available
      </div>
    );
  }

  if (fetching && lines.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-xs text-muted-foreground/60">
        Loading logs...
      </div>
    );
  }

  return (
    <div className="h-full p-2">
      <LogViewer lines={lines} streaming={streaming} />
    </div>
  );
}

/* -- ShellTab -- */

function ShellTab({
  run,
  isActive,
  hasPod,
}: {
  run: AgentRun;
  isActive: boolean;
  hasPod: boolean;
}) {
  const [debugLoading, setDebugLoading] = useState(false);
  const debugActive = !!run.status.debugActive;

  async function handleDebugStart() {
    setDebugLoading(true);
    try {
      await fetch(`/api/v1/runs/${run.id}/debug`, { method: "POST" });
    } catch (err) {
      console.error("Failed to start debug:", err);
    } finally {
      setDebugLoading(false);
    }
  }

  async function handleDebugStop() {
    setDebugLoading(true);
    try {
      await fetch(`/api/v1/runs/${run.id}/debug`, { method: "DELETE" });
    } catch (err) {
      console.error("Failed to stop debug:", err);
    } finally {
      setDebugLoading(false);
    }
  }

  if (!isActive && !debugActive) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-4">
        <p className="text-xs text-muted-foreground/60 text-center">
          Run completed. Start a debug session for shell access.
        </p>
        <Button
          data-testid="debug-run-btn"
          size="sm"
          onClick={handleDebugStart}
          disabled={debugLoading}
          className="text-xs disabled:opacity-40"
        >
          {debugLoading ? "Starting..." : "Debug Run"}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {debugActive && !isActive && (
        <div className="flex items-center justify-between border-b border-border px-3 py-1">
          <span className="text-xs text-secondary font-medium">Debug session active</span>
          <Button
            data-testid="stop-debug-btn"
            variant="destructive"
            size="sm"
            onClick={handleDebugStop}
            disabled={debugLoading}
            className="text-xs"
          >
            {debugLoading ? "Stopping..." : "Stop Debug"}
          </Button>
        </div>
      )}
      {hasPod ? (
        <div className="flex-1 overflow-hidden">
          <ShellTerminal runId={run.id} />
        </div>
      ) : (
        <div className="flex flex-1 items-center justify-center text-xs text-muted-foreground/60">
          Waiting for pod...
        </div>
      )}
    </div>
  );
}

/* -- TracesTab -- */

function TracesTab({ runId }: { runId: string }) {
  const { spans, loading } = useTraces(runId);
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | null>(null);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-xs text-muted-foreground/60">
        Loading traces...
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-shrink-0 border-b border-border overflow-y-auto max-h-[40%]">
        <TraceTimeline
          spans={spans}
          selectedSpanId={selectedSpan?.id}
          onSelectSpan={setSelectedSpan}
        />
      </div>
      <div className="flex-1 overflow-y-auto">
        {selectedSpan === null && (
          <div
            data-testid="traces-empty-state"
            className="flex h-full items-center justify-center text-xs text-muted-foreground/60"
          >
            Select a span to view details
          </div>
        )}
        {selectedSpan !== null && selectedSpan.type === "tool" && selectedSpan.diff && (
          <DiffViewer diff={selectedSpan.diff} />
        )}
        {selectedSpan !== null && selectedSpan.type === "tool" && !selectedSpan.diff && (
          <SpanMetadataView span={selectedSpan} />
        )}
        {selectedSpan !== null && selectedSpan.type === "llm" && (
          <SpanMetadataView span={selectedSpan} />
        )}
        {selectedSpan !== null && selectedSpan.type !== "tool" && selectedSpan.type !== "llm" && (
          <SpanMetadataView span={selectedSpan} />
        )}
      </div>
    </div>
  );
}

/* -- SpanMetadataView -- */

function SpanMetadataView({ span }: { span: TraceSpan }) {
  return (
    <div className="p-4 space-y-2">
      <MetaRow label="Name" value={span.name} />
      <MetaRow label="Type" value={span.type} />
      <MetaRow label="Start" value={new Date(span.startTime).toLocaleString()} />
      <MetaRow label="End" value={new Date(span.endTime).toLocaleString()} />
      {span.parentId && <MetaRow label="Parent" value={span.parentId} mono />}
      {span.metadata && Object.keys(span.metadata).length > 0 && (
        <div>
          <h4 className="mb-1 text-xs font-medium text-muted-foreground/60">Metadata</h4>
          <pre className="border border-border bg-muted p-2 text-xs font-mono text-muted-foreground overflow-x-auto whitespace-pre-wrap">
            {JSON.stringify(span.metadata, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

/* -- MetaRow -- */

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <span className="text-xs text-muted-foreground/60 whitespace-nowrap">{label}</span>
      <span
        className={`text-xs text-muted-foreground text-right truncate ${mono ? "font-mono" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}
