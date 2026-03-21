import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import type { TraceSpan, FileDiff } from "../types/agent-run";
import { Badge } from "./ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { cn } from "../lib/utils";
import { ROLE_STYLES, roleFromSpanName, displaySpanName } from "../lib/role-styles";

import {
  ChevronRightIcon,
  ChevronDownIcon,
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

// -- Operation-based bar colors (by span name suffix) -------

function resolveOperation(span: TraceSpan): string {
  // Use span name's last segment, but fall back to metadata.tool for old "*.tool" spans
  const op = span.name.split(".").pop() || "";
  if (op === "tool" && span.metadata?.tool) {
    return String(span.metadata.tool);
  }
  return op;
}

const OP_COLORS: Record<string, { bar: string; text: string }> = {
  thought: { bar: "bg-blue-500/30 border-l-2 border-blue-500",     text: "text-blue-400" },
  bash:    { bar: "bg-emerald-500/30 border-l-2 border-emerald-500", text: "text-emerald-400" },
  write:   { bar: "bg-violet-500/30 border-l-2 border-violet-500", text: "text-violet-400" },
  read:    { bar: "bg-slate-400/20 border-l-2 border-slate-400",   text: "text-slate-400" },
  started: { bar: "bg-amber-500/30 border-l-2 border-amber-500",   text: "text-amber-400" },
};
const DEFAULT_OP = { bar: "bg-emerald-500/30 border-l-2 border-emerald-500", text: "text-emerald-400" };

function getOperationColor(span: TraceSpan): string {
  return (OP_COLORS[resolveOperation(span)] ?? DEFAULT_OP).bar;
}

function getOperationTextColor(span: TraceSpan): string {
  return (OP_COLORS[resolveOperation(span)] ?? DEFAULT_OP).text;
}

function formatCost(usd: number): string {
  if (usd <= 0) return "";
  if (usd < 0.001) return "<$0.001";
  return `$${usd.toFixed(3)}`;
}

// -- Stage span aggregation (client-side, tasks 7.4/7.5) -----

interface StageAggregates {
  inputTokens: number;
  outputTokens: number;
  toolCount: number;
  toolErrors: number;
  childCount: number;
  estimatedCostUsd: number;
}

function computeStageAggregates(
  stageSpan: TraceSpan,
  allSpans: TraceSpan[]
): StageAggregates {
  // Collect ALL descendants, not just direct children
  const descendants: TraceSpan[] = [];
  const queue = [stageSpan.id];
  while (queue.length > 0) {
    const parentId = queue.shift()!;
    for (const s of allSpans) {
      if (s.parentId === parentId) {
        descendants.push(s);
        queue.push(s.id);
      }
    }
  }
  let inputTokens = 0;
  let outputTokens = 0;
  let toolCount = 0;
  let toolErrors = 0;
  for (const child of descendants) {
    inputTokens +=
      (child.metadata?.["gen_ai.usage.input_tokens"] as number) || 0;
    outputTokens +=
      (child.metadata?.["gen_ai.usage.output_tokens"] as number) || 0;
    if (child.type === "tool") toolCount++;
    if (child.metadata?.error) toolErrors++;
  }
  // Simple cost estimate: $0.15/M input + $0.75/M output (default pricing)
  const model =
    (stageSpan.metadata?.["gen_ai.request.model"] as string) || "";
  const pricing: Record<string, { input: number; output: number }> = {
    "deepseek-v3.1": { input: 0.15, output: 0.75 },
    "deepseek-v3.2": { input: 0.26, output: 0.38 },
    "qwen3-coder": { input: 0.22, output: 1.0 },
    "qwen3:8b": { input: 0.0, output: 0.0 },
    "mistral-medium": { input: 0.4, output: 2.0 },
  };
  const p = pricing[model] ?? { input: 0.15, output: 0.75 };
  const estimatedCostUsd =
    (inputTokens * p.input + outputTokens * p.output) / 1_000_000;

  return {
    inputTokens,
    outputTokens,
    toolCount,
    toolErrors,
    childCount: descendants.length,
    estimatedCostUsd,
  };
}

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
  hasChildren?: boolean;
  parentDurationMs: number;  // duration of the immediate parent span
  offsetInParentMs: number;  // offset from parent's start time
}

function buildFlatTree(
  spans: TraceSpan[],
  collapsedSpanIds: Set<string> = new Set()
): {
  flat: FlatSpan[];
  traceStartMs: number;
  traceDurationMs: number;
  childrenOf: Map<string, TraceSpan[]>;
} {
  if (spans.length === 0) {
    return {
      flat: [],
      traceStartMs: 0,
      traceDurationMs: 0,
      childrenOf: new Map(),
    };
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
  function walk(node: TraceSpan, depth: number, parentStartMs?: number, parentDurMs?: number) {
    const startMs = new Date(node.startTime).getTime();
    const durationMs = getDurationMs(node.startTime, node.endTime);
    const hasChildren = (childrenOf.get(node.id) ?? []).length > 0;
    const pDur = parentDurMs ?? traceDurationMs;
    const pStart = parentStartMs ?? traceStartMs;
    flat.push({
      span: node,
      depth,
      durationMs,
      offsetMs: startMs - traceStartMs,
      hasChildren,
      parentDurationMs: pDur,
      offsetInParentMs: startMs - pStart,
    });
    // Skip children if this span is collapsed
    if (collapsedSpanIds.has(node.id)) return;
    const children = childrenOf.get(node.id) ?? [];
    children.sort(
      (a, b) =>
        new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
    );
    for (const child of children) {
      walk(child, depth + 1, startMs, durationMs);
    }
  }
  for (const root of roots) {
    walk(root, 0);
  }

  return { flat, traceStartMs, traceDurationMs, childrenOf };
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
  allSpans,
  runId,
  onClose,
}: {
  span: TraceSpan;
  allSpans: TraceSpan[];
  runId?: string;
  onClose: () => void;
}) {
  const [diffData, setDiffData] = useState<FileDiff[] | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);

  useEffect(() => {
    if (!span.hasDiff || !runId) return;
    setDiffData(null);
    setDiffLoading(true);
    fetch(`/api/v1/runs/${runId}/traces/${span.id}/diff`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((data) => {
        if (data?.files) setDiffData(data.files);
        else setDiffData([]);
      })
      .catch((err) => {
        console.error("Failed to fetch diff:", err);
        setDiffData([]);
      })
      .finally(() => setDiffLoading(false));
  }, [span.id, span.hasDiff, runId]);

  const durationMs = getDurationMs(span.startTime, span.endTime);
  const isFailed = span.metadata?.error !== undefined || span.status === "error";
  const isStage = span.type === "stage";
  const role = roleFromSpanName(span.name);
  const roleStyle = ROLE_STYLES[isStage ? "system" : role];

  const meta = span.metadata ?? {};
  const toolInput = meta.toolInput as string | undefined;
  const contentText = (meta.content as string | undefined) ?? (meta.thinking as string | undefined);
  const checkpointSha = meta.checkpointSha as string | undefined;
  const stage = meta.stage as string | undefined;

  // Compute stage aggregates for stage/pipeline spans (tasks 7.4, 7.5)
  const stageAgg = useMemo(() => {
    if (!isStage) return null;
    return computeStageAggregates(span, allSpans);
  }, [isStage, span, allSpans]);

  // For root pipeline spans, aggregate across all child stages recursively
  const pipelineAgg = useMemo(() => {
    if (!isStage) return null;
    const isRoot = !span.parentId;
    if (!isRoot) return null;
    // Aggregate all descendant spans (not just direct children)
    let inputTokens = 0;
    let outputTokens = 0;
    let toolCount = 0;
    let toolErrors = 0;
    let childCount = 0;
    for (const s of allSpans) {
      if (s.id === span.id) continue;
      inputTokens +=
        (s.metadata?.["gen_ai.usage.input_tokens"] as number) || 0;
      outputTokens +=
        (s.metadata?.["gen_ai.usage.output_tokens"] as number) || 0;
      if (s.type === "tool") toolCount++;
      if (s.metadata?.error) toolErrors++;
      childCount++;
    }
    const model =
      (span.metadata?.["gen_ai.request.model"] as string) || "";
    const pricing: Record<string, { input: number; output: number }> = {
      "deepseek-v3.1": { input: 0.15, output: 0.75 },
      "deepseek-v3.2": { input: 0.26, output: 0.38 },
      "qwen3-coder": { input: 0.22, output: 1.0 },
      "qwen3:8b": { input: 0.0, output: 0.0 },
      "mistral-medium": { input: 0.4, output: 2.0 },
    };
    const p = pricing[model] ?? { input: 0.15, output: 0.75 };
    const estimatedCostUsd =
      (inputTokens * p.input + outputTokens * p.output) / 1_000_000;
    return {
      inputTokens,
      outputTokens,
      toolCount,
      toolErrors,
      childCount,
      estimatedCostUsd,
    };
  }, [isStage, span, allSpans]);

  const agg = pipelineAgg ?? stageAgg;

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
                "text-sm font-mono truncate",
                isStage ? "font-bold" : "font-medium",
                isFailed ? FAILED_STYLES.text : roleStyle.text
              )}
            >
              {displaySpanName(span.name)}
              {isStage && meta.attempt != null && (
                <span className="ml-1 text-muted-foreground font-normal">
                  {"(attempt " + String(meta.attempt) + ")"}
                </span>
              )}
            </h3>
            {/* Status badge for stage spans */}
            {isStage && span.status && (
              <Badge
                variant="outline"
                className={cn(
                  "text-[9px] px-1.5 py-0",
                  span.status === "ok"
                    ? "border-emerald-500/40 text-emerald-400"
                    : span.status === "error"
                      ? "border-red-500/40 text-red-400"
                      : "border-muted-foreground/40 text-muted-foreground"
                )}
              >
                {span.status === "ok" ? "PASSED" : span.status === "error" ? "FAILED" : span.status.toUpperCase()}
              </Badge>
            )}
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
          {!isStage && role && (
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

          {isStage && meta.attempt != null && (
            <>
              <div className="text-muted-foreground">Attempt</div>
              <div className="font-mono text-foreground">
                {String(meta.attempt)}
              </div>
            </>
          )}

          <div className="text-muted-foreground">Status</div>
          <div className={cn(
            "font-medium uppercase",
            isFailed ? "text-red-400"
              : span.status === "ok" ? "text-emerald-400"
              : "text-foreground"
          )}>
            {span.status ?? (isFailed ? "error" : span.endTime ? "ok" : "running")}
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

          {isFailed && meta.error != null && (
            <>
              <div className="text-red-400">Error</div>
              <div className="text-red-400 font-mono break-all">
                {String(meta.error)}
              </div>
            </>
          )}
        </div>

        {/* Stage aggregate stats (tasks 7.4/7.5) */}
        {agg && (agg.inputTokens > 0 || agg.toolCount > 0) && (
          <div className="space-y-2">
            {/* Tokens */}
            {(agg.inputTokens > 0 || agg.outputTokens > 0) && (
              <div className="space-y-1">
                <div className="text-xs font-medium text-foreground">
                  Tokens
                </div>
                <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 pl-2">
                  <div className="text-muted-foreground">Input</div>
                  <div className="font-mono text-foreground">
                    {agg.inputTokens.toLocaleString()}
                  </div>
                  <div className="text-muted-foreground">Output</div>
                  <div className="font-mono text-foreground">
                    {agg.outputTokens.toLocaleString()}
                  </div>
                  <div className="text-muted-foreground">Total</div>
                  <div className="font-mono text-foreground">
                    {(agg.inputTokens + agg.outputTokens).toLocaleString()}
                  </div>
                </div>
              </div>
            )}

            {/* Cost */}
            {agg.estimatedCostUsd > 0 && (
              <div className="space-y-1">
                <div className="text-xs font-medium text-foreground">
                  Estimated Cost
                </div>
                <div className="font-mono text-foreground pl-2">
                  ${agg.estimatedCostUsd.toFixed(4)} USD
                </div>
              </div>
            )}

            {/* Tools */}
            {agg.toolCount > 0 && (
              <div className="space-y-1">
                <div className="text-xs font-medium text-foreground">
                  Tools
                </div>
                <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 pl-2">
                  <div className="text-muted-foreground">Total</div>
                  <div className="font-mono text-foreground">
                    {agg.toolCount}
                  </div>
                  <div className="text-muted-foreground">Errors</div>
                  <div
                    className={cn(
                      "font-mono",
                      agg.toolErrors > 0
                        ? "text-red-400"
                        : "text-foreground"
                    )}
                  >
                    {agg.toolErrors}
                  </div>
                </div>
              </div>
            )}

            {/* Child count */}
            <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
              <div className="text-muted-foreground">Child spans</div>
              <div className="font-mono text-foreground">
                {agg.childCount}
              </div>
            </div>
          </div>
        )}

        {/* Tool input — formatted JSON, collapsed by default */}
        {toolInput && (
          <Collapsible>
            <CollapsibleTrigger className="flex items-center gap-1 text-xs font-medium text-foreground hover:text-foreground/80 transition-colors group">
              <ChevronRightIcon className="h-3 w-3 text-muted-foreground group-data-[state=open]:rotate-90 transition-transform" />
              Tool Input
            </CollapsibleTrigger>
            <CollapsibleContent>
              <pre className="mt-1 p-2 bg-background border border-border rounded text-[11px] font-mono text-foreground overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all">
                {(() => { try { return JSON.stringify(JSON.parse(toolInput), null, 2); } catch { return toolInput; } })()}
              </pre>
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Response content */}
        {contentText && (
          <Collapsible defaultOpen>
            <CollapsibleTrigger className="flex items-center gap-1 text-xs font-medium text-foreground hover:text-foreground/80 transition-colors group">
              <ChevronRightIcon className="h-3 w-3 text-muted-foreground group-data-[state=open]:rotate-90 transition-transform" />
              Response
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-1 p-2 bg-background border border-border rounded text-[11px] text-foreground overflow-y-auto max-h-64 whitespace-pre-wrap">
                {contentText}
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Extra metadata (everything else) */}
        {meta &&
          Object.keys(meta).filter(
            (k) =>
              !["error", "toolInput", "thinking", "content", "checkpointSha", "prevCheckpointSHA", "checkpointSHA", "stage", "attempt", "role", "durationMs", "tool", "gen_ai.request.model", "gen_ai.usage.input_tokens", "gen_ai.usage.output_tokens", "gen_ai.usage.total_tokens", "gen_ai.usage.cache_read_tokens", "gen_ai.context.window_size", "gen_ai.context.utilization_pct", "agentRunId", "pipeline.result", "pipeline.stages", "pipeline.attempts", "result"].includes(k)
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
                          !["error", "toolInput", "thinking", "content", "checkpointSha", "prevCheckpointSHA", "checkpointSHA", "stage", "attempt", "role", "durationMs", "tool", "gen_ai.request.model", "gen_ai.usage.input_tokens", "gen_ai.usage.output_tokens", "gen_ai.usage.total_tokens", "gen_ai.usage.cache_read_tokens", "gen_ai.context.window_size", "gen_ai.context.utilization_pct", "agentRunId", "pipeline.result", "pipeline.stages", "pipeline.attempts", "result"].includes(k)
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
const STAGE_ROW_HEIGHT = 36;
const OVERSCAN = 8;
const INDENT_PX = 20;
const LABEL_WIDTH = 280;
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
  const [collapsedSpanIds, setCollapsedSpanIds] = useState<Set<string>>(
    new Set()
  );
  const prevSpanCount = useRef(spans.length);

  // Use controlled selectedSpanId if provided, otherwise use internal
  const selectedSpanId = controlledSelectedSpanId ?? internalSelectedSpanId;

  // Toggle collapse for stage spans
  const toggleCollapse = useCallback((spanId: string) => {
    setCollapsedSpanIds((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  }, []);

  // Build tree structure (collapse-aware)
  const { flat, traceDurationMs, traceStartMs } = useMemo(
    () => buildFlatTree(spans, collapsedSpanIds),
    [spans, collapsedSpanIds]
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

  // Compute positions for each row (stage spans are slightly taller)
  const rowPositions = useMemo(() => {
    const positions: { top: number; height: number; flatIndex: number }[] = [];
    let y = 0;
    for (let i = 0; i < flat.length; i++) {
      const h =
        flat[i].span.type === "stage" ? STAGE_ROW_HEIGHT : SPAN_ROW_HEIGHT;
      positions.push({ top: y, height: h, flatIndex: i });
      y += h;
    }
    return positions;
  }, [flat]);

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

  // Time axis ticks — show real timestamps
  const timeTicks = useMemo(() => {
    const count = 5;
    const ticks: { label: string; pct: number }[] = [];
    for (let i = 0; i <= count; i++) {
      const ms = (traceDurationMs / count) * i;
      const timestamp = new Date(traceStartMs + ms);
      const h = timestamp.getHours();
      const m = timestamp.getMinutes().toString().padStart(2, "0");
      const s = timestamp.getSeconds().toString().padStart(2, "0");
      const ampm = h >= 12 ? "pm" : "am";
      const h12 = h % 12 || 12;
      ticks.push({
        label: `${h12}:${m}:${s}${ampm}`,
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
        <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
          <span className="flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm bg-blue-500" /> thought</span>
          <span className="flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm bg-emerald-500" /> bash</span>
          <span className="flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm bg-violet-500" /> write</span>
          <span className="flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm bg-slate-400" /> read</span>
          <span className="flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm bg-red-500" /> error</span>
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
                  const { span, depth, durationMs, offsetMs, hasChildren } =
                    flat[pos.flatIndex];
                  const isStageSpan = span.type === "stage";
                  const isActive = !span.endTime;
                  const isSelected = span.id === selectedSpanId;
                  const isFailed =
                    span.metadata?.error !== undefined ||
                    span.status === "error" ||
                    (span.type === "tool" &&
                      span.metadata?.exitCode !== 0 &&
                      span.metadata?.exitCode !== undefined);
                  const isCollapsed = collapsedSpanIds.has(span.id);
                  const isComplete = !!span.endTime && !isFailed;

                  // Global timeline positioning — all bars on the same time axis
                  const barLeftPct = traceDurationMs > 0
                    ? (offsetMs / traceDurationMs) * 100
                    : 0;
                  // Width relative to parent for visibility, but clamped to trace bounds
                  const proportionalPct = traceDurationMs > 0
                    ? (durationMs / traceDurationMs) * 100
                    : 100;
                  const isInstant = durationMs <= 1; // 0ms or 1ms = thin vertical line
                  const barWidthPct = isInstant
                    ? 0.15
                    : Math.max(
                        isStageSpan ? 0.5 : 1.5,
                        Math.min(proportionalPct, 100 - barLeftPct)
                      );

                  // Operation-based coloring (error overrides)
                  const barClass = isFailed
                    ? FAILED_STYLES.bar
                    : isStageSpan
                      ? "" // stages don't get bars
                      : getOperationColor(span);

                  const textClass = isFailed
                    ? FAILED_STYLES.text
                    : isStageSpan
                      ? "text-foreground"
                      : getOperationTextColor(span);

                  const rowHeight = isStageSpan
                    ? STAGE_ROW_HEIGHT
                    : SPAN_ROW_HEIGHT;

                  // Stage aggregates for header display
                  const stageAgg = isStageSpan
                    ? computeStageAggregates(span, spans)
                    : null;

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
                        onClick={() => handleSelectSpan(span)}
                        className={cn(
                          "flex items-center w-full text-left transition-colors group",
                          isSelected
                            ? "bg-accent/20 border-l-2 border-l-primary"
                            : "hover:bg-muted/40",
                          isStageSpan && "bg-muted/10 border-b border-border/50"
                        )}
                        style={{ height: rowHeight }}
                      >
                        {/* Label column */}
                        <div
                          className="flex items-center gap-1.5 flex-shrink-0 px-2 h-full border-r border-border overflow-hidden"
                          style={{
                            width: LABEL_WIDTH,
                            paddingLeft: depth * INDENT_PX + 8,
                          }}
                        >
                          {/* Collapse toggle for stage/parent spans */}
                          {hasChildren ? (
                            <span
                              className="flex-shrink-0 cursor-pointer text-muted-foreground hover:text-foreground"
                              onClick={(e) => {
                                e.stopPropagation();
                                toggleCollapse(span.id);
                              }}
                            >
                              {isCollapsed ? (
                                <ChevronRightIcon className="h-3 w-3" />
                              ) : (
                                <ChevronDownIcon className="h-3 w-3" />
                              )}
                            </span>
                          ) : (
                            <span className="inline-block w-3 flex-shrink-0" />
                          )}

                          {/* Status dot: green=ok, red=error, gray=running */}
                          <span
                            className={cn(
                              "inline-block h-2 w-2 rounded-full flex-shrink-0",
                              isFailed
                                ? FAILED_STYLES.dot
                                : isComplete
                                  ? "bg-emerald-500"
                                  : "bg-muted-foreground/40"
                            )}
                          />

                          {/* Span name */}
                          <span
                            className={cn(
                              "text-[11px] font-mono truncate",
                              isStageSpan ? "font-bold text-foreground" : textClass
                            )}
                          >
                            {displaySpanName(span.name)}
                          </span>

                          {/* Stage status badge */}
                          {isStageSpan && span.status && (
                            <Badge
                              variant="outline"
                              className={cn(
                                "text-[8px] px-1 py-0 ml-auto flex-shrink-0",
                                span.status === "ok"
                                  ? "border-emerald-500/40 text-emerald-400"
                                  : span.status === "error"
                                    ? "border-red-500/40 text-red-400"
                                    : "border-muted-foreground/40 text-muted-foreground"
                              )}
                            >
                              {span.status === "ok"
                                ? "PASS"
                                : span.status === "error"
                                  ? "FAIL"
                                  : span.status.toUpperCase()}
                            </Badge>
                          )}

                          {/* Diff line count indicator */}
                          {span.hasDiff && (
                            <span className="ml-auto flex-shrink-0 text-[9px] font-mono text-muted-foreground">
                              <span className="text-emerald-400">+{String(span.metadata?.["diff.additions"] ?? "?")}</span>
                              <span className="text-muted-foreground/40">/</span>
                              <span className="text-red-400">-{String(span.metadata?.["diff.deletions"] ?? "?")}</span>
                            </span>
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

                          {isStageSpan ? (
                            /* Stage row: waterfall bar + summary stats */
                            <>
                              {/* Stage bar — shows position within parent timeline */}
                              <div
                                className={cn(
                                  "absolute rounded-sm h-6",
                                  isFailed
                                    ? "bg-red-500/15 border border-red-500/30"
                                    : "bg-amber-500/10 border border-amber-500/20"
                                )}
                                style={{
                                  left: `${Math.min(barLeftPct, 97)}%`,
                                  width: `${Math.min(barWidthPct, 100 - Math.min(barLeftPct, 97))}%`,
                                  minWidth: MIN_BAR_WIDTH_PX,
                                }}
                              />
                              {/* Stats label after the bar */}
                              <span
                                className="absolute text-[10px] font-mono text-muted-foreground whitespace-nowrap flex items-center gap-2"
                                style={{
                                  left: `calc(${Math.min(barLeftPct + barWidthPct, 100)}% + 4px)`,
                                }}
                              >
                                <span>{formatDuration(durationMs)}</span>
                                {stageAgg && (
                                  <>
                                    <span className="text-border">·</span>
                                    <span>{stageAgg.childCount} spans</span>
                                    {stageAgg.estimatedCostUsd > 0 && (
                                      <>
                                        <span className="text-border">·</span>
                                        <span>{formatCost(stageAgg.estimatedCostUsd)}</span>
                                      </>
                                    )}
                                    {stageAgg.toolErrors > 0 && (
                                      <>
                                        <span className="text-border">·</span>
                                        <span className="text-red-400">{stageAgg.toolErrors} err</span>
                                      </>
                                    )}
                                  </>
                                )}
                              </span>
                            </>
                          ) : (
                            /* Leaf span: waterfall bar positioned relative to parent */
                            <>
                              <div
                                className={cn(
                                  "absolute transition-all",
                                  isInstant ? "h-5 w-px rounded-none" : "h-5 rounded-sm",
                                  barClass,
                                  isActive && "animate-pulse"
                                )}
                                style={{
                                  left: `${Math.min(barLeftPct, 97)}%`,
                                  width: `${Math.min(barWidthPct, 100 - Math.min(barLeftPct, 97))}%`,
                                  minWidth: MIN_BAR_WIDTH_PX,
                                }}
                              />

                              {/* Duration label — positioned after the bar end to avoid overlap */}
                              <span
                                className="absolute text-[10px] font-mono text-muted-foreground whitespace-nowrap"
                                style={{
                                  left: `calc(${Math.min(barLeftPct + barWidthPct, 100)}% + 4px)`,
                                }}
                              >
                                <span className="text-foreground/70">{formatDuration(durationMs)}</span>
                                {/* Inline diff stats for tool spans */}
                                {span.hasDiff && (
                                  <>
                                    <span className="text-border mx-1">·</span>
                                    <span className="text-emerald-400/70">+{String(span.metadata?.["diff.additions"] ?? "?")}</span>
                                    <span className="text-muted-foreground/40">/</span>
                                    <span className="text-red-400/70">-{String(span.metadata?.["diff.deletions"] ?? "?")}</span>
                                  </>
                                )}
                                {/* Inline metrics for thought spans — dot-separated, semantic colors */}
                                {span.type === "llm" && (span.metadata?.["gen_ai.usage.input_tokens"] as number) > 0 && (() => {
                                  const inTok = (span.metadata?.["gen_ai.usage.input_tokens"] as number) || 0;
                                  const outTok = (span.metadata?.["gen_ai.usage.output_tokens"] as number) || 0;
                                  const ctxPct = (span.metadata?.["gen_ai.context.utilization_pct"] as number) || 0;
                                  const cost = (inTok * 0.15 + outTok * 0.75) / 1_000_000;
                                  return (
                                    <>
                                      <span className="text-border mx-1">·</span>
                                      <span className="text-blue-400/70">{Math.round(inTok / 1000)}k in</span>
                                      <span className="text-border mx-0.5">·</span>
                                      <span className="text-violet-400/70">{outTok} out</span>
                                      {ctxPct > 0 && (
                                        <>
                                          <span className="text-border mx-0.5">·</span>
                                          <span className={cn(
                                            ctxPct > 80 ? "text-red-400/70" : ctxPct > 50 ? "text-amber-400/70" : "text-emerald-400/70"
                                          )}>{ctxPct}% ctx</span>
                                        </>
                                      )}
                                      {cost > 0.0001 && (
                                        <>
                                          <span className="text-border mx-0.5">·</span>
                                          <span className="text-amber-400/70">{formatCost(cost)}</span>
                                        </>
                                      )}
                                    </>
                                  );
                                })()}
                              </span>
                            </>
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
                allSpans={spans}
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
