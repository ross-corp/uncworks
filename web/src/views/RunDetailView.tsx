import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
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
  const [run, setRun] = useState<AgentRun | null>(null);
  const [tab, setTab] = useState<Tab>("activity");
  const { spans } = useTraces(id || "");

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
        // TODO: toggle info overlay
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

      {/* Footer */}
      <div className="border-t px-4 py-1 text-xs text-muted-foreground">
        1 activity · 2 files · 3 shell · 4 traces · 5 verify · esc back
      </div>
    </div>
  );
}
