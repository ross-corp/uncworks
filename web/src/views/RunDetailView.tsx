import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { useToast } from "../components/Toast";
import RunStatusBadge from "../components/RunStatusBadge";
import ActivityFeed from "../components/ActivityFeed";
import StageProgress from "../components/StageProgress";
import FileExplorer from "../components/FileExplorer";
import ShellTerminal from "../components/ShellTerminal";
import TraceTimeline from "../components/TraceTimeline";
import VerificationPanel from "../components/VerificationPanel";
import { useTraces } from "../hooks/useTraces";

type Tab = "activity" | "files" | "shell" | "traces" | "verify";

const TABS: { key: Tab; label: string; num: string }[] = [
  { key: "activity", label: "Activity", num: "1" },
  { key: "files", label: "Files", num: "2" },
  { key: "shell", label: "Shell", num: "3" },
  { key: "traces", label: "Traces", num: "4" },
  { key: "verify", label: "Verify", num: "5" },
];

export default function RunDetailView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const client = useClient();
  const { toast } = useToast();
  const [run, setRun] = useState<AgentRun | null>(null);
  const [tab, setTab] = useState<Tab>("activity");
  const [showInfo, setShowInfo] = useState(false);
  const [hitlInput, setHitlInput] = useState("");
  const { spans } = useTraces(id || "");

  const sendHitl = useCallback(async () => {
    if (!id || !hitlInput.trim()) return;
    try {
      await client.sendHumanInput(id, hitlInput.trim());
      setHitlInput("");
      toast("Input sent", "success");
    } catch {
      toast("Failed to send input", "error");
    }
  }, [id, hitlInput, client, toast]);

  // Fetch run and poll
  useEffect(() => {
    if (!id) return;
    let cancelled = false;

    async function fetch() {
      try {
        const result = await client.getAgentRun(id!);
        if (!cancelled) setRun(mapRun(result));
      } catch {
        // silent
      }
    }

    fetch();
    const interval = setInterval(fetch, 5000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [id, client]);

  // Keyboard: esc back, number keys for tabs
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        navigate("/");
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
  }, [navigate]);

  if (!run) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Loading...
      </div>
    );
  }

  const isSpecDriven = run.status.stage || run.spec.orchestrationMode === "spec-driven";

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <span className="font-semibold">{run.spec.displayName || run.name}</span>
          <RunStatusBadge phase={run.status.phase} />
          {run.status.stage && (
            <span className="text-xs text-muted-foreground">{run.status.stage}</span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">esc back · 1-5 tabs · i info</span>
      </div>

      {/* Stage progress (spec-driven only) */}
      {isSpecDriven && (
        <div className="border-b px-4 py-1">
          <StageProgress stage={run.status.stage} phase={run.status.phase} />
        </div>
      )}

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b px-4 py-1">
        {TABS.map((t) => (
          <button
            key={t.key}
            data-testid={`detail-tab-${t.key}`}
            onClick={() => setTab(t.key)}
            className={`px-2 py-0.5 text-xs transition-colors ${
              tab === t.key
                ? "bg-accent text-accent-foreground font-medium"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <span className="text-muted-foreground/50 mr-1">{t.num}</span>
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 min-h-0 overflow-hidden">
        {tab === "activity" && <ActivityFeed runId={run.id} />}
        {tab === "files" && <FileExplorer runId={run.id} />}
        {tab === "shell" && <ShellTerminal runId={run.id} />}
        {tab === "traces" && (
          <div className="h-full overflow-y-auto">
            <TraceTimeline spans={spans} onSelectSpan={() => {}} />
          </div>
        )}
        {tab === "verify" && <VerificationPanel runId={run.id} />}
      </div>

      {/* HITL overlay */}
      {run.status.phase === "waiting_for_input" && (
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
              onClick={sendHitl}
              className="px-3 py-1 text-xs bg-primary text-primary-foreground"
            >
              Send
            </button>
          </div>
        </div>
      )}

      {/* Info overlay */}
      {showInfo && (
        <div className="border-t bg-muted/50 px-4 py-3 text-xs space-y-1">
          <div className="flex gap-8">
            <div><span className="text-muted-foreground">ID</span> <span>{run.id}</span></div>
            <div><span className="text-muted-foreground">Created</span> <span>{run.createdAt ? new Date(run.createdAt).toLocaleString() : "—"}</span></div>
            <div><span className="text-muted-foreground">Model</span> <span>{run.spec.modelTier || "default"}</span></div>
          </div>
          <div className="flex gap-8">
            <div><span className="text-muted-foreground">Repo</span> <span>{run.spec.repos?.[0]?.url || "—"}</span></div>
            <div><span className="text-muted-foreground">Mode</span> <span>{run.spec.orchestrationMode || "single"}</span></div>
            {run.status.stage && <div><span className="text-muted-foreground">Stage</span> <span>{run.status.stage} (attempt {(run.status.retryCount ?? 0) + 1})</span></div>}
          </div>
          <div className="text-muted-foreground truncate">Prompt: {run.spec.prompt}</div>
        </div>
      )}

      {/* Footer */}
      <div className="border-t px-4 py-1 text-xs text-muted-foreground">
        1 activity · 2 files · 3 shell · 4 traces · 5 verify · i info · esc back
      </div>
    </div>
  );
}
