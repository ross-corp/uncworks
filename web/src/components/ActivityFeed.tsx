import { useState, useEffect, useRef } from "react";
import { apiFetch } from "../hooks/apiFetch";

interface LogEntry {
  timestamp: string;
  type: string;
  content: string;
  toolName?: string;
  toolInput?: string;
  model?: string;
}

export default function ActivityFeed({ runId }: { runId: string }) {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function fetch() {
      try {
        const r = await apiFetch(`/api/v1/runs/${runId}/logs/structured`);
        if (r.ok) {
          const data = await r.json();
          if (!cancelled) setEntries(data ?? []);
        }
      } catch {
        // silent
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetch();
    const interval = setInterval(fetch, 3000);
    return () => { cancelled = true; clearInterval(interval); };
  }, [runId]);

  // Auto-scroll
  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [entries, autoScroll]);

  // Detect user scroll
  function handleScroll() {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setAutoScroll(atBottom);
  }

  if (loading && entries.length === 0) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">Loading activity...</div>;
  }

  if (entries.length === 0) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">No activity yet</div>;
  }

  return (
    <div ref={containerRef} onScroll={handleScroll} className="h-full overflow-y-auto p-4 space-y-1">
      {entries.map((entry, i) => (
        <EntryRow key={i} entry={entry} />
      ))}
      <div ref={bottomRef} />
    </div>
  );
}

function EntryRow({ entry }: { entry: LogEntry }) {
  const [expanded, setExpanded] = useState(false);
  const ts = entry.timestamp?.slice(11, 19) || "";

  switch (entry.type) {
    case "user":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className="w-14 shrink-0 text-xs font-medium text-blue-500">user</span>
          <span className="text-sm">{entry.content}</span>
        </div>
      );
    case "assistant":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className="w-14 shrink-0 text-xs font-medium text-green-500">agent</span>
          <span className="text-sm whitespace-pre-wrap">{entry.content}</span>
        </div>
      );
    case "tool_call":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className="w-14 shrink-0 text-xs font-medium text-purple-500">tool</span>
          <div className="min-w-0">
            <button onClick={() => setExpanded(!expanded)} className="text-sm text-purple-400 hover:text-purple-300">
              {expanded ? "▼" : "▶"} {entry.toolName}
            </button>
            {expanded && entry.toolInput && (
              <pre className="mt-1 p-2 bg-muted text-xs overflow-x-auto rounded">{formatJSON(entry.toolInput)}</pre>
            )}
          </div>
        </div>
      );
    case "tool_result":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className="w-14 shrink-0 text-xs text-orange-400">result</span>
          <ToolResult content={entry.content} toolName={entry.toolName} />
        </div>
      );
    case "system":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className="w-14 shrink-0 text-xs text-yellow-500">system</span>
          <span className="text-xs italic text-muted-foreground">{entry.content}</span>
        </div>
      );
    default:
      return null;
  }
}

function ToolResult({ content }: { content: string; toolName?: string }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = content.length > 200;

  if (!isLong) {
    return <pre className="text-xs text-muted-foreground whitespace-pre-wrap break-all">{content}</pre>;
  }

  return (
    <div className="min-w-0">
      <button onClick={() => setExpanded(!expanded)} className="text-xs text-muted-foreground hover:text-foreground">
        {expanded ? "▼ collapse" : "▶ expand"} ({content.length} chars)
      </button>
      <pre className={`mt-1 text-xs text-muted-foreground whitespace-pre-wrap break-all ${expanded ? "" : "line-clamp-3"}`}>
        {content}
      </pre>
    </div>
  );
}

function formatJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2); } catch { return s; }
}
