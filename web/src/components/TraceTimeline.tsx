import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import type { TraceSpan, FileDiff } from "../types/agent-run";
import { Badge } from "./ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { cn } from "../lib/utils";
import {
  ChevronRightIcon,
  ChevronDownIcon,
  FileIcon,
} from "lucide-react";

// ============================================================
// Trace Timeline — Flame Graph / Waterfall View
// Horizontal span bars nested by parent-child, width proportional
// to duration. Click to expand details + diff viewer.
// ============================================================

// -- Span type visual config --------------------------------

const SPAN_TYPE_STYLES: Record<
  string,
  {
    label: string;
    bar: string;
    barActive: string;
    text: string;
    dot: string;
  }
> = {
  llm: {
    label: "LLM",
    bar: "bg-blue-500/20 border-l-2 border-blue-500",
    barActive: "bg-blue-500/30 border-l-2 border-blue-500",
    text: "text-blue-400",
    dot: "bg-blue-500",
  },
  tool: {
    label: "TOOL",
    bar: "bg-green-500/20 border-l-2 border-green-500",
    barActive: "bg-green-500/30 border-l-2 border-green-500",
    text: "text-green-400",
    dot: "bg-green-500",
  },
  thought: {
    label: "THINK",
    bar: "bg-muted border-l-2 border-muted-foreground",
    barActive: "bg-muted border-l-2 border-muted-foreground",
    text: "text-muted-foreground",
    dot: "bg-muted-foreground",
  },
  input: {
    label: "INPUT",
    bar: "bg-yellow-500/20 border-l-2 border-yellow-500",
    barActive: "bg-yellow-500/30 border-l-2 border-yellow-500",
    text: "text-yellow-400",
    dot: "bg-yellow-500",
  },
  delegate: {
    label: "DELEG",
    bar: "bg-purple-500/20 border-l-2 border-purple-500",
    barActive: "bg-purple-500/30 border-l-2 border-purple-500",
    text: "text-purple-400",
    dot: "bg-purple-500",
  },
  lifecycle: {
    label: "LIFECYCLE",
    bar: "bg-yellow-500/20 border-l-2 border-yellow-500",
    barActive: "bg-yellow-500/30 border-l-2 border-yellow-500",
    text: "text-yellow-400",
    dot: "bg-yellow-500",
  },
};

const FAILED_STYLES = {
  bar: "bg-red-500/20 border-l-2 border-red-500",
  text: "text-red-400",
  dot: "bg-red-500",
};

// -- Helpers ------------------------------------------------

function getDurationMs(startTime: string, endTime: string): number {
  const start = new Date(startTime).getTime();
  const end = endTime ? new Date(endTime).getTime() : Date.now();
  return Math.max(0, end - start);
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const secs = Math.floor(ms / 1000);
  if (secs < 60) return `${secs}.${Math.floor((ms % 1000) / 100)}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

function formatTime(iso: string): string {
  if (!iso) return "--";
  return new Date(iso).toLocaleTimeString();
}

// -- Tree building ------------------------------------------

interface FlatSpan {
  span: TraceSpan;
  depth: number;
  durationMs: number;
  offsetMs: number;
}

function buildFlatTree(spans: TraceSpan[]): {
  flat: FlatSpan[];
  traceStartMs: number;
  traceDurationMs: number;
} {
  if (spans.length === 0) {
    return { flat: [], traceStartMs: 0, traceDurationMs: 0 };
  }

  // Build lookup maps
  const byId = new Map<string, TraceSpan>();
  const childrenOf = new Map<string, TraceSpan[]>();

  for (const span of spans) {
    byId.set(span.id, span);
  }

  // Separate roots and children
  const roots: TraceSpan[] = [];
  for (const span of spans) {
    if (!span.parentId || !byId.has(span.parentId)) {
      roots.push(span);
    } else {
      const siblings = childrenOf.get(span.parentId) ?? [];
      siblings.push(span);
      childrenOf.set(span.parentId, siblings);
    }
  }

  // Sort roots by start time
  roots.sort(
    (a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
  );

  // Find global trace start for offset calculation
  let traceStartMs = Infinity;
  let traceEndMs = -Infinity;
  for (const span of spans) {
    const s = new Date(span.startTime).getTime();
    const e = span.endTime ? new Date(span.endTime).getTime() : Date.now();
    if (s < traceStartMs) traceStartMs = s;
    if (e > traceEndMs) traceEndMs = e;
  }
  const traceDurationMs = Math.max(1, traceEndMs - traceStartMs);

  // DFS flatten
  const flat: FlatSpan[] = [];
  function walk(node: TraceSpan, depth: number) {
    const startMs = new Date(node.startTime).getTime();
    const durationMs = getDurationMs(node.startTime, node.endTime);
    flat.push({
      span: node,
      depth,
      durationMs,
      offsetMs: startMs - traceStartMs,
    });
    const children = childrenOf.get(node.id) ?? [];
    children.sort(
      (a, b) =>
        new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
    );
    for (const child of children) {
      walk(child, depth + 1);
    }
  }
  for (const root of roots) {
    walk(root, 0);
  }

  return { flat, traceStartMs, traceDurationMs };
}

// -- Diff viewer --------------------------------------------

function DiffViewer({ files }: { files: FileDiff[] }) {
  return (
    <div className="space-y-2">
      {files.map((file) => (
        <Collapsible key={file.path} defaultOpen>
          <CollapsibleTrigger className="flex items-center gap-2 w-full px-3 py-1.5 text-left text-xs font-mono hover:bg-muted/50 transition-colors rounded border border-border bg-background group">
            <ChevronRightIcon className="h-3 w-3 text-muted-foreground flex-shrink-0 group-data-[state=open]:rotate-90 transition-transform" />
            <FileIcon className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="text-foreground truncate">{file.path}</span>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="border border-border rounded mt-1 overflow-x-auto">
              <pre className="text-[11px] leading-[18px] font-mono">
                {file.patch.split("\n").map((line, idx) => {
                  let cls = "px-3 ";
                  if (line.startsWith("+") && !line.startsWith("+++"))
                    cls += "bg-green-500/10 text-green-400";
                  else if (line.startsWith("-") && !line.startsWith("---"))
                    cls += "bg-red-500/10 text-red-400";
                  else if (line.startsWith("@@"))
                    cls += "bg-blue-500/10 text-blue-400";
                  else cls += "text-muted-foreground";
                  return <div key={idx} className={cls}>{line}</div>;
                })}
              </pre>
            </div>
          </CollapsibleContent>
        </Collapsible>
      ))}
    </div>
  );
}

// -- Span detail panel --------------------------------------

function SpanDetail({ span, runId }: { span: TraceSpan; runId?: string }) {
  const [diffData, setDiffData] = useState<FileDiff[] | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);

  // Fetch diff on mount if span has diff
  useEffect(() => {
    if (!span.hasDiff || !runId) return;
    setDiffLoading(true);
    import("../hooks/apiFetch").then(({ apiFetch }) => {
      apiFetch(`/api/v1/runs/${runId}/traces/${span.id}/diff`)
        .then((r) => r.ok ? r.json() : null)
        .then((data) => {
          if (data?.files) setDiffData(data.files);
        })
        .catch(() => {})
        .finally(() => setDiffLoading(false));
    });
  }, [span.id, span.hasDiff, runId]);
  const durationMs = getDurationMs(span.startTime, span.endTime);
  const isFailed = span.metadata?.error !== undefined;

  return (
    <div className="bg-muted/30 border-t border-border px-4 py-3 space-y-3 text-xs">
      {/* Metadata grid */}
      <div className="grid grid-cols-2 gap-x-6 gap-y-1 max-w-lg">
        <div className="text-muted-foreground">Type</div>
        <div className="font-medium text-foreground">
          {(SPAN_TYPE_STYLES[span.type]?.label ?? span.type).toUpperCase()}
        </div>

        <div className="text-muted-foreground">Start</div>
        <div className="font-mono text-foreground">
          {formatTime(span.startTime)}
        </div>

        <div className="text-muted-foreground">End</div>
        <div className="font-mono text-foreground">
          {span.endTime ? formatTime(span.endTime) : "running..."}
        </div>

        <div className="text-muted-foreground">Duration</div>
        <div className="font-mono text-foreground">
          {formatDuration(durationMs)}
        </div>

        {span.parentId && (
          <>
            <div className="text-muted-foreground">Parent</div>
            <div className="font-mono text-foreground truncate">
              {span.parentId}
            </div>
          </>
        )}

        {isFailed && (
          <>
            <div className="text-red-400">Error</div>
            <div className="text-red-400 font-mono truncate">
              {String(span.metadata?.error)}
            </div>
          </>
        )}
      </div>

      {/* Extra metadata */}
      {span.metadata &&
        Object.keys(span.metadata).filter((k) => k !== "error").length > 0 && (
          <Collapsible>
            <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors group">
              <ChevronRightIcon className="h-3 w-3 group-data-[state=open]:rotate-90 transition-transform" />
              Metadata
            </CollapsibleTrigger>
            <CollapsibleContent>
              <pre className="mt-1 p-2 bg-background border border-border rounded text-[11px] font-mono text-foreground overflow-x-auto max-h-48 overflow-y-auto">
                {JSON.stringify(
                  Object.fromEntries(
                    Object.entries(span.metadata).filter(
                      ([k]) => k !== "error"
                    )
                  ),
                  null,
                  2
                )}
              </pre>
            </CollapsibleContent>
          </Collapsible>
        )}

      {/* Diff viewer */}
      {span.hasDiff && (
        <div className="space-y-1.5">
          {diffLoading && (
            <div className="text-xs text-muted-foreground">Loading diff...</div>
          )}
          {diffData && diffData.length > 0 && (
            <>
              <div className="text-xs font-medium text-foreground">
                Changes ({diffData.length}{" "}
                {diffData.length === 1 ? "file" : "files"})
              </div>
              <DiffViewer files={diffData} />
            </>
          )}
          {diffData && diffData.length === 0 && (
            <div className="text-xs text-muted-foreground">No file changes in this span</div>
          )}
        </div>
      )}
    </div>
  );
}

// -- Constants for virtualization ---------------------------

const SPAN_ROW_HEIGHT = 32;
const DETAIL_HEIGHT_ESTIMATE = 200;
const OVERSCAN = 8;
const INDENT_PX = 20;
const LABEL_WIDTH = 240;
const MIN_BAR_WIDTH_PX = 4;

// -- Main component -----------------------------------------

export default function TraceTimeline({
  spans,
  selectedSpanId,
  onSelectSpan,
  runId,
  agentType,
}: {
  spans: TraceSpan[];
  selectedSpanId?: string;
  onSelectSpan: (span: TraceSpan) => void;
  runId?: string;
  agentType?: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [containerHeight, setContainerHeight] = useState(400);
  const [expandedSpanIds, setExpandedSpanIds] = useState<Set<string>>(
    new Set()
  );
  const [userScrolled, setUserScrolled] = useState(false);
  const prevSpanCount = useRef(spans.length);

  // Build tree structure
  const { flat, traceDurationMs } = useMemo(
    () => buildFlatTree(spans),
    [spans]
  );

  // Toggle detail expansion
  const toggleExpand = useCallback(
    (span: TraceSpan) => {
      setExpandedSpanIds((prev) => {
        const next = new Set(prev);
        if (next.has(span.id)) {
          next.delete(span.id);
        } else {
          next.add(span.id);
        }
        return next;
      });
      onSelectSpan(span);
    },
    [onSelectSpan]
  );

  // Track container resize
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerHeight(entry.contentRect.height);
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Auto-scroll to bottom on new spans
  useEffect(() => {
    if (spans.length > prevSpanCount.current && !userScrolled) {
      const el = containerRef.current;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
    }
    prevSpanCount.current = spans.length;
  }, [spans.length, userScrolled]);

  const handleScroll = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    setScrollTop(el.scrollTop);
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setUserScrolled(!atBottom);
  }, []);

  // Compute positions for each row (accounting for expanded detail panels)
  const rowPositions = useMemo(() => {
    const positions: { top: number; height: number; flatIndex: number }[] = [];
    let y = 0;
    for (let i = 0; i < flat.length; i++) {
      const isExpanded = expandedSpanIds.has(flat[i].span.id);
      const rowH = SPAN_ROW_HEIGHT + (isExpanded ? DETAIL_HEIGHT_ESTIMATE : 0);
      positions.push({ top: y, height: rowH, flatIndex: i });
      y += rowH;
    }
    return positions;
  }, [flat, expandedSpanIds]);

  const totalHeight = useMemo(() => {
    if (rowPositions.length === 0) return 0;
    const last = rowPositions[rowPositions.length - 1];
    return last.top + last.height;
  }, [rowPositions]);

  // Determine visible rows
  const visibleRange = useMemo(() => {
    const viewTop = scrollTop;
    const viewBottom = scrollTop + containerHeight;

    let startIdx = 0;
    let endIdx = rowPositions.length;

    // Binary search for start
    let lo = 0;
    let hi = rowPositions.length - 1;
    while (lo <= hi) {
      const mid = Math.floor((lo + hi) / 2);
      if (rowPositions[mid].top + rowPositions[mid].height < viewTop) {
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }
    startIdx = Math.max(0, lo - OVERSCAN);

    // Binary search for end
    lo = startIdx;
    hi = rowPositions.length - 1;
    while (lo <= hi) {
      const mid = Math.floor((lo + hi) / 2);
      if (rowPositions[mid].top > viewBottom) {
        hi = mid - 1;
      } else {
        lo = mid + 1;
      }
    }
    endIdx = Math.min(rowPositions.length, lo + OVERSCAN);

    return { startIdx, endIdx };
  }, [scrollTop, containerHeight, rowPositions]);

  // Time axis ticks
  const timeTicks = useMemo(() => {
    const count = 5;
    const ticks: { label: string; pct: number }[] = [];
    for (let i = 0; i <= count; i++) {
      const ms = (traceDurationMs / count) * i;
      ticks.push({
        label: formatDuration(Math.round(ms)),
        pct: (i / count) * 100,
      });
    }
    return ticks;
  }, [traceDurationMs]);

  if (spans.length === 0) {
    return (
      <div
        data-testid="trace-timeline"
        className="flex items-center justify-center py-12 text-muted-foreground text-sm"
      >
        No trace spans recorded
      </div>
    );
  }

  return (
    <div data-testid="trace-timeline" className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border bg-muted/30">
        <div className="flex items-center gap-2">
          {runId && (
            <span className="text-xs font-mono text-foreground truncate max-w-[200px]">
              {runId}
            </span>
          )}
          {agentType && (
            <Badge variant="outline" className="text-[10px]">
              {agentType}
            </Badge>
          )}
          <span className="text-xs text-muted-foreground">
            {spans.length} span{spans.length !== 1 ? "s" : ""}
          </span>
        </div>
        {/* Legend */}
        <div className="flex items-center gap-3">
          {Object.entries(SPAN_TYPE_STYLES).map(([type, style]) => (
            <div key={type} className="flex items-center gap-1.5">
              <span
                className={cn("inline-block h-2 w-2 rounded-sm", style.dot)}
              />
              <span className="text-[10px] text-muted-foreground uppercase tracking-wider">
                {style.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Column headers: label column + time axis */}
      <div className="flex border-b border-border bg-muted/20">
        <div
          className="flex-shrink-0 px-3 py-1 text-[10px] text-muted-foreground uppercase tracking-wider border-r border-border"
          style={{ width: LABEL_WIDTH }}
        >
          Span
        </div>
        <div className="flex-1 relative py-1 px-2">
          {timeTicks.map((tick, i) => (
            <span
              key={i}
              className="absolute text-[10px] text-muted-foreground font-mono -translate-x-1/2"
              style={{ left: `${tick.pct}%` }}
            >
              {tick.label}
            </span>
          ))}
        </div>
      </div>

      {/* Virtualized waterfall body */}
      <div
        ref={containerRef}
        className="flex-1 overflow-y-auto min-h-0"
        style={{ maxHeight: "100%" }}
        onScroll={handleScroll}
      >
        <div style={{ height: totalHeight, position: "relative" }}>
          {rowPositions.slice(visibleRange.startIdx, visibleRange.endIdx).map((pos) => {
            const { span, depth, durationMs, offsetMs } = flat[pos.flatIndex];
            const style = SPAN_TYPE_STYLES[span.type] ?? SPAN_TYPE_STYLES.tool;
            const isActive = !span.endTime;
            const isExpanded = expandedSpanIds.has(span.id);
            const isSelected = span.id === selectedSpanId;
            const isFailed =
              span.metadata?.error !== undefined ||
              (span.type === "tool" && span.metadata?.exitCode !== 0 && span.metadata?.exitCode !== undefined);

            // Bar positioning within the waterfall
            const barLeftPct =
              traceDurationMs > 0 ? (offsetMs / traceDurationMs) * 100 : 0;
            const barWidthPct =
              traceDurationMs > 0
                ? Math.max(
                    (MIN_BAR_WIDTH_PX / (containerRef.current?.clientWidth ?? 600)) * 100,
                    (durationMs / traceDurationMs) * 100
                  )
                : 100;

            const barClass = isFailed
              ? FAILED_STYLES.bar
              : isActive
                ? style.barActive
                : style.bar;

            const textClass = isFailed ? FAILED_STYLES.text : style.text;

            return (
              <div
                key={span.id}
                style={{
                  position: "absolute",
                  top: pos.top,
                  left: 0,
                  right: 0,
                }}
              >
                {/* Span row */}
                <button
                  onClick={() => toggleExpand(span)}
                  className={cn(
                    "flex items-center w-full text-left transition-colors group",
                    isSelected
                      ? "bg-accent/10"
                      : "hover:bg-muted/40"
                  )}
                  style={{ height: SPAN_ROW_HEIGHT }}
                >
                  {/* Label column */}
                  <div
                    className="flex items-center gap-1.5 flex-shrink-0 px-2 h-full border-r border-border overflow-hidden"
                    style={{
                      width: LABEL_WIDTH,
                      paddingLeft: depth * INDENT_PX + 8,
                    }}
                  >
                    {/* Expand chevron */}
                    <span className="flex-shrink-0 w-3 h-3">
                      {isExpanded ? (
                        <ChevronDownIcon className="h-3 w-3 text-muted-foreground" />
                      ) : (
                        <ChevronRightIcon className="h-3 w-3 text-muted-foreground" />
                      )}
                    </span>

                    {/* Type dot */}
                    <span
                      className={cn(
                        "inline-block h-2 w-2 rounded-sm flex-shrink-0",
                        isFailed ? FAILED_STYLES.dot : style.dot
                      )}
                    />

                    {/* Span name */}
                    <span
                      className={cn(
                        "text-[11px] font-mono truncate",
                        textClass
                      )}
                    >
                      {span.name}
                    </span>

                    {/* Diff indicator */}
                    {span.hasDiff && (
                      <Badge
                        variant="outline"
                        className="text-[9px] px-1 py-0 ml-auto flex-shrink-0 border-blue-500/40 text-blue-400"
                      >
                        DIFF
                      </Badge>
                    )}
                  </div>

                  {/* Waterfall bar column */}
                  <div className="flex-1 relative h-full flex items-center px-1">
                    {/* Background grid lines */}
                    {timeTicks.map((tick, i) => (
                      <div
                        key={i}
                        className="absolute top-0 bottom-0 border-l border-border/30"
                        style={{ left: `${tick.pct}%` }}
                      />
                    ))}

                    {/* Duration bar */}
                    <div
                      className={cn(
                        "absolute h-5 rounded-sm transition-all",
                        barClass,
                        isActive && "animate-pulse"
                      )}
                      style={{
                        left: `${barLeftPct}%`,
                        width: `${barWidthPct}%`,
                        minWidth: MIN_BAR_WIDTH_PX,
                      }}
                    >
                      {/* Duration label inside bar if wide enough */}
                      {barWidthPct > 8 && (
                        <span
                          className={cn(
                            "absolute inset-0 flex items-center px-1.5 text-[10px] font-mono truncate",
                            textClass
                          )}
                        >
                          {formatDuration(durationMs)}
                        </span>
                      )}
                    </div>

                    {/* Duration label outside bar if narrow */}
                    {barWidthPct <= 8 && (
                      <span
                        className="absolute text-[10px] font-mono text-muted-foreground whitespace-nowrap"
                        style={{
                          left: `calc(${barLeftPct + barWidthPct}% + 4px)`,
                        }}
                      >
                        {formatDuration(durationMs)}
                      </span>
                    )}
                  </div>
                </button>

                {/* Expanded detail section */}
                {isExpanded && <SpanDetail span={span} runId={runId} />}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
