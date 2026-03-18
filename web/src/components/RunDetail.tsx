import { useState, useEffect } from "react";
import type { AgentRun, TraceSpan, RunGraphNode, RunGraphEdge } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";
import SpecEditor from "./SpecEditor";
import LogViewer from "./LogViewer";
import FileExplorer from "./FileExplorer";
import ShellTerminal from "./ShellTerminal";
import TraceTimeline from "./TraceTimeline";
import DiffViewer from "./DiffViewer";
import RunGraph from "./RunGraph";
import { useWatchRun } from "../hooks/useWatchRun";
import { useTraces } from "../hooks/useTraces";
import { apiFetch } from "../hooks/apiFetch";
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

export default function RunDetail({
  run,
  onClose,
  onCancel,
  onClone,
  onSendInput,
  onRefresh,
}: {
  run: AgentRun;
  onClose: () => void;
  onCancel: (id: string) => void;
  onClone: (run: AgentRun) => void;
  onSendInput: (id: string, input: string) => void;
  onRefresh?: () => void;
}) {
  const [activeTab, setActiveTab] = useState<TabId>("info");
  const [humanInput, setHumanInput] = useState("");

  const isActive = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  const hasPod = !!run.status.podName;
  const hasOrchestration = (run.children && run.children.length > 0) || !!run.spec.parentRunId;

  const isWatchable = run.status.phase === "pending" || isActive;
  const streamRunId = isWatchable ? run.id : null;
  const { logLines, isStreaming } = useWatchRun(streamRunId, onRefresh ? () => onRefresh() : undefined);

  // Escape closes detail
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        const tag = (e.target as HTMLElement)?.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
        onClose();
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  return (
    <div data-testid="run-detail" className="flex h-full flex-col bg-background">
      {/* Header with breadcrumb and close */}
      <div className="flex items-center justify-between border-b border-border px-5 py-3">
        <div className="flex items-center gap-2 min-w-0">
          <button
            onClick={onClose}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors uppercase tracking-widest whitespace-nowrap"
          >
            Runs
          </button>
          <span className="text-xs text-muted-foreground/40">/</span>
          <span className="text-sm font-semibold text-foreground font-mono fx-glow">
            {run.name}
          </span>
          {run.spec.displayName && (
            <span className="text-xs text-muted-foreground truncate max-w-[200px]">
              {run.spec.displayName}
            </span>
          )}
          <span className="ml-2" data-testid="detail-phase">
            <PhaseBadge phase={run.status.phase} />
          </span>
        </div>
        <div className="flex items-center gap-2">
          {run.status.traceID && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigator.clipboard.writeText(run.status.traceID)}
              className="text-xs"
            >
              Copy Trace
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onClone(run)}
            className="text-xs"
          >
            Clone
          </Button>
          {isActive && (
            <Button
              data-testid="detail-cancel"
              variant="destructive"
              size="sm"
              onClick={() => onCancel(run.id)}
              className="text-xs"
            >
              Cancel
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label="Close"
            className="text-lg leading-none"
          >
            &times;
          </Button>
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-0 border-b border-border bg-background px-2">
        {TABS.map((tab) => {
          const active = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              data-testid={tab.testId}
              onClick={() => setActiveTab(tab.id)}
              className={`relative px-4 py-2 text-sm font-medium transition-colors ${
                active
                  ? "text-foreground"
                  : "text-muted-foreground/60 hover:text-muted-foreground"
              }`}
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
          <>
            {hasOrchestration && (
              <RunGraphSection run={run} />
            )}
            <InfoTab
              run={run}
              humanInput={humanInput}
              setHumanInput={setHumanInput}
              onSendInput={onSendInput}
            />
          </>
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
    <div className="flex-1 overflow-y-auto p-6">
      {/* Name & badges */}
      <h2 data-testid="detail-name" className="mb-2 text-lg font-semibold fx-glow">
        <span className="font-mono">{run.name}</span>
        {run.spec.displayName && (
          <span className="ml-2 text-sm text-muted-foreground font-sans">{run.spec.displayName}</span>
        )}
      </h2>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <PhaseBadge phase={run.status.phase} />
        <BackendBadge backend={run.spec.backend} />
        <ModelTierBadge tier={run.spec.modelTier} />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          {/* Metadata */}
          <div className="space-y-2">
            <MetaRow label="Status" value={run.status.phase} />
            <MetaRow label="Duration" value={duration()} />
            <MetaRow label="Created" value={formatTime(run.createdAt)} />
            <MetaRow label="Started" value={formatTime(run.status.startedAt)} />
            {run.status.completedAt && (
              <MetaRow label="Completed" value={formatTime(run.status.completedAt)} />
            )}
            {run.spec.workspaceName && (
              <MetaRow label="Workspace" value={run.spec.workspaceName} />
            )}
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

          {/* Repositories */}
          <div data-testid="detail-repos">
            <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
              Repositories
            </h3>
            {run.spec.repos.length > 0 ? (
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
            ) : (
              <span className="text-xs text-muted-foreground/60">&mdash;</span>
            )}
          </div>
        </div>

        <div className="space-y-4">
          {/* Prompt */}
          <div>
            <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
              Prompt
            </h3>
            <div className="border border-border bg-muted p-3 text-sm text-muted-foreground whitespace-pre-wrap max-h-64 overflow-y-auto">
              {run.spec.prompt}
            </div>
          </div>

          {/* Spec */}
          {run.spec.specContent && (
            <div>
              <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
                Spec
              </h3>
              <SpecEditor value={run.spec.specContent} readOnly height="200px" />
              {run.spec.specSource && (
                <div className="mt-1 text-xs text-muted-foreground/60">
                  Source: {run.spec.specSource}
                </div>
              )}
            </div>
          )}

          {/* Env vars */}
          {Object.keys(run.spec.envVars).length > 0 && (
            <div>
              <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
                Environment
              </h3>
              <div className="space-y-1">
                {Object.entries(run.spec.envVars).map(([k, v]) => (
                  <div key={k} className="font-mono text-xs">
                    <span className="text-muted-foreground/60">{k}</span>
                    <span className="text-muted-foreground">=</span>
                    <span className="text-foreground">{v}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Status message */}
          {run.status.message && (
            <div>
              <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
                Status Message
              </h3>
              <div
                className={`border p-3 text-sm whitespace-pre-wrap ${
                  run.status.phase === "failed"
                    ? "border-destructive/30 bg-destructive/5 text-destructive"
                    : run.status.phase === "waiting_for_input"
                    ? "border-secondary/30 bg-secondary/5 text-secondary"
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
              <h3 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
                Human Input
              </h3>
              <textarea
                data-testid="detail-hitl-input"
                className="w-full border border-input bg-background px-3 py-2 text-sm text-foreground placeholder-muted-foreground/60 outline-none transition-colors focus:border-primary min-h-[80px] resize-y"
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
                onClick={() => {
                  if (humanInput.trim()) {
                    onSendInput(run.id, humanInput.trim());
                    setHumanInput("");
                  }
                }}
                disabled={!humanInput.trim()}
                className="mt-2 text-sm disabled:opacity-40"
              >
                Send Input
              </Button>
            </div>
          )}
        </div>
      </div>
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
    apiFetch(`/api/v1/runs/${run.id}/logs`)
      .then((r) => r.ok ? r.text() : "")
      .then((text) => {
        if (!cancelled && text) setFetchedLines(text.split("\n"));
      })
      .catch(() => {})
      .finally(() => { if (!cancelled) setFetching(false); });
    return () => { cancelled = true; };
  }, [run.id, hasPod, isStreaming, logLines.length]);

  const persistedLines = run.status.logOutput
    ? run.status.logOutput.split("\n")
    : [];

  const lines = logLines.length > 0 ? logLines : fetchedLines.length > 0 ? fetchedLines : persistedLines;
  const streaming = isStreaming && logLines.length > 0;

  if (lines.length === 0 && !isStreaming && !fetching) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
        No logs available
      </div>
    );
  }

  if (fetching && lines.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
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
      await apiFetch(`/api/v1/runs/${run.id}/debug`, { method: "POST" });
    } catch (err) {
      console.error("Failed to start debug:", err);
    } finally {
      setDebugLoading(false);
    }
  }

  async function handleDebugStop() {
    setDebugLoading(true);
    try {
      await apiFetch(`/api/v1/runs/${run.id}/debug`, { method: "DELETE" });
    } catch (err) {
      console.error("Failed to stop debug:", err);
    } finally {
      setDebugLoading(false);
    }
  }

  if (!isActive && !debugActive) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-8">
        <p className="text-sm text-muted-foreground/60 text-center">
          The agent run has completed. Start a debug session to get shell access to the workspace.
        </p>
        <Button
          data-testid="debug-run-btn"
          onClick={handleDebugStart}
          disabled={debugLoading}
          className="text-sm disabled:opacity-40"
        >
          {debugLoading ? "Starting..." : "Debug Run"}
        </Button>
        <VSCodeConnectInfo podName={run.status.podName} />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {debugActive && !isActive && (
        <div className="flex items-center justify-between border-b border-border px-3 py-2">
          <span className="text-xs text-secondary font-medium">Debug session active</span>
          <Button
            data-testid="stop-debug-btn"
            variant="destructive"
            size="sm"
            onClick={handleDebugStop}
            disabled={debugLoading}
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
        <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground/60">
          Waiting for pod to become ready...
        </div>
      )}
      <VSCodeConnectInfo podName={run.status.podName} />
    </div>
  );
}

/* -- VS Code Connection Info -- */

function VSCodeConnectInfo({ podName }: { podName: string }) {
  const [expanded, setExpanded] = useState(false);
  if (!podName) return null;

  const portForwardCmd = `kubectl port-forward pod/${podName} 50052:50052 -n aot-local`;

  return (
    <div className="w-full max-w-md">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors"
      >
        <span className={`transition-transform ${expanded ? "rotate-90" : ""}`}>&#9654;</span>
        VS Code Connection
      </button>
      {expanded && (
        <div className="mt-2 border border-border bg-muted p-3">
          <p className="mb-2 text-xs text-muted-foreground/60">
            Connect VS Code Remote Containers to the workspace:
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-background px-2 py-1 font-mono text-xs text-muted-foreground break-all">
              {portForwardCmd}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigator.clipboard.writeText(portForwardCmd)}
              className="text-xs flex-shrink-0"
            >
              Copy
            </Button>
          </div>
          <p className="mt-2 text-xs text-muted-foreground/60">
            Then attach to the container using VS Code &quot;Attach to Running Container&quot;.
          </p>
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
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
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
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
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
          <h4 className="mb-1 text-xs font-medium uppercase tracking-widest text-muted-foreground/60">
            Metadata
          </h4>
          <pre className="border border-border bg-muted p-3 text-xs font-mono text-muted-foreground overflow-x-auto whitespace-pre-wrap">
            {JSON.stringify(span.metadata, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

/* -- RunGraphSection -- */

function RunGraphSection({ run }: { run: AgentRun }) {
  // Build a simple graph from the run's children and parent info
  const nodes: RunGraphNode[] = [];
  const edges: RunGraphEdge[] = [];

  // Add this run as a node
  const thisRole = run.spec.parentRunId ? "junior" : (run.children && run.children.length > 0 ? "senior" : "single");
  nodes.push({
    name: run.name,
    phase: run.status.phase,
    role: thisRole,
    startedAt: run.status.startedAt,
    completedAt: run.status.completedAt,
  });

  // Add parent edge if this is a junior
  if (run.spec.parentRunId) {
    edges.push({ parent: run.spec.parentRunId, child: run.name });
    // Add parent node placeholder
    nodes.push({
      name: run.spec.parentRunId,
      phase: "running", // We don't have the parent's actual phase
      role: "senior",
    });
  }

  // Add children
  if (run.children) {
    for (const childName of run.children) {
      nodes.push({
        name: childName,
        phase: "pending",
        role: "junior",
      });
      edges.push({ parent: run.name, child: childName });
    }
  }

  return (
    <div className="border-b border-border">
      <RunGraph nodes={nodes} edges={edges} />
    </div>
  );
}

/* -- MetaRow -- */

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <span className="text-xs text-muted-foreground/60 whitespace-nowrap">{label}</span>
      <span
        className={`text-sm text-muted-foreground text-right truncate ${mono ? "font-mono text-xs" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}
