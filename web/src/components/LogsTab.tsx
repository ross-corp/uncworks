import { useState, useEffect } from "react";
import type { AgentRun } from "../types/agent-run";
import { apiFetch } from "../hooks/apiFetch";
import LogViewer from "./LogViewer";
import AgentLogView from "./AgentLogView";

type LogView = "structured" | "raw";

export default function LogsTab({
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
  const [view, setView] = useState<LogView>("structured");
  const [fetchedLines, setFetchedLines] = useState<string[]>([]);
  const [fetching, setFetching] = useState(false);

  useEffect(() => {
    if (!hasPod || isStreaming || logLines.length > 0) return;
    let cancelled = false;
    setFetching(true);
    apiFetch(`/api/v1/runs/${run.id}/logs`)
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
  const rawLines =
    logLines.length > 0 ? logLines : fetchedLines.length > 0 ? fetchedLines : persistedLines;
  const streaming = isStreaming && logLines.length > 0;

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 px-2 py-1 border-b border-border shrink-0">
        <button
          onClick={() => setView("structured")}
          className={`px-2 py-0.5 text-xs font-medium transition-colors ${
            view === "structured"
              ? "bg-muted text-foreground"
              : "text-muted-foreground/60 hover:text-muted-foreground"
          }`}
        >
          Agent
        </button>
        <button
          onClick={() => setView("raw")}
          className={`px-2 py-0.5 text-xs font-medium transition-colors ${
            view === "raw"
              ? "bg-muted text-foreground"
              : "text-muted-foreground/60 hover:text-muted-foreground"
          }`}
        >
          Raw
        </button>
      </div>
      <div className="flex-1 min-h-0">
        {view === "structured" ? (
          <AgentLogView runId={run.id} />
        ) : (
          <div className="h-full p-2">
            {rawLines.length === 0 && !streaming && !fetching ? (
              <div className="flex h-full items-center justify-center text-xs text-muted-foreground/60">
                No raw logs available
              </div>
            ) : fetching && rawLines.length === 0 ? (
              <div className="flex h-full items-center justify-center text-xs text-muted-foreground/60">
                Loading logs...
              </div>
            ) : (
              <LogViewer lines={rawLines} streaming={streaming} />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
