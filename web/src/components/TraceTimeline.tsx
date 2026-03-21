import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import type { TraceSpan, FileDiff } from "../types/agent-run";
import { Badge } from "./ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { cn } from "../lib/utils";
import { ROLE_STYLES, roleFromSpanName } from "../lib/role-styles";
import type { RoleName } from "../lib/role-styles";
import {
  ChevronRightIcon,
  FileIcon,
  XIcon,
} from "lucide-react";

// ============================================================
// Trace Timeline — Waterfall View with Right Split Detail Panel
// Horizontal span bars nested by parent-child, width proportional
// to duration. Click a span to view details in a right panel.
// ============================================================

// -- Failed span override styles ----------------------------

const FAILED_STYLES = {
  bar: "bg-red-500/20 border-l-2 border-red-500",
  text: "text-red-400",
  dot: "bg-red-500",
};

// -- Role-based bar styles ----------------------------------

function roleBarStyles(role: RoleName) {
  const s = ROLE_STYLES[role];
  return {
    bar: cn(s.bg, "border-l-2", s.border),
    barActive: cn(s.bg, "border-l-2", s.border),
    text: s.text,
    dot: s.dot,
  };
}

// -- Legend entries for roles --------------------------------

const LEGEND_ROLES: { role: RoleName; label: string }[] = [
  { role: "manage", label: "MANAGE" },
  { role: "implement", label: "IMPLEMENT" },
  { role: "system", label: "SYSTEM" },
  { role: "delegate", label: "DELEGATE" },
];

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

  const byId = new Map<string, TraceSpan>();
  const childrenOf = new Map<string, TraceSpan[]>();

  for (const span of spans) {
    byId.set(span.id, span);
  }

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

  roots.sort(
    (a, b) =>
      new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
  );

  let traceStartMs = Infinity;
  let traceEndMs = -Infinity;
  for (const span of spans) {
    const s = new Date(span.startTime).getTime();
    const e = span.endTime ? new Date(span.endTime).getTime() : Date.now();
    if (s < traceStartMs) traceStartMs = s;
    if (e > traceEndMs) traceEndMs = e;
  }
  const traceDurationMs = Math.max(1, traceEndMs - traceStartMs);

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
                  return (
                    <div key={idx} className={cls}>
                      {line}
                    </div>
                  );
                })}
              </pre>
            </div>
          </CollapsibleContent>
        </Collapsible>
      ))}
    </div>
  );
}

// -- Span detail panel (right side) -------------------------

function SpanDetail({
  span,
  runId,
  onClose,
}: {
  span: TraceSpan;
  runId?: string;
  onClose: () => void;
}) {
  const [diffData, setDiffData] = useState<FileDiff[] | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);

  useEffect(() => {
    if (!span.hasDiff || !runId) return;
    setDiffLoading(true);
    import("../hooks/apiFetch").then(({ apiFetch }) => {
      apiFetch(`/api/v1/runs/${runId}/traces/${span.id}/diff`)
        .then((r) => (r.ok ? r.json() : null))
        .then((data) => {
          if (data?.files) setDiffData(data.files);
        })
        .catch(() => {})
        .finally(() => setDiffLoading(false));
    });
  }, [span.id, span.hasDiff, runId]);

  const durationMs = getDurationMs(span.startTime, span.endTime);
  const isFailed = span.metadata?.error !== undefined;
  const role = roleFromSpanName(span.name);
  const roleStyle = ROLE_STYLES[role];

  const meta = span.metadata ?? {};
  const toolInput = meta.toolInput as string | undefined;
  const thinkingText =
    span.type === "thought" ? (meta.thinking as string | undefined) : undefined;
  const checkpointSha = meta.checkpointSha as string | undefined;
  const stage = meta.stage as string | undefined;

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/30 flex-shrink-0">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "inline-block h-2 w-2 rounded-sm flex-shrink-0",
                isFailed ? FAILED_STYLES.dot : roleStyle.dot
              )}
            />
            <h3
              className={cn(
                "text-sm font-mono font-medium truncate",
                isFailed ? FAILED_STYLES.text : roleStyle.text
              )}
            >
              {span.name}
            </h3>
          </div>
          <div className="text-xs text-muted-foreground font-mono mt-0.5">
            {formatDuration(durationMs)}
          </div>
        </div>
        <button
          onClick={onClose}
          className="flex-shrink-0 p-1 rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
        >
          <XIcon className="h-4 w-4" />
        </button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto min-h-0 px-4 py-3 space-y-4 text-xs">
        {/* Metadata grid */}
        <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
          {role && (
            <>
              <div className="text-muted-foreground">Role</div>
              <div className={cn("font-medium capitalize", roleStyle.text)}>
                {role}
              </div>
            </>
          )}

          {stage && (
            <>
              <div className="text-muted-foreground">Stage</div>
              <div className="font-medium text-foreground">{stage}</div>
            </>
          )}

          <div className="text-muted-foreground">Type</div>
          <div className="font-medium text-foreground uppercase">
            {span.type}
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

          {checkpointSha && (
            <>
              <div className="text-muted-foreground">Checkpoint</div>
              <div className="font-mono text-foreground truncate">
                {checkpointSha}
              </div>
            </>
          )}

          {isFailed && (
            <>
              <div className="text-red-400">Error</div>
              <div className="text-red-400 font-mono break-all">
                {String(meta.error)}
              </div>
            </>
          )}
        </div>

        {/* Tool input */}
        {toolInput && (
          <div className="space-y-1.5">
            <div className="text-xs font-medium text-foreground">
              Tool Input
            </div>
            <pre className="p-2 bg-background border border-border rounded text-[11px] font-mono text-foreground overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all">
              {toolInput}
            </pre>
          </div>
        )}

        {/* Thinking content */}
        {thinkingText && (
          <div className="space-y-1.5">
            <div className="text-xs font-medium text-foreground">Thinking</div>
            <div className="p-2 bg-background border border-border rounded text-[11px] text-foreground overflow-y-auto max-h-64 whitespace-pre-wrap">
              {thinkingText}
            </div>
          </div>
        )}

        {/* Extra metadata (everything else) */}
        {meta &&
          Object.keys(meta).filter(
            (k) =>
              !["error", "toolInput", "thinking", "checkpointSha", "stage"].includes(k)
          ).length > 0 && (
            <Collapsible>
              <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors group">
                <ChevronRightIcon className="h-3 w-3 group-data-[state=open]:rotate-90 transition-transform" />
                All Metadata
              </CollapsibleTrigger>
              <CollapsibleContent>
                <pre className="mt-1 p-2 bg-background border border-border rounded text-[11px] font-mono text-foreground overflow-x-auto max-h-48 overflow-y-auto">
                  {JSON.stringify(
                    Object.fromEntries(
                      Object.entries(meta).filter(
                        ([k]) =>
                          !["error", "toolInput", "thinking", "checkpointSha", "stage"].includes(k)
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
              <div className="text-xs text-muted-foreground">
                Loading diff...
              </div>
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
              <div className="text-xs text-muted-foreground">
                No file changes in this span
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// -- Constants for virtualization ---------------------------

const SPAN_ROW_HEIGHT = 32;
const OVERSCAN = 8;
const INDENT_PX = 20;
const LABEL_WIDTH = 240;
const MIN_BAR_WIDTH_PX = 4;
const DEFAULT_DETAIL_WIDTH = 400;
const MIN_DETAIL_WIDTH = 280;
const MAX_DETAIL_WIDTH = 700;

// -- Main component -----------------------------------------

export default function TraceTimeline({
  spans,
  selectedSpanId: controlledSelectedSpanId,
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
  const [internalSelectedSpanId, setInternalSelectedSpanId] = useState<
    string | null
  >(null);
  const [userScrolled, setUserScrolled] = useState(false);
  const [detailWidth, setDetailWidth] = useState(DEFAULT_DETAIL_WIDTH);
  const [isDragging, setIsDragging] = useState(false);
  const prevSpanCount = useRef(spans.length);

  // Use controlled selectedSpanId if provided, otherwise use internal
  const selectedSpanId = controlledSelectedSpanId ?? internalSelectedSpanId;

  // Build tree structure
  const { flat, traceDurationMs } = useMemo(
    () => buildFlatTree(spans),
    [spans]
  );

  // Find the selected span object
  const selectedSpan = useMemo(() => {
    if (!selectedSpanId) return null;
    const entry = flat.find((f) => f.span.id === selectedSpanId);
    return entry?.span ?? null;
  }, [flat, selectedSpanId]);

  // Select a span (single selection, no toggle)
  const handleSelectSpan = useCallback(
    (span: TraceSpan) => {
      setInternalSelectedSpanId(span.id);
      onSelectSpan(span);
    },
    [onSelectSpan]
  );

  // Close the detail panel
  const handleCloseDetail = useCallback(() => {
    setInternalSelectedSpanId(null);
  }, []);

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

  // Draggable divider for resizing the detail panel
  const handleDividerMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsDragging(true);
      const startX = e.clientX;
      const startWidth = detailWidth;

      const onMouseMove = (ev: MouseEvent) => {
        const delta = startX - ev.clientX;
        const newWidth = Math.min(
          MAX_DETAIL_WIDTH,
          Math.max(MIN_DETAIL_WIDTH, startWidth + delta)
        );
        setDetailWidth(newWidth);
      };

      const onMouseUp = () => {
        setIsDragging(false);
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", onMouseUp);
      };

      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
    },
    [detailWidth]
  );

  // Compute positions for each row
  const rowPositions = useMemo(() => {
    const positions: { top: number; height: number; flatIndex: number }[] = [];
    let y = 0;
    for (let i = 0; i < flat.length; i++) {
      positions.push({ top: y, height: SPAN_ROW_HEIGHT, flatIndex: i });
      y += SPAN_ROW_HEIGHT;
    }
    return positions;
  }, [flat]);

  const totalHeight = useMemo(() => {
    return flat.length * SPAN_ROW_HEIGHT;
  }, [flat.length]);

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
      <div className="flex items-center justify-between px-3 py-2 border-b border-border bg-muted/30 flex-shrink-0">
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
        {/* Legend — role-based */}
        <div className="flex items-center gap-3">
          {LEGEND_ROLES.map(({ role, label }) => (
            <div key={role} className="flex items-center gap-1.5">
              <span
                className={cn(
                  "inline-block h-2 w-2 rounded-sm",
                  ROLE_STYLES[role].dot
                )}
              />
              <span className="text-[10px] text-muted-foreground uppercase tracking-wider">
                {label}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Two-column body: waterfall + detail panel */}
      <div className="flex flex-1 min-h-0 overflow-hidden">
        {/* Left: waterfall */}
        <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
          {/* Column headers: label column + time axis */}
          <div className="flex border-b border-border bg-muted/20 flex-shrink-0">
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
            onScroll={handleScroll}
          >
            <div style={{ height: totalHeight, position: "relative" }}>
              {rowPositions
                .slice(visibleRange.startIdx, visibleRange.endIdx)
                .map((pos) => {
                  const { span, depth, durationMs, offsetMs } =
                    flat[pos.flatIndex];
                  const role = roleFromSpanName(span.name);
                  const rStyle = roleBarStyles(role);
                  const isActive = !span.endTime;
                  const isSelected = span.id === selectedSpanId;
                  const isFailed =
                    span.metadata?.error !== undefined ||
                    (span.type === "tool" &&
                      span.metadata?.exitCode !== 0 &&
                      span.metadata?.exitCode !== undefined);

                  // Stage separator: check previous span's stage
                  const currentStage = span.metadata?.stage as
                    | string
                    | undefined;
                  let showStageSeparator = false;
                  if (currentStage && pos.flatIndex > 0) {
                    const prevSpan = flat[pos.flatIndex - 1].span;
                    const prevStage = prevSpan.metadata?.stage as
                      | string
                      | undefined;
                    if (prevStage !== currentStage) {
                      showStageSeparator = true;
                    }
                  } else if (
                    currentStage &&
                    pos.flatIndex === 0
                  ) {
                    showStageSeparator = true;
                  }

                  // Bar positioning within the waterfall
                  const barLeftPct =
                    traceDurationMs > 0
                      ? (offsetMs / traceDurationMs) * 100
                      : 0;
                  const barWidthPct =
                    traceDurationMs > 0
                      ? Math.max(
                          (MIN_BAR_WIDTH_PX /
                            (containerRef.current?.clientWidth ?? 600)) *
                            100,
                          (durationMs / traceDurationMs) * 100
                        )
                      : 100;

                  const barClass = isFailed
                    ? FAILED_STYLES.bar
                    : isActive
                      ? rStyle.barActive
                      : rStyle.bar;

                  const textClass = isFailed ? FAILED_STYLES.text : rStyle.text;

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
                      {/* Stage separator */}
                      {showStageSeparator && (
                        <div className="flex items-center gap-2 px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground absolute -top-4 left-0 right-0 pointer-events-none">
                          <div className="flex-1 border-t border-border" />
                          <span>{currentStage}</span>
                          <div className="flex-1 border-t border-border" />
                        </div>
                      )}

                      {/* Span row */}
                      <button
                        onClick={() => handleSelectSpan(span)}
                        className={cn(
                          "flex items-center w-full text-left transition-colors group",
                          isSelected
                            ? "bg-accent/10 ring-1 ring-inset ring-accent/20"
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
                          {/* Role dot */}
                          <span
                            className={cn(
                              "inline-block h-2 w-2 rounded-sm flex-shrink-0",
                              isFailed ? FAILED_STYLES.dot : rStyle.dot
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
                    </div>
                  );
                })}
            </div>
          </div>
        </div>

        {/* Draggable divider + Detail panel */}
        {selectedSpan && (
          <>
            {/* Divider */}
            <div
              className={cn(
                "w-1 flex-shrink-0 cursor-col-resize transition-colors hover:bg-accent/50",
                isDragging ? "bg-accent" : "bg-border"
              )}
              onMouseDown={handleDividerMouseDown}
            />

            {/* Right: detail panel */}
            <div
              className="flex-shrink-0 overflow-hidden border-l border-border bg-background"
              style={{ width: detailWidth }}
            >
              <SpanDetail
                span={selectedSpan}
                runId={runId}
                onClose={handleCloseDetail}
              />
            </div>
          </>
        )}

        {/* Empty state when no span selected — show hint at right edge */}
        {!selectedSpan && (
          <div className="flex-shrink-0 w-0 overflow-hidden" />
        )}
      </div>
    </div>
  );
}
