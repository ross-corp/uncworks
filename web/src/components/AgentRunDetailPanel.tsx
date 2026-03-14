import { useState, useEffect } from "react";
import type { AgentRun, TraceSpan } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";
import SpecEditor from "./SpecEditor";
import LogViewer from "./LogViewer";
import FileExplorer from "./FileExplorer";
import ShellTerminal from "./ShellTerminal";
import TraceTimeline from "./TraceTimeline";
import DiffViewer from "./DiffViewer";
import { useWatchRun } from "../hooks/useWatchRun";
import { useTraces } from "../hooks/useTraces";

type TabId = "info" | "logs" | "files" | "shell" | "traces";

interface Tab {
  id: TabId;
  label: string;
  testId: string;
  alwaysEnabled: boolean;
}

const TABS: Tab[] = [
  { id: "info", label: "Info", testId: "detail-tab-info", alwaysEnabled: true },
  { id: "logs", label: "Logs", testId: "detail-tab-logs", alwaysEnabled: true },
  { id: "files", label: "Files", testId: "detail-tab-files", alwaysEnabled: false },
  { id: "shell", label: "Shell", testId: "detail-tab-shell", alwaysEnabled: true },
  { id: "traces", label: "Traces", testId: "detail-tab-traces", alwaysEnabled: true },
];

export default function AgentRunDetailPanel({
  run,
  onClose,
  onCancel,
  onClone,
  onSendInput,
}: {
  run: AgentRun;
  onClose: () => void;
  onCancel: (id: string) => void;
  onClone: (run: AgentRun) => void;
  onSendInput: (id: string, input: string) => void;
}) {
  const [activeTab, setActiveTab] = useState<TabId>("info");
  const [humanInput, setHumanInput] = useState("");

  const isActive = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  // 12.3: Use debugActive status field for hasPod logic
  const hasPod = !!run.status.podName && (isActive || !!run.status.debugActive);

  // Watch run for live log streaming
  const streamRunId = isActive ? run.id : null;
  const { logLines, isStreaming } = useWatchRun(streamRunId);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  function isTabEnabled(tab: Tab): boolean {
    if (tab.alwaysEnabled) return true;
    return hasPod;
  }

  return (
    <div data-testid="detail-panel" className="flex h-full flex-col border-l border-edge bg-surface-0">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-edge px-5 py-3">
        <span className="font-mono text-sm text-txt-tertiary truncate">{run.id}</span>
        <button onClick={onClose} className="btn-ghost px-2 text-lg" aria-label="Close">
          &times;
        </button>
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-0 border-b border-edge bg-surface-0 px-2">
        {TABS.map((tab) => {
          const enabled = isTabEnabled(tab);
          const active = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              data-testid={tab.testId}
              disabled={!enabled}
              onClick={() => enabled && setActiveTab(tab.id)}
              className={`relative px-4 py-2 text-sm font-medium transition-colors ${
                active
                  ? "text-txt-primary"
                  : enabled
                  ? "text-txt-tertiary hover:text-txt-secondary"
                  : "text-txt-tertiary/40 cursor-not-allowed"
              }`}
              title={!enabled ? "Pod is not available" : undefined}
            >
              {tab.label}
              {active && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-accent" />
              )}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "info" && (
          <InfoTab
            run={run}
            humanInput={humanInput}
            setHumanInput={setHumanInput}
            onSendInput={onSendInput}
          />
        )}
        {activeTab === "logs" && (
          <LogsTab run={run} logLines={logLines} isStreaming={isStreaming} hasPod={hasPod} />
        )}
        {activeTab === "files" && hasPod && (
          <FileExplorer runId={run.id} />
        )}
        {activeTab === "files" && !hasPod && (
          <DisabledMessage message="Pod is no longer available. File browsing requires an active pod." />
        )}
        {activeTab === "shell" && (
          <ShellTab run={run} isActive={isActive} hasPod={hasPod} />
        )}
        {activeTab === "traces" && (
          <TracesTab runId={run.id} />
        )}
      </div>

      {/* Footer */}
      <div className="flex justify-between border-t border-edge px-5 py-3">
        <div className="flex items-center gap-2">
          {run.status.traceID && (
            <button
              onClick={() => navigator.clipboard.writeText(run.status.traceID)}
              className="btn-ghost text-sm"
            >
              Copy Trace
            </button>
          )}
          <button
            onClick={() => onClone(run)}
            className="btn-ghost text-sm"
          >
            Clone Run
          </button>
        </div>
        {isActive && (
          <button
            data-testid="detail-cancel"
            onClick={() => onCancel(run.id)}
            className="rounded bg-danger px-3 py-1.5 text-sm font-medium text-white hover:bg-danger/80 transition-colors"
          >
            Cancel Run
          </button>
        )}
      </div>
    </div>
  );
}

/* ── InfoTab: extracted from original detail panel (zero behavior change) ── */

function InfoTab({
  run,
  humanInput,
  setHumanInput,
  onSendInput,
}: {
  run: AgentRun;
  humanInput: string;
  setHumanInput: (v: string) => void;
  onSendInput: (id: string, input: string) => void;
}) {
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
    <div className="flex-1 overflow-y-auto p-5">
      {/* Name & badges */}
      <h2 data-testid="detail-name" className="mb-2 text-base font-semibold">{run.name}</h2>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <span data-testid="detail-phase"><PhaseBadge phase={run.status.phase} /></span>
        <BackendBadge backend={run.spec.backend} />
        <ModelTierBadge tier={run.spec.modelTier} />
      </div>

      {/* Workspace */}
      {run.spec.workspaceName && (
        <div className="mb-4 space-y-2">
          <MetaRow label="Workspace" value={run.spec.workspaceName} />
        </div>
      )}

      {/* Repositories */}
      <div className="mb-4" data-testid="detail-repos">
        <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
          Repositories
        </h3>
        {run.spec.repos.length > 0 ? (
          <div className="space-y-1">
            {run.spec.repos.map((repo, i) => (
              <div key={i} className="flex items-baseline justify-between gap-4">
                <span className="font-mono text-xs text-txt-secondary truncate">
                  {repoName(repo.url)}
                </span>
                <span className="font-mono text-xs text-txt-tertiary whitespace-nowrap">
                  :{repo.branch}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <span className="text-xs text-txt-tertiary">&mdash;</span>
        )}
      </div>

      {/* Metadata grid */}
      <div className="mb-4 space-y-2">
        <MetaRow label="Created" value={formatTime(run.createdAt)} />
        <MetaRow label="Started" value={formatTime(run.status.startedAt)} />
        <MetaRow label="Duration" value={duration()} />
        {run.status.podName && (
          <MetaRow label="Pod" value={run.status.podName} mono />
        )}
        {run.status.deploymentName && (
          <MetaRow label="Deployment" value={run.status.deploymentName} mono />
        )}
        {run.status.traceID && (
          <MetaRow label="Trace ID" value={run.status.traceID} mono />
        )}
        <MetaRow label="TTL" value={`${run.spec.ttlSeconds}s`} />
      </div>

      {/* Env vars */}
      {Object.keys(run.spec.envVars).length > 0 && (
        <div className="mb-4">
          <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Environment
          </h3>
          <div className="space-y-1">
            {Object.entries(run.spec.envVars).map(([k, v]) => (
              <div key={k} className="font-mono text-xs">
                <span className="text-txt-tertiary">{k}</span>
                <span className="text-txt-secondary">=</span>
                <span className="text-txt-primary">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Spec */}
      {run.spec.specContent && (
        <div className="mb-4">
          <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Spec
          </h3>
          <SpecEditor value={run.spec.specContent} readOnly height="200px" />
          {run.spec.specSource && (
            <div className="mt-1 text-xs text-txt-tertiary">
              Source: {run.spec.specSource}
            </div>
          )}
        </div>
      )}

      {/* Prompt */}
      <div className="mb-4">
        <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
          Prompt
        </h3>
        <div className="rounded border border-edge bg-surface-2 p-3 text-sm text-txt-secondary whitespace-pre-wrap">
          {run.spec.prompt}
        </div>
      </div>

      {/* Status message */}
      {run.status.message && (
        <div className="mb-4">
          <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Status Message
          </h3>
          <div
            className={`rounded border p-3 text-sm whitespace-pre-wrap ${
              run.status.phase === "failed"
                ? "border-red-500/30 bg-red-500/5 text-red-400"
                : run.status.phase === "waiting_for_input"
                ? "border-purple-500/30 bg-purple-500/5 text-purple-400"
                : "border-edge bg-surface-2 text-txt-secondary"
            }`}
          >
            {run.status.message}
          </div>
        </div>
      )}

      {/* HITL Input */}
      {isWaiting && (
        <div className="mb-4">
          <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Human Input
          </h3>
          <textarea
            data-testid="detail-hitl-input"
            className="input-field min-h-[80px] resize-y"
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
          <button
            data-testid="detail-hitl-send"
            onClick={() => {
              if (humanInput.trim()) {
                onSendInput(run.id, humanInput.trim());
                setHumanInput("");
              }
            }}
            disabled={!humanInput.trim()}
            className="btn-primary mt-2 text-sm disabled:opacity-40"
          >
            Send Input
          </button>
        </div>
      )}
    </div>
  );
}

/* ── LogsTab ── */

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

  // Fetch logs from pod when available and not streaming
  useEffect(() => {
    if (!hasPod || isStreaming || logLines.length > 0) return;
    let cancelled = false;
    setFetching(true);
    fetch(`/api/v1/runs/${run.id}/logs`)
      .then((r) => r.ok ? r.text() : "")
      .then((text) => {
        if (!cancelled && text) setFetchedLines(text.split("\n"));
      })
      .catch(() => {})
      .finally(() => { if (!cancelled) setFetching(false); });
    return () => { cancelled = true; };
  }, [run.id, hasPod, isStreaming, logLines.length]);

  // Use streamed log lines for active runs, fetched from pod, or persisted logOutput
  const persistedLines = run.status.logOutput
    ? run.status.logOutput.split("\n")
    : [];

  const lines = logLines.length > 0 ? logLines : fetchedLines.length > 0 ? fetchedLines : persistedLines;
  const streaming = isStreaming && logLines.length > 0;

  if (lines.length === 0 && !isStreaming && !fetching) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
        No logs available
      </div>
    );
  }

  if (fetching && lines.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
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

/* ── ShellTab (12.1 + 12.2) ── */

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

  // 12.1: When Deployment replicas=0 (not active and not debugging), show Debug Run button
  if (!isActive && !debugActive) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-8">
        <p className="text-sm text-txt-tertiary text-center">
          The agent run has completed. Start a debug session to get shell access to the workspace.
        </p>
        <button
          data-testid="debug-run-btn"
          onClick={handleDebugStart}
          disabled={debugLoading}
          className="btn-primary text-sm disabled:opacity-40"
        >
          {debugLoading ? "Starting..." : "Debug Run"}
        </button>

        {/* 12.2: VS Code connection info */}
        <VSCodeConnectInfo podName={run.status.podName} />
      </div>
    );
  }

  // When debug is active or run is active, show terminal with optional stop button
  return (
    <div className="flex h-full flex-col">
      {debugActive && !isActive && (
        <div className="flex items-center justify-between border-b border-edge px-3 py-2">
          <span className="text-xs text-green-400 font-medium">Debug session active</span>
          <button
            data-testid="stop-debug-btn"
            onClick={handleDebugStop}
            disabled={debugLoading}
            className="rounded bg-danger/80 px-2 py-1 text-xs font-medium text-white hover:bg-danger transition-colors disabled:opacity-40"
          >
            {debugLoading ? "Stopping..." : "Stop Debug"}
          </button>
        </div>
      )}
      {hasPod ? (
        <div className="flex-1 overflow-hidden">
          <ShellTerminal runId={run.id} />
        </div>
      ) : (
        <div className="flex flex-1 items-center justify-center text-sm text-txt-tertiary">
          Waiting for pod to become ready...
        </div>
      )}

      {/* 12.2: VS Code connection info */}
      <VSCodeConnectInfo podName={run.status.podName} />
    </div>
  );
}

/* ── VS Code Connection Info (12.2) ── */

function VSCodeConnectInfo({ podName }: { podName: string }) {
  const [expanded, setExpanded] = useState(false);

  if (!podName) return null;

  const portForwardCmd = `kubectl port-forward pod/${podName} 50052:50052 -n aot-local`;

  return (
    <div className="w-full max-w-md">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-xs text-txt-tertiary hover:text-txt-secondary transition-colors"
      >
        <span className={`transition-transform ${expanded ? "rotate-90" : ""}`}>&#9654;</span>
        VS Code Connection
      </button>
      {expanded && (
        <div className="mt-2 rounded border border-edge bg-surface-2 p-3">
          <p className="mb-2 text-xs text-txt-tertiary">
            Connect VS Code Remote Containers to the workspace:
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-surface-0 px-2 py-1 font-mono text-xs text-txt-secondary break-all">
              {portForwardCmd}
            </code>
            <button
              onClick={() => navigator.clipboard.writeText(portForwardCmd)}
              className="btn-ghost px-2 text-xs flex-shrink-0"
            >
              Copy
            </button>
          </div>
          <p className="mt-2 text-xs text-txt-tertiary">
            Then attach to the container using VS Code &quot;Attach to Running Container&quot;.
          </p>
        </div>
      )}
    </div>
  );
}

/* ── TracesTab (11.4, 11.5, 11.7, 12.5) ── */

function TracesTab({ runId }: { runId: string }) {
  const { spans, loading } = useTraces(runId);
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | null>(null);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
        Loading traces...
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Timeline */}
      <div className="flex-shrink-0 border-b border-edge overflow-y-auto max-h-[40%]">
        <TraceTimeline
          spans={spans}
          selectedSpanId={selectedSpan?.id}
          onSelectSpan={setSelectedSpan}
        />
      </div>

      {/* Detail pane */}
      <div className="flex-1 overflow-y-auto">
        {selectedSpan === null && (
          <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
            Select a span to view details
          </div>
        )}

        {/* 11.4: Tool span -> show DiffViewer */}
        {selectedSpan !== null && selectedSpan.type === "tool" && selectedSpan.diff && (
          <DiffViewer diff={selectedSpan.diff} />
        )}

        {/* Tool span without diff */}
        {selectedSpan !== null && selectedSpan.type === "tool" && !selectedSpan.diff && (
          <SpanMetadataView span={selectedSpan} />
        )}

        {/* 11.5: LLM span -> show metadata */}
        {selectedSpan !== null && selectedSpan.type === "llm" && (
          <SpanMetadataView span={selectedSpan} />
        )}

        {/* Other span types -> show metadata */}
        {selectedSpan !== null && selectedSpan.type !== "tool" && selectedSpan.type !== "llm" && (
          <SpanMetadataView span={selectedSpan} />
        )}
      </div>
    </div>
  );
}

/* ── SpanMetadataView ── */

function SpanMetadataView({ span }: { span: TraceSpan }) {
  return (
    <div className="p-4 space-y-3">
      <div className="space-y-2">
        <MetaRow label="Name" value={span.name} />
        <MetaRow label="Type" value={span.type} />
        <MetaRow label="Start" value={new Date(span.startTime).toLocaleString()} />
        <MetaRow label="End" value={new Date(span.endTime).toLocaleString()} />
        {span.parentId && <MetaRow label="Parent" value={span.parentId} mono />}
      </div>

      {span.metadata && Object.keys(span.metadata).length > 0 && (
        <div>
          <h4 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Metadata
          </h4>
          <pre className="rounded border border-edge bg-surface-2 p-3 text-xs font-mono text-txt-secondary overflow-x-auto whitespace-pre-wrap">
            {JSON.stringify(span.metadata, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

/* ── DisabledMessage ── */

function DisabledMessage({ message }: { message: string }) {
  return (
    <div className="flex h-full items-center justify-center p-8 text-center text-sm text-txt-tertiary">
      {message}
    </div>
  );
}

/* ── MetaRow ── */

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <span className="text-xs text-txt-tertiary whitespace-nowrap">{label}</span>
      <span
        className={`text-sm text-txt-secondary text-right truncate ${mono ? "font-mono text-xs" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}
