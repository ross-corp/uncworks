import { useState, useEffect } from "react";
import type { AgentRun } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";
import SpecEditor from "./SpecEditor";
import LogViewer from "./LogViewer";
import FileExplorer from "./FileExplorer";
import ShellTerminal from "./ShellTerminal";
import { useWatchRun } from "../hooks/useWatchRun";

type TabId = "info" | "logs" | "files" | "shell";

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
  { id: "shell", label: "Shell", testId: "detail-tab-shell", alwaysEnabled: false },
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
  const isRetained = !!run.status.podName && !!run.status.retainUntil && new Date(run.status.retainUntil).getTime() > Date.now();
  const hasPod = !!run.status.podName && (isActive || isRetained);

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

  // Pod expiry countdown
  const podExpiresIn = getPodExpiresIn(run);

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

        {/* Pod expiry indicator */}
        {podExpiresIn !== null && (
          <span className="ml-auto text-xs text-yellow-400 px-2">
            Pod expires in {podExpiresIn}
          </span>
        )}
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
        {activeTab === "shell" && hasPod && (
          <ShellTerminal runId={run.id} />
        )}
        {activeTab === "shell" && !hasPod && (
          <DisabledMessage message="Pod is no longer available. Shell access requires an active pod." />
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

/* ── Pod expiry helper ── */

function getPodExpiresIn(run: AgentRun): string | null {
  // Check for retainUntil on status (added by section 10)
  const statusAny = run.status as unknown as Record<string, unknown>;
  if (typeof statusAny.retainUntil !== "string" || !statusAny.retainUntil) return null;

  const retainUntil = new Date(statusAny.retainUntil as string).getTime();
  const now = Date.now();
  const diffMs = retainUntil - now;

  if (diffMs <= 0) return null;

  const mins = Math.ceil(diffMs / 60000);
  return `${mins} min`;
}
