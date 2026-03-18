import { useState, useEffect, useRef } from "react";
import { apiFetch } from "../hooks/apiFetch";

interface AgentLogEntry {
  timestamp: string;
  type: "user" | "assistant" | "tool_call" | "tool_result" | "system";
  content: string;
  toolName?: string;
  toolInput?: string;
  model?: string;
}

export default function AgentLogView({ runId }: { runId: string }) {
  const [entries, setEntries] = useState<AgentLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    apiFetch(`/api/v1/runs/${runId}/logs/structured`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((data: AgentLogEntry[]) => {
        if (!cancelled) setEntries(data ?? []);
      })
      .catch((err) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => { cancelled = true; };
  }, [runId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [entries]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
        Loading agent log...
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
        Failed to load agent log: {error}
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
        No agent activity recorded
      </div>
    );
  }

  return (
    <div
      data-testid="agent-log-view"
      className="h-full overflow-y-auto font-mono text-xs"
      style={{ background: "var(--unc-bg)" }}
    >
      <div className="flex flex-col gap-0.5 p-2">
        {entries.map((entry, i) => (
          <LogEntry key={i} entry={entry} />
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

function LogEntry({ entry }: { entry: AgentLogEntry }) {
  const ts = formatTime(entry.timestamp);

  switch (entry.type) {
    case "user":
      return (
        <div className="flex gap-2 py-1 border-b border-border/30">
          <span className="shrink-0 text-muted-foreground/50 w-16">{ts}</span>
          <span className="shrink-0 font-semibold text-blue-400 w-20">user</span>
          <span className="text-foreground whitespace-pre-wrap break-words min-w-0">{entry.content}</span>
        </div>
      );

    case "assistant":
      return (
        <div className="flex gap-2 py-1 border-b border-border/30">
          <span className="shrink-0 text-muted-foreground/50 w-16">{ts}</span>
          <span className="shrink-0 font-semibold text-green-400 w-20">
            agent
            {entry.model && (
              <span className="ml-1 text-muted-foreground/40 font-normal text-[10px]">
                {entry.model}
              </span>
            )}
          </span>
          <span className="text-foreground whitespace-pre-wrap break-words min-w-0">{entry.content}</span>
        </div>
      );

    case "tool_call":
      return (
        <ToolCallEntry entry={entry} ts={ts} />
      );

    case "tool_result":
      return (
        <ToolResultEntry entry={entry} ts={ts} />
      );

    case "system":
      return (
        <div className="flex gap-2 py-1 border-b border-border/30">
          <span className="shrink-0 text-muted-foreground/50 w-16">{ts}</span>
          <span className="shrink-0 font-semibold text-yellow-500 w-20">system</span>
          <span className="text-muted-foreground italic">{entry.content}</span>
        </div>
      );

    default:
      return null;
  }
}

function ToolCallEntry({ entry, ts }: { entry: AgentLogEntry; ts: string }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex gap-2 py-1 border-b border-border/30">
      <span className="shrink-0 text-muted-foreground/50 w-16">{ts}</span>
      <span className="shrink-0 font-semibold text-purple-400 w-20">tool</span>
      <div className="min-w-0">
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-purple-300 hover:text-purple-200 transition-colors"
        >
          {expanded ? "▼" : "▶"} {entry.toolName}
        </button>
        {expanded && entry.toolInput && (
          <pre className="mt-1 p-2 bg-muted/30 text-muted-foreground text-[11px] overflow-x-auto whitespace-pre-wrap break-all">
            {formatJSON(entry.toolInput)}
          </pre>
        )}
      </div>
    </div>
  );
}

function ToolResultEntry({ entry, ts }: { entry: AgentLogEntry; ts: string }) {
  const [expanded, setExpanded] = useState(false);
  const content = entry.content || "";
  const isLong = content.length > 200;
  const preview = isLong ? content.slice(0, 200) + "..." : content;

  return (
    <div className="flex gap-2 py-1 border-b border-border/30">
      <span className="shrink-0 text-muted-foreground/50 w-16">{ts}</span>
      <span className="shrink-0 text-orange-400 w-20">
        {entry.toolName ? `← ${entry.toolName}` : "← result"}
      </span>
      <div className="min-w-0">
        {isLong ? (
          <>
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              {expanded ? "▼ collapse" : "▶ expand"} ({content.length} chars)
            </button>
            {expanded && (
              <pre className="mt-1 p-2 bg-muted/30 text-muted-foreground text-[11px] overflow-x-auto whitespace-pre-wrap break-all max-h-60 overflow-y-auto">
                {content}
              </pre>
            )}
            {!expanded && (
              <pre className="mt-1 text-muted-foreground/60 text-[11px] whitespace-pre-wrap break-all">
                {preview}
              </pre>
            )}
          </>
        ) : (
          <pre className="text-muted-foreground text-[11px] whitespace-pre-wrap break-all">
            {content}
          </pre>
        )}
      </div>
    </div>
  );
}

function formatTime(timestamp: string): string {
  if (!timestamp) return "";
  try {
    const d = new Date(timestamp);
    return d.toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" });
  } catch {
    return "";
  }
}

function formatJSON(input: string): string {
  try {
    return JSON.stringify(JSON.parse(input), null, 2);
  } catch {
    return input;
  }
}
