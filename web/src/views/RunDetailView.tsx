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
import { useTraces } from "../hooks/useTraces";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../components/ui/tabs";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "../components/ui/sheet";

type Tab = "logs" | "traces" | "files" | "shell";

const TABS: { key: Tab; label: string; num: string }[] = [
  { key: "logs", label: "Logs", num: "1" },
  { key: "traces", label: "Traces", num: "2" },
  { key: "files", label: "Files", num: "3" },
  { key: "shell", label: "Shell", num: "4" },
];

export default function RunDetailView() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const client = useClient();
  const { toast } = useToast();
  const [run, setRun] = useState<AgentRun | null>(null);
  const [tab, setTab] = useState<Tab>("logs");
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

  const cancelRun = useCallback(async () => {
    if (!id) return;
    try {
      await client.cancelAgentRun(id);
      toast("Run cancelled", "success");
    } catch {
      toast("Failed to cancel run", "error");
    }
  }, [id, client, toast]);

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

  // Keyboard: esc back, number keys for tabs, i for info
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      // Don't intercept keys when typing in an input
      const target = e.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;

      if (e.key === "Escape") {
        if (showInfo) {
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
  }, [navigate, showInfo]);

  if (!run) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Loading...
      </div>
    );
  }

  const isSpecDriven = run.spec.orchestrationMode === "spec-driven";
  const isRunning = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  const isFailed = run.status.phase === "failed" || run.status.phase === "cancelled";

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
          {run.status.phase === "failed" && run.status.message && (
            <span className="text-xs text-red-500 truncate max-w-[32rem]" title={run.status.message}>
              {run.status.message.length > 80
                ? run.status.message.slice(0, 80) + "..."
                : run.status.message}
            </span>
          )}
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

          {/* Retry/Clone button - visible when failed or cancelled */}
          {isFailed && (
            <button
              onClick={() => navigate(`/new?clone=${id}`)}
              className="px-2 py-0.5 text-xs bg-primary text-primary-foreground hover:opacity-90 transition-colors"
            >
              Retry
            </button>
          )}

          <span className="text-xs text-muted-foreground">esc back · 1-4 tabs · i info</span>
        </div>
      </div>

      {/* Stage progress (spec-driven only) */}
      {isSpecDriven && (
        <div className="border-b px-4 py-1">
          <StageProgress stage={run.status.stage} phase={run.status.phase} />
        </div>
      )}

      {/* Tabs */}
      <Tabs
        value={tab}
        onValueChange={(v) => setTab(v as Tab)}
        className="flex-1 min-h-0 flex flex-col gap-0"
      >
        <TabsList>
          {TABS.map((t) => (
            <TabsTrigger
              key={t.key}
              value={t.key}
              data-testid={`detail-tab-${t.key}`}
            >
              <span className="opacity-40 mr-1">{t.num}</span>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="logs" className="flex-1 min-h-0 overflow-hidden">
          <ActivityFeed runId={run.id} phase={run.status.phase} />
        </TabsContent>
        <TabsContent value="traces" className="flex-1 min-h-0 overflow-hidden">
          <div className="h-full">
            <TraceTimeline spans={spans} runId={run.id} onSelectSpan={() => {}} />
          </div>
        </TabsContent>
        <TabsContent value="files" className="flex-1 min-h-0 overflow-hidden">
          <FileExplorer runId={run.id} />
        </TabsContent>
        <TabsContent value="shell" className="flex-1 min-h-0 overflow-hidden">
          <ShellTerminal runId={run.id} phase={run.status.phase} />
        </TabsContent>
      </Tabs>

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
              <div className="flex justify-between">
                <span className="text-muted-foreground">ID</span>
                <span className="font-mono text-xs">{run.id}</span>
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
          </div>
        </SheetContent>
      </Sheet>

    </div>
  );
}
