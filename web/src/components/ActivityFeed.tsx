import { useState, useEffect, useRef, useMemo } from "react";
import { apiFetch } from "../hooks/apiFetch";
import { usePoll } from "../hooks/usePoll";
import type { AgentRunPhase } from "../types/agent-run";
import { ROLE_STYLES } from "../lib/role-styles";

interface LogEntry {
  timestamp: string;
  type: string;
  content: string;
  toolName?: string;
  toolInput?: string;
  model?: string;
  spanId?: string;
}

/** A processed entry that may have a paired tool_result attached. */
interface DisplayEntry {
  entry: LogEntry;
  /** For tool_call entries, the paired tool_result (if any). */
  pairedResult?: LogEntry;
  /** Synthetic label override: "user" | "manage" | "implement" | "system" | "delegate" */
  label: "user" | "manage" | "implement" | "system" | "delegate";
}

interface ThinkingState {
  thinking: boolean;
  text: string;
  toolName?: string;
}

/**
 * Minimal regex-based markdown renderer.
 * Handles: fenced code blocks, inline code, bold, headers, unordered/ordered lists.
 * Returns an array of React elements.
 */
function renderMarkdown(text: string): React.ReactNode[] {
  const lines = text.split("\n");
  const elements: React.ReactNode[] = [];
  let i = 0;
  let key = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code block
    if (line.trimStart().startsWith("```")) {
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trimStart().startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      elements.push(
        <pre key={key++} className="bg-muted rounded p-2 font-mono text-xs my-1 overflow-x-auto whitespace-pre-wrap">
          {codeLines.join("\n")}
        </pre>
      );
      continue;
    }

    // Headers
    const headerMatch = line.match(/^(#{1,6})\s+(.*)/);
    if (headerMatch) {
      const level = headerMatch[1].length;
      const content = headerMatch[2];
      const sizeClass =
        level === 1 ? "text-lg font-bold" :
        level === 2 ? "text-base font-bold" :
        level === 3 ? "text-sm font-semibold" :
        "text-sm font-medium";
      elements.push(
        <div key={key++} className={`${sizeClass} mt-1`}>{renderInline(content)}</div>
      );
      i++;
      continue;
    }

    // Unordered list items
    if (/^\s*[-*]\s+/.test(line)) {
      const listItems: React.ReactNode[] = [];
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) {
        const itemText = lines[i].replace(/^\s*[-*]\s+/, "");
        listItems.push(<li key={key++} className="ml-4 list-disc">{renderInline(itemText)}</li>);
        i++;
      }
      elements.push(<ul key={key++} className="my-1">{listItems}</ul>);
      continue;
    }

    // Ordered list items
    if (/^\s*\d+\.\s+/.test(line)) {
      const listItems: React.ReactNode[] = [];
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
        const itemText = lines[i].replace(/^\s*\d+\.\s+/, "");
        listItems.push(<li key={key++} className="ml-4 list-decimal">{renderInline(itemText)}</li>);
        i++;
      }
      elements.push(<ol key={key++} className="my-1">{listItems}</ol>);
      continue;
    }

    // Blank line
    if (line.trim() === "") {
      elements.push(<div key={key++} className="h-1" />);
      i++;
      continue;
    }

    // Regular paragraph line
    elements.push(<div key={key++}>{renderInline(line)}</div>);
    i++;
  }

  return elements;
}

/** Render inline markdown: bold, inline code. */
function renderInline(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  // Split on inline code first, then handle bold within non-code segments.
  // Pattern: alternate between `code` and non-code segments.
  const segments = text.split(/(`[^`]+`)/g);
  let key = 0;

  for (const seg of segments) {
    if (seg.startsWith("`") && seg.endsWith("`") && seg.length > 1) {
      parts.push(
        <code key={key++} className="bg-muted px-1 rounded font-mono text-xs">{seg.slice(1, -1)}</code>
      );
    } else {
      // Handle bold within this segment
      const boldParts = seg.split(/(\*\*[^*]+\*\*)/g);
      for (const bp of boldParts) {
        if (bp.startsWith("**") && bp.endsWith("**") && bp.length > 4) {
          parts.push(<strong key={key++} className="font-bold">{bp.slice(2, -2)}</strong>);
        } else if (bp) {
          parts.push(<span key={key++}>{bp}</span>);
        }
      }
    }
  }

  return parts;
}

/** Detect if a string looks like JSON and try to format it. */
function tryFormatAsJSON(s: string): { isJSON: boolean; formatted: string } {
  const trimmed = s.trim();
  if ((trimmed.startsWith("{") && trimmed.endsWith("}")) || (trimmed.startsWith("[") && trimmed.endsWith("]"))) {
    try {
      const parsed = JSON.parse(trimmed);
      return { isJSON: true, formatted: JSON.stringify(parsed, null, 2) };
    } catch {
      // not valid JSON
    }
  }
  return { isJSON: false, formatted: s };
}

export default function ActivityFeed({ runId, phase }: { runId: string; phase?: AgentRunPhase }) {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [thinking, setThinking] = useState<ThinkingState | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [showJumpButton, setShowJumpButton] = useState(false);
  const prevEntryCountRef = useRef(0);

  // Clear stale state when switching to a different run
  useEffect(() => {
    setEntries([]);
    setLoading(true);
    setThinking(null);
    prevEntryCountRef.current = 0;
  }, [runId]);

  // Poll structured logs
  usePoll(async () => {
    try {
      const r = await apiFetch(`/api/v1/runs/${runId}/logs/structured`);
      if (r.ok) {
        const data = await r.json();
        setEntries(data ?? []);
      }
    } catch {
      // silent
    } finally {
      setLoading(false);
    }
  }, 3000, [runId]);

  // Clear thinking when new completed entries arrive
  useEffect(() => {
    if (entries.length > prevEntryCountRef.current) {
      setThinking(null);
    }
    prevEntryCountRef.current = entries.length;
  }, [entries]);

  // Poll thinking endpoint (only when run is active)
  const isActive = phase === "running" || phase === "waiting_for_input";

  useEffect(() => {
    if (!isActive) {
      setThinking(null);
    }
  }, [isActive]);

  usePoll(async () => {
    if (!isActive) return;
    try {
      const r = await apiFetch(`/api/v1/runs/${runId}/logs/thinking`);
      if (r.ok) {
        const data: ThinkingState = await r.json();
        setThinking(data.thinking ? data : null);
      }
    } catch {
      // silent
    }
  }, 2000, [isActive, runId]);

  // Auto-scroll
  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [entries, thinking, autoScroll]);

  // Detect user scroll
  function handleScroll() {
    const el = containerRef.current;
    if (!el) return;
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
    setAutoScroll(distFromBottom < 50);
    setShowJumpButton(distFromBottom > 100);
  }

  function scrollToBottom() {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }

  // Pre-process entries: pair tool_call with tool_result, split user "---" entries
  const displayEntries = useMemo(() => buildDisplayEntries(entries), [entries]);

  if (loading && entries.length === 0) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">Loading activity...</div>;
  }

  if (entries.length === 0) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">No activity yet</div>;
  }

  return (
    <div className="relative h-full">
      <div ref={containerRef} onScroll={handleScroll} className="h-full overflow-y-auto overscroll-none p-4 space-y-1">
        {displayEntries.map((de, i) => (
          <EntryRow key={i} display={de} />
        ))}
        {thinking?.thinking && (thinking.text || thinking.toolName) && (
          <ThinkingEntry text={thinking.text} toolName={thinking.toolName} />
        )}
        <div ref={bottomRef} />
      </div>
      {showJumpButton && (
        <button
          onClick={scrollToBottom}
          className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium shadow-lg hover:bg-primary/90 transition-colors"
        >
          ↓ Jump to latest
        </button>
      )}
    </div>
  );
}

/** Build display entries: pair tool_call/tool_result, split user "---" prompts. */
function buildDisplayEntries(entries: LogEntry[]): DisplayEntry[] {
  const result: DisplayEntry[] = [];
  // Track which tool_result indices have been consumed by a preceding tool_call
  const consumedResults = new Set<number>();

  // First pass: find tool_call -> tool_result pairs.
  // A tool_result is paired with the closest preceding tool_call that shares the same toolName
  // (or simply the immediately preceding tool_call if names are not available).
  for (let i = 0; i < entries.length; i++) {
    if (entries[i].type === "tool_call") {
      // Look ahead for the next tool_result
      for (let j = i + 1; j < entries.length; j++) {
        if (entries[j].type === "tool_result") {
          if (!consumedResults.has(j)) {
            consumedResults.add(j);
            // We'll store the pairing when we build display entries below
            break;
          }
        }
        // Stop looking if we hit another tool_call (it would claim the next result)
        if (entries[j].type === "tool_call") break;
      }
    }
  }

  // Second pass: build display entries
  // Reset consumed tracking to re-pair during construction
  consumedResults.clear();

  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i];

    if (entry.type === "tool_result" && consumedResults.has(i)) {
      // Already attached to a tool_call; skip
      continue;
    }

    if (entry.type === "user") {
      // Check for "---" separator to split user prompt from manage-injected instructions
      const separatorIdx = entry.content.indexOf("\n---\n");
      if (separatorIdx !== -1) {
        const userPart = entry.content.slice(0, separatorIdx).trim();
        const managePart = entry.content.slice(separatorIdx + 5).trim(); // skip "\n---\n"
        if (userPart) {
          result.push({
            entry: { ...entry, content: userPart },
            label: "user",
          });
        }
        if (managePart) {
          result.push({
            entry: { ...entry, content: managePart },
            label: "manage",
          });
        }
        if (!userPart && !managePart) {
          result.push({ entry, label: "user" });
        }
      } else {
        result.push({ entry, label: "user" });
      }
    } else if (entry.type === "delegate") {
      // Delegate entries: find the paired tool_result (delegate_task response)
      let pairedResult: LogEntry | undefined;
      for (let j = i + 1; j < entries.length; j++) {
        if (entries[j].type === "tool_result" && !consumedResults.has(j)) {
          pairedResult = entries[j];
          consumedResults.add(j);
          break;
        }
        if (entries[j].type === "tool_call" || entries[j].type === "delegate") break;
      }
      result.push({ entry, pairedResult, label: "delegate" });
    } else if (entry.type === "tool_call") {
      // Find paired result
      let pairedResult: LogEntry | undefined;
      for (let j = i + 1; j < entries.length; j++) {
        if (entries[j].type === "tool_result" && !consumedResults.has(j)) {
          pairedResult = entries[j];
          consumedResults.add(j);
          break;
        }
        if (entries[j].type === "tool_call" || entries[j].type === "delegate") break;
      }
      result.push({ entry, pairedResult, label: "implement" });
    } else if (entry.type === "tool_result") {
      // Orphaned tool_result (no preceding tool_call matched)
      result.push({ entry, label: "implement" });
    } else if (entry.type === "assistant") {
      result.push({ entry, label: "implement" });
    } else if (entry.type === "system") {
      result.push({ entry, label: "system" });
    } else {
      result.push({ entry, label: "system" });
    }
  }

  return result;
}

function ThinkingEntry({ text, toolName }: { text: string; toolName?: string }) {
  return (
    <div className="flex gap-3 py-1">
      <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">
        <span className={`animate-pulse ${ROLE_STYLES.implement.text}`}>--</span>
      </span>
      <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.implement.text} opacity-50`}>implement</span>
      <span className="text-sm italic text-muted-foreground/50 whitespace-pre-wrap">
        {toolName && <span className={`${ROLE_STYLES.implement.text} opacity-50`}>[{toolName}] </span>}
        {text}
      </span>
    </div>
  );
}

/** Span ID badge -- clickable, could link to traces tab. */
function SpanBadge({ spanId }: { spanId: string }) {
  return (
    <button
      title={`Trace span: ${spanId}`}
      onClick={() => {
        // Dispatch a custom event that a parent (e.g. layout with tabs) can listen for
        // to switch to the traces tab and highlight this span.
        window.dispatchEvent(new CustomEvent("navigate-trace", { detail: { spanId } }));
      }}
      className="inline-flex items-center ml-2 px-1.5 py-0.5 rounded text-[10px] font-mono bg-purple-500/15 text-purple-400 hover:bg-purple-500/25 hover:text-purple-300 transition-colors cursor-pointer"
    >
      span:{spanId.slice(0, 8)}
    </button>
  );
}

/** Expandable markdown content for long text entries. */
function ExpandableContent({ content, className }: { content: string; className?: string }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = content.length > 200;

  if (!isLong) {
    return <span className={`text-sm whitespace-pre-wrap ${className ?? ""}`}>{content}</span>;
  }

  if (!expanded) {
    return (
      <span className={`text-sm ${className ?? ""}`}>
        <span className="whitespace-pre-wrap">{truncate(content, 100)}</span>
        <button
          onClick={() => setExpanded(true)}
          className="ml-1 text-xs text-blue-400 hover:text-blue-300"
        >
          [show more]
        </button>
      </span>
    );
  }

  return (
    <div className={`text-sm min-w-0 ${className ?? ""}`}>
      <button
        onClick={() => setExpanded(false)}
        className="text-xs text-blue-400 hover:text-blue-300 mb-1"
      >
        [show less]
      </button>
      <div className="whitespace-pre-wrap">{renderMarkdown(content)}</div>
    </div>
  );
}

function EntryRow({ display }: { display: DisplayEntry }) {
  const [expanded, setExpanded] = useState(false);
  const { entry, pairedResult, label } = display;
  const ts = entry.timestamp?.slice(11, 19) || "";

  const spanBadge = entry.spanId ? <SpanBadge spanId={entry.spanId} /> : null;

  switch (label) {
    case "user":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.user.text}`}>user{spanBadge}</span>
          <ExpandableContent content={entry.content} />
        </div>
      );

    case "manage":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.manage.text}`}>manage{spanBadge}</span>
          <ExpandableContent content={entry.content} />
        </div>
      );

    case "system":
      return (
        <div className="flex gap-3 py-1">
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className={`w-[72px] shrink-0 text-xs ${ROLE_STYLES.system.text}`}>system{spanBadge}</span>
          <span className="text-xs italic text-muted-foreground">{entry.content}</span>
        </div>
      );

    case "delegate": {
      // Parse the task from toolInput JSON
      let taskDesc = "";
      let contextDesc = "";
      if (entry.toolInput) {
        try {
          const parsed = JSON.parse(entry.toolInput);
          taskDesc = parsed.task || "";
          contextDesc = parsed.context || "";
        } catch {
          taskDesc = entry.toolInput;
        }
      }
      return (
        <div className={`flex gap-3 py-1 pl-4 border-l-2 ${ROLE_STYLES.delegate.border}`}>
          <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
          <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.delegate.text}`}>delegate{spanBadge}</span>
          <div className="min-w-0">
            <div className={`text-sm ${ROLE_STYLES.delegate.text}`}>
              {taskDesc || "delegate_task"}
            </div>
            {contextDesc && (
              <div className="text-xs text-muted-foreground/60 mt-0.5">{contextDesc}</div>
            )}
            {pairedResult && (
              <div className="text-xs text-muted-foreground/50 mt-0.5">
                {truncate(pairedResult.content, 120)}
              </div>
            )}
          </div>
        </div>
      );
    }

    case "implement": {
      // Assistant text (no tool)
      if (entry.type === "assistant") {
        return (
          <div className="flex gap-3 py-1">
            <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
            <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.implement.text}`}>implement{spanBadge}</span>
            <ExpandableContent content={entry.content} className="text-foreground" />
          </div>
        );
      }

      // Tool call (with optional paired result)
      if (entry.type === "tool_call") {
        const resultFailed = pairedResult
          ? /failed|error/i.test(pairedResult.content)
          : false;

        return (
          <div className="flex gap-3 py-1">
            <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
            <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.implement.text}`}>implement{spanBadge}</span>
            <div className="min-w-0">
              <button
                onClick={() => setExpanded(!expanded)}
                className={`text-sm ${ROLE_STYLES.implement.text} hover:opacity-80`}
              >
                {expanded ? "v" : ">"} {entry.toolName}
              </button>
              {/* Inline result summary (always visible, muted) */}
              {pairedResult && !expanded && (
                <span className={`ml-2 text-xs ${resultFailed ? ROLE_STYLES.error.text : "text-muted-foreground/60"}`}>
                  {truncate(pairedResult.content, 80)}
                </span>
              )}
              {expanded && (
                <div className="mt-1 space-y-1">
                  {entry.toolInput && (
                    <pre className="p-2 bg-muted text-xs overflow-x-auto rounded">{formatJSON(entry.toolInput)}</pre>
                  )}
                  {pairedResult && (
                    <div className={`border-l-2 ${ROLE_STYLES.implement.border} pl-2`}>
                      <ToolResult content={pairedResult.content} failed={resultFailed} />
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        );
      }

      // Orphaned tool_result (no paired tool_call)
      if (entry.type === "tool_result") {
        const failed = /failed|error/i.test(entry.content);
        return (
          <div className="flex gap-3 py-1">
            <span className="w-16 shrink-0 text-muted-foreground/50 text-xs">{ts}</span>
            <span className={`w-[72px] shrink-0 text-xs font-medium ${ROLE_STYLES.implement.text}`}>implement{spanBadge}</span>
            <ToolResult content={entry.content} failed={failed} />
          </div>
        );
      }

      return null;
    }

    default:
      return null;
  }
}

function ToolResult({ content, failed }: { content: string; failed: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = content.length > 200;
  const colorClass = failed ? "text-red-400" : "text-muted-foreground";

  if (!isLong) {
    return <pre className={`text-xs ${colorClass} whitespace-pre-wrap break-all`}>{content}</pre>;
  }

  const { isJSON, formatted } = tryFormatAsJSON(content);

  return (
    <div className="min-w-0">
      <button onClick={() => setExpanded(!expanded)} className={`text-xs ${colorClass} hover:text-foreground`}>
        {expanded ? "v collapse" : "> expand"} ({content.length} chars){isJSON ? " [JSON]" : ""}
      </button>
      {expanded ? (
        isJSON ? (
          <pre className={`mt-1 p-2 bg-muted rounded text-xs font-mono ${colorClass} whitespace-pre-wrap break-all overflow-x-auto`}>
            {formatted}
          </pre>
        ) : (
          <div className={`mt-1 text-xs ${colorClass} whitespace-pre-wrap break-all`}>
            {renderMarkdown(content)}
          </div>
        )
      ) : (
        <pre className={`mt-1 text-xs ${colorClass} whitespace-pre-wrap break-all line-clamp-3`}>
          {content}
        </pre>
      )}
    </div>
  );
}

function formatJSON(s: string): string {
  try { return JSON.stringify(JSON.parse(s), null, 2); } catch { return s; }
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s;
  return s.slice(0, max) + "...";
}
