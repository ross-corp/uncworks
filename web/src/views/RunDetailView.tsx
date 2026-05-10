import React, { useState, useEffect, useCallback, useRef } from "react";
import { usePoll } from "../hooks/usePoll";
import { useCopilotContext } from "../hooks/useCopilotContext";
import { ScrollTextIcon, GitBranchIcon, FolderIcon, TerminalIcon } from "lucide-react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { toast } from "sonner";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import RunStatusBadge from "../components/RunStatusBadge";
import ActivityFeed from "../components/ActivityFeed";
import StageProgress from "../components/StageProgress";
import FileExplorer from "../components/FileExplorer";
import ShellTerminal from "../components/ShellTerminal";
import TraceTimeline, { SpanDetail } from "../components/TraceTimeline";
import FailureDiagnosisPanel from "../components/FailureDiagnosisPanel";
import HitlModal from "../components/HitlModal";
import { useTraces } from "../hooks/useTraces";
import type { TraceSpan } from "../types/agent-run";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "../components/ui/sheet";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel,
  AlertDialogContent, AlertDialogDescription, AlertDialogFooter,
  AlertDialogHeader, AlertDialogTitle,
} from "../components/ui/alert-dialog";

type Tab = "logs" | "traces" | "files" | "shell";

const ACTIVE_PHASES = new Set(["running", "planning", "verifying", "waiting_for_input"]);

function formatElapsed(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const mins = Math.floor(totalSec / 60);
  const secs = totalSec % 60;
  if (mins > 0) return `${mins}m ${secs}s`;
  return `${secs}s`;
}

// ── Horizontal phase step indicator ──────────────────────────────────────────

type StepStatus = "done" | "active" | "failed" | "future";

const PHASE_STEPS = [
  { key: "planning", label: "Planning" },
  { key: "executing", label: "Executing" },
  { key: "verifying", label: "Verifying" },
] as const;

function getStepStatus(stepKey: string, stage: string | undefined, phase: string | undefined): StepStatus {
  const order = ["planning", "executing", "verifying"];
  const stepIdx = order.indexOf(stepKey);
  let currentIdx = -1;
  if (stage === "planning") currentIdx = 0;
  else if (stage === "executing") currentIdx = 1;
  else if (stage === "verifying") currentIdx = 2;

  if (phase === "succeeded") return "done";
  if (phase === "failed" || phase === "cancelled") {
    if (stepIdx < currentIdx) return "done";
    if (stepIdx === currentIdx) return "failed";
    return "future";
  }
  if (stepIdx < currentIdx) return "done";
  if (stepIdx === currentIdx) return "active";
  return "future";
}

function PhaseStepRow({ stage, phase }: { stage?: string; phase?: string }) {
  return (
    <div className="flex items-center gap-1">
      {PHASE_STEPS.map((step, i) => {
        const status = getStepStatus(step.key, stage, phase);
        return (
          <div key={step.key} className="flex items-center gap-1">
            {i > 0 && <div className="w-4 h-px bg-border" />}
            <div className="flex items-center gap-1">
              {status === "done" && <span className="text-green-500 text-xs leading-none">✓</span>}
              {status === "failed" && <span className="text-red-500 text-xs leading-none">✗</span>}
              {status === "active" && (
                <span className="relative flex size-1.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
                  <span className="relative inline-flex rounded-full size-1.5 bg-blue-500" />
                </span>
              )}
              {status === "future" && (
                <span className="size-1.5 rounded-full border border-muted-foreground/40 inline-block" />
              )}
              <span className={`text-xs ${
                status === "active" ? "text-blue-500 font-medium"
                : status === "done" ? "text-green-500"
                : status === "failed" ? "text-red-500"
                : "text-muted-foreground/60"
              }`}>{step.label}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────

const TABS: { key: Tab; label: string; num: string; icon: React.ElementType }[] = [
  { key: "logs", label: "Logs", num: "1", icon: ScrollTextIcon },
  { key: "traces", label: "Traces", num: "2", icon: GitBranchIcon },
  { key: "files", label: "Files", num: "3", icon: FolderIcon },
  { key: "shell", label: "Shell", num: "4", icon: TerminalIcon },
];

export default function RunDetailView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const client = useClient();
  const [run, setRun] = useState<AgentRun | null>(null);
  const [tab, setTab] = useState<Tab>("logs");
  const [showInfo, setShowInfo] = useState(false);
  const [idCopied, setIdCopied] = useState(false);
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | null>(null);
  const [hitlInput, setHitlInput] = useState("");
  const [hitlModalOpen, setHitlModalOpen] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const [pendingArchive, setPendingArchive] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const prevPhaseRef = useRef<string | undefined>(undefined);
  const retriesRef = useRef(0);
  const { spans, loading: tracesLoading } = useTraces(id || "", run?.status.phase);

  // Register run context for the global CopilotPanel.
  useCopilotContext(
    run
      ? { type: "run", content: `${run.status.phase}: ${run.spec.prompt}`, label: run.name }
      : null
  );

  // Live elapsed counter
  useEffect(() => {
    if (!run) return;
    const isActive = ACTIVE_PHASES.has(run.status.phase);
    if (!isActive) {
      if (run.status.completedAt && run.createdAt) {
        setElapsed(new Date(run.status.completedAt).getTime() - new Date(run.createdAt).getTime());
      }
      return;
    }
    const start = run.createdAt ? new Date(run.createdAt).getTime() : Date.now();
    const tick = () => setElapsed(Date.now() - start);
    tick();
    const interval = setInterval(tick, 1000);
    return () => clearInterval(interval);
  }, [run?.status.phase, run?.createdAt, run?.status.completedAt]);

  // Auto-open HITL modal when phase transitions to waiting_for_input
  useEffect(() => {
    if (!run) return;
    const phase = run.status.phase;
    if (phase === "waiting_for_input" && prevPhaseRef.current !== "waiting_for_input") {
      setHitlModalOpen(true);
    }
    prevPhaseRef.current = phase;
  }, [run?.status.phase]);

  const elapsedStr = elapsed > 0 ? formatElapsed(elapsed) : "";

  const sendHitl = useCallback(async (input?: string) => {
    const value = input ?? hitlInput;
    if (!id || !value.trim()) return;
    try {
      await client.sendHumanInput(id, value.trim());
      setHitlInput("");
      setHitlModalOpen(false);
      toast.success("Input sent · Resuming run");
    } catch {
      toast.error("Failed to send input");
    }
  }, [id, hitlInput, client]);

  const cancelRun = useCallback(async () => {
    if (!id) return;
    try {
      await client.cancelAgentRun(id);
      toast.success("Run cancelled");
    } catch {
      toast.error("Failed to cancel run");
    }
  }, [id, client]);

  // Reset volatile state when navigating to a different run
  useEffect(() => {
    setRun(null);
    setLoadError(false);
    setSelectedSpan(null);
    setHitlInput("");
    setHitlModalOpen(false);
    setElapsed(0);
    setPendingArchive(false);
    prevPhaseRef.current = undefined;
    retriesRef.current = 0;
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch run and poll
  usePoll(async () => {
    if (!id) return;
    try {
      const result = await client.getAgentRun(id);
      setRun(mapRun(result));
      setLoadError(false);
      retriesRef.current = 0;
    } catch {
      retriesRef.current++;
      if (retriesRef.current >= 3) setLoadError(true);
    }
  }, 5000, [id, client]);

  // Keyboard: esc back, number keys for tabs, i for info
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      // Don't intercept keys when typing in an input
      const target = e.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;

      if (e.key === "Escape") {
        if (selectedSpan) {
          setSelectedSpan(null);
        } else if (showInfo) {
          setShowInfo(false);
        } else {
          navigate("/");
        }
        return;
      }
      const num = parseInt(e.key);
      if (num >= 1 && num <= TABS.length) {
        setTab(TABS[num - 1].key);
      }
      if (e.key === "i") {
        setShowInfo((s) => !s);
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [navigate, showInfo, selectedSpan]);

  if (!run) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        {loadError ? (
          <div className="text-center">
            <div>Run not found</div>
            <Button size="sm" variant="ghost" className="mt-2" onClick={() => navigate("/")}>
              Back to runs
            </Button>
          </div>
        ) : (
          "Loading..."
        )}
      </div>
    );
  }

  const isSpecDriven = run.spec.orchestrationMode === "spec-driven";
  const isRunning = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  const isWaiting = run.status.phase === "waiting_for_input";
  const hitlPrompt = run.status.message || "The agent is requesting your input to continue.";

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2 justify-between">
        <div className="flex flex-col gap-0.5 flex-1 min-w-0">
          <div className="text-xs text-muted-foreground">
            <Link to="/" className="hover:text-foreground transition-colors">Runs</Link>
            {" / "}
            <span>{run.spec.displayName || run.name}</span>
          </div>
          <div className="flex items-center gap-3">
          <span className="font-semibold">{run.spec.displayName || run.name}</span>
          <RunStatusBadge phase={run.status.phase} stage={run.status.stage} />
          {/* Waiting for input amber badge — task 7.3 */}
          {isWaiting && (
            <button
              onClick={() => setHitlModalOpen(true)}
              className="px-2 py-0.5 text-xs rounded bg-amber-500/20 text-amber-400 border border-amber-500/40 hover:bg-amber-500/30 transition-colors animate-pulse"
            >
              Waiting for Input
            </button>
          )}
          {/* Horizontal phase step indicator — task 6.1 */}
          {isSpecDriven && (
            <PhaseStepRow stage={run.status.stage} phase={run.status.phase} />
          )}
          {/* Live elapsed counter — task 6.2; also shown for completed runs */}
          {elapsedStr && (
            <span className="text-xs text-muted-foreground font-mono">{elapsedStr}</span>
          )}
          {/* Cost and diff stats — only populated after run completes */}
          {run.status.totalCost && (
            <span className="text-xs text-muted-foreground">{run.status.totalCost}</span>
          )}
          {(run.status.totalAdditions || run.status.totalDeletions) ? (
            <span className="text-xs font-mono">
              <span className="text-green-600 dark:text-green-400">+{run.status.totalAdditions ?? 0}</span>
              <span className="text-red-600 dark:text-red-400 ml-1">-{run.status.totalDeletions ?? 0}</span>
            </span>
          ) : null}
          {run.status.message && !isWaiting && (
            <span
              className={`text-xs truncate max-w-[32rem] ${
                run.status.phase === "failed" ? "text-red-500" : "text-muted-foreground"
              }`}
              title={run.status.message}
            >
              {run.status.message.length > 80
                ? run.status.message.slice(0, 80) + "..."
                : run.status.message}
            </span>
          )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {/* Cancel button - visible when running or waiting */}
          {isRunning && (
            <button
              onClick={cancelRun}
              className="px-2 py-0.5 text-xs bg-red-600 text-white hover:bg-red-700 transition-colors"
            >
              Cancel
            </button>
          )}

          {/* Retry button - visible when failed */}
          {run.status.phase === "failed" && (
            <button
              onClick={() => navigate(`/new?clone=${id}`)}
              className="px-2 py-0.5 text-xs bg-primary text-primary-foreground hover:opacity-90 transition-colors"
            >
              Retry
            </button>
          )}

          {/* Clone button - visible when succeeded or cancelled */}
          {(run.status.phase === "succeeded" || run.status.phase === "cancelled") && (
            <button
              onClick={() => navigate(`/new?clone=${id}`)}
              className="px-2 py-0.5 text-xs border border-primary/20 text-primary hover:bg-primary/5 transition-colors"
            >
              Clone
            </button>
          )}

          {/* View PR button - visible when PR URL exists and run is not archived */}
          {run.status.prUrl && !run.status.archived && (
            <a
              href={run.status.prUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="px-2 py-0.5 text-xs border border-green-500/30 bg-green-500/10 text-green-600 hover:bg-green-500/20 transition-colors"
            >
              View PR
            </a>
          )}

          {/* Archive button - visible when not running */}
          {!isRunning && !run.status.archived && (
            <button
              onClick={() => setPendingArchive(true)}
              className="px-2 py-0.5 text-xs border text-muted-foreground hover:text-foreground transition-colors"
            >
              Archive
            </button>
          )}

          <span className="text-xs text-muted-foreground">1-4 sections · esc close/back · i info</span>
        </div>
      </div>

      {/* CI autofix status bar */}
      {run.status.parentPRUrl && (
        <div className="flex items-center gap-2 border-b bg-orange-500/10 px-4 py-1">
          <span className="text-xs">CI Autofix run targeting </span>
          <a
            href={run.status.parentPRUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-blue-500 hover:underline"
          >
            {run.status.parentPRUrl.split("/").pop()}
          </a>
          {run.status.ciFixAttempts && run.status.ciFixAttempts > 0 && (
            <span className="text-xs text-muted-foreground">(attempt {run.status.ciFixAttempts})</span>
          )}
        </div>
      )}
      {run.status.lastCIStatus && (
        <div className={`flex items-center gap-2 border-b px-4 py-1 ${
          run.status.lastCIStatus === "success" ? "bg-green-500/10" : "bg-red-500/10"
        }`}>
          <span className="text-xs">
            CI: {run.status.lastCIStatus === "success" ? "passing" : "failing"}
          </span>
        </div>
      )}

      {/* Stage progress (spec-driven only) */}
      {isSpecDriven && (
        <div className="border-b px-4 py-1">
          <StageProgress stage={run.status.stage} phase={run.status.phase} />
        </div>
      )}

      {/* Failure diagnosis panel — tasks 6.3/6.4 */}
      {run.status.phase === "failed" && (
        <FailureDiagnosisPanel
          run={run}
          elapsed={elapsedStr}
          onViewTraces={() => setTab("traces")}
          onRetry={() => navigate(`/new?clone=${id}`)}
          onEditRetry={() => navigate(`/new?clone=${id}`)}
          onArchive={() => setPendingArchive(true)}
        />
      )}

      {/* Three-zone layout: sidebar nav + main content + right detail panel */}
      <div className="flex flex-1 min-h-0 overflow-hidden">
        {/* Left sidebar nav */}
        <nav className="flex flex-col items-center gap-1 border-r bg-muted/20 px-1 py-2 w-12 flex-shrink-0">
          {TABS.map((t) => {
            const Icon = t.icon;
            const isActive = tab === t.key;
            return (
              <button
                key={t.key}
                data-testid={`detail-tab-${t.key}`}
                onClick={() => setTab(t.key)}
                title={`${t.label} (${t.num})`}
                className={`flex flex-col items-center gap-0.5 w-10 py-2 rounded transition-colors text-[9px] font-medium ${
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                }`}
              >
                <Icon className="h-4 w-4" />
                <span>{t.label}</span>
              </button>
            );
          })}
        </nav>

        {/* Main content area */}
        <div className="flex-1 min-w-0 min-h-0 overflow-hidden">
          {tab === "logs" && (
            <ActivityFeed runId={run.id} phase={run.status.phase} />
          )}
          {tab === "traces" && (
            <TraceTimeline
              spans={spans}
              loading={tracesLoading}
              runId={run.id}
              selectedSpanId={selectedSpan?.id}
              onSelectSpan={(span) => setSelectedSpan(span)}
            />
          )}
          {tab === "files" && (
            <FileExplorer runId={run.id} phase={run.status.phase} />
          )}
          {tab === "shell" && (
            <ShellTerminal runId={run.id} phase={run.status.phase} />
          )}
        </div>

        {/* Right detail panel — shown when a trace span is selected */}
        {selectedSpan && (
          <div className="flex-shrink-0 w-[360px] border-l border-border bg-background overflow-hidden">
            <SpanDetail
              span={selectedSpan}
              allSpans={spans}
              runId={run.id}
              onClose={() => setSelectedSpan(null)}
            />
          </div>
        )}
      </div>

      {/* HITL inline overlay (fallback when modal is closed) */}
      {run.status.phase === "waiting_for_input" && !hitlModalOpen && (
        <div className="border-t bg-yellow-500/10 px-4 py-2">
          <div className="text-xs text-yellow-500 mb-1">Agent is waiting for input</div>
          <div className="flex gap-2">
            <input
              className="flex-1 border bg-background px-2 py-1 text-sm outline-none focus:border-primary"
              value={hitlInput}
              onChange={(e) => setHitlInput(e.target.value)}
              placeholder="Type your response..."
              onKeyDown={(e) => { if (e.key === "Enter") sendHitl(); }}
              autoFocus
            />
            <button
              onClick={() => sendHitl()}
              className="px-3 py-1 text-xs bg-primary text-primary-foreground"
            >
              Send
            </button>
          </div>
        </div>
      )}

      {/* HITL Modal — tasks 7.1/7.2/7.4 */}
      <HitlModal
        open={hitlModalOpen}
        promptText={hitlPrompt}
        onSubmit={(input) => sendHitl(input)}
        onClose={() => setHitlModalOpen(false)}
      />

      {/* Info Sheet (side panel) */}
      <Sheet open={showInfo} onOpenChange={setShowInfo}>
        <SheetContent side="right">
          <SheetHeader>
            <SheetTitle>Run Details</SheetTitle>
            <SheetDescription>
              {run.spec.displayName || run.name}
            </SheetDescription>
          </SheetHeader>
          <div className="px-4 space-y-4 text-sm">
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground">ID</span>
                <div className="flex items-center gap-1">
                  <span className="font-mono text-xs">{run.id}</span>
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(run.id);
                      setIdCopied(true);
                      setTimeout(() => setIdCopied(false), 1500);
                    }}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors px-1"
                    title="Copy ID"
                  >
                    {idCopied ? "Copied!" : "📋"}
                  </button>
                </div>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <span>{run.createdAt ? new Date(run.createdAt).toLocaleString() : "---"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Model</span>
                <span>{run.spec.modelTier || "default"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Repo</span>
                <span className="text-xs truncate max-w-[200px]">{run.spec.repos?.[0]?.url || "---"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Mode</span>
                <span>{run.spec.orchestrationMode || "single"}</span>
              </div>
              {run.status.stage && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Stage</span>
                  <span>{run.status.stage} (attempt {(run.status.retryCount ?? 0) + 1})</span>
                </div>
              )}
            </div>
            <div className="border-t pt-3">
              <span className="text-muted-foreground text-xs">Prompt</span>
              <p className="mt-1 text-xs">{run.spec.prompt}</p>
            </div>
            {run.status.logOutput && (
              <div className="border-t pt-3">
                <details>
                  <summary className="text-muted-foreground text-xs cursor-pointer select-none">
                    Stored Logs ({run.status.logOutput.split("\n").length} lines)
                  </summary>
                  <pre className="mt-2 text-xs bg-muted/50 rounded p-2 overflow-auto max-h-96 font-mono whitespace-pre-wrap break-all">
                    {run.status.logOutput}
                  </pre>
                </details>
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <AlertDialog open={pendingArchive} onOpenChange={setPendingArchive}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive this run?</AlertDialogTitle>
            <AlertDialogDescription>
              The workspace will be deleted. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={async () => {
              setPendingArchive(false);
              try {
                const resp = await apiFetch(`/api/v1/runs/${id}/archive`, {
                  method: "POST",
                  headers: { "Content-Type": "application/json" },
                  body: JSON.stringify({ archived: true }),
                });
                if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
                toast.success("Run archived");
                navigate("/");
              } catch {
                toast.error("Failed to archive run");
              }
            }}>Archive</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
