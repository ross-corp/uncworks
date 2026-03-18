import { useState, useRef, useEffect, useCallback } from "react";
import type { TraceSpan } from "../types/agent-run";
import { Badge } from "./ui/badge";

// ============================================================
// Trace Timeline v2 — MU-TH-UR aesthetic
// Horizontal span bars with phosphor glow, CRT scanlines,
// monospace labels, and interactive tooltips/click handling.
// ============================================================

const SPAN_TYPE_CONFIG: Record<
  string,
  { color: string; activeColor: string; label: string }
> = {
  llm: {
    color: "var(--muthr-dim-green)",
    activeColor: "var(--muthr-green)",
    label: "LLM",
  },
  tool: {
    color: "var(--muthr-dim-green)",
    activeColor: "var(--muthr-green)",
    label: "TOOL",
  },
  thought: {
    color: "var(--muthr-dim-green)",
    activeColor: "var(--muthr-green)",
    label: "THINK",
  },
  input: {
    color: "var(--muthr-dim-green)",
    activeColor: "var(--muthr-green)",
    label: "INPUT",
  },
};

function formatDuration(startTime: string, endTime: string): string {
  const start = new Date(startTime).getTime();
  const end = endTime ? new Date(endTime).getTime() : Date.now();
  const ms = end - start;
  if (ms < 1000) return `${ms}ms`;
  const secs = Math.floor(ms / 1000);
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

function formatTime(iso: string): string {
  if (!iso) return "--";
  return new Date(iso).toLocaleTimeString();
}

/** Span item height for virtualization */
const SPAN_HEIGHT = 36;
const OVERSCAN = 5;

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
  const [hoveredSpanId, setHoveredSpanId] = useState<string | null>(null);
  const [userScrolled, setUserScrolled] = useState(false);
  const prevSpanCount = useRef(spans.length);

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

  // Auto-scroll to latest span
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
    // Detect if user has scrolled away from bottom
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setUserScrolled(!atBottom);
  }, []);

  // Virtualization
  const totalHeight = spans.length * SPAN_HEIGHT;
  const startIndex = Math.max(0, Math.floor(scrollTop / SPAN_HEIGHT) - OVERSCAN);
  const endIndex = Math.min(
    spans.length,
    Math.ceil((scrollTop + containerHeight) / SPAN_HEIGHT) + OVERSCAN
  );
  const visibleSpans = spans.slice(startIndex, endIndex);

  if (spans.length === 0) {
    return (
      <div
        className="flex items-center justify-center py-8 muthr-scanlines"
        style={{
          background: "var(--muthr-bg)",
          fontFamily: "var(--muthr-font)",
        }}
      >
        <span
          className="text-[11px] uppercase tracking-widest"
          style={{ color: "var(--muthr-dim-green)" }}
        >
          No trace spans recorded
        </span>
      </div>
    );
  }

  return (
    <div
      data-testid="trace-timeline"
      className="flex flex-col muthr-scanlines"
      style={{ background: "var(--muthr-bg)", fontFamily: "var(--muthr-font)" }}
    >
      {/* Timeline header */}
      <div
        className="flex items-center justify-between px-3 py-2 border-b"
        style={{ borderColor: "var(--muthr-dim-green)" }}
      >
        <div className="flex items-center gap-2">
          {runId && (
            <span
              className="text-[10px] uppercase tracking-widest truncate max-w-[200px]"
              style={{ color: "var(--muthr-green)" }}
            >
              {runId}
            </span>
          )}
          {agentType && (
            <span
              className="text-[10px] uppercase tracking-widest"
              style={{ color: "var(--muthr-dim-green)" }}
            >
              {agentType}
            </span>
          )}
        </div>
        {/* Legend */}
        <div className="flex items-center gap-3">
          {Object.entries(SPAN_TYPE_CONFIG).map(([type, config]) => (
            <div key={type} className="flex items-center gap-1">
              <span
                className="inline-block h-2 w-2"
                style={{ background: config.activeColor }}
              />
              <span
                className="text-[10px] uppercase tracking-wider"
                style={{ color: "var(--muthr-dim-green)" }}
              >
                {config.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Virtualized span list */}
      <div
        ref={containerRef}
        className="overflow-y-auto"
        style={{ maxHeight: "300px" }}
        onScroll={handleScroll}
      >
        <div style={{ height: totalHeight, position: "relative" }}>
          {visibleSpans.map((span, i) => {
            const index = startIndex + i;
            const config = SPAN_TYPE_CONFIG[span.type] ?? SPAN_TYPE_CONFIG.tool;
            const isActive = !span.endTime;
            const isSelected = span.id === selectedSpanId;
            const isHovered = span.id === hoveredSpanId;
            const isFailed = span.metadata?.error !== undefined;

            let barColor = config.color;
            if (isActive) barColor = config.activeColor;
            if (isFailed) barColor = "var(--muthr-amber)";

            return (
              <div
                key={span.id}
                style={{
                  position: "absolute",
                  top: index * SPAN_HEIGHT,
                  left: 0,
                  right: 0,
                  height: SPAN_HEIGHT,
                }}
              >
                <button
                  onClick={() => onSelectSpan(span)}
                  onMouseEnter={() => setHoveredSpanId(span.id)}
                  onMouseLeave={() => setHoveredSpanId(null)}
                  className="flex items-center gap-2 w-full h-full px-3 text-left transition-colors relative"
                  style={{
                    background: isSelected
                      ? "rgba(0, 255, 65, 0.08)"
                      : isHovered
                        ? "rgba(0, 255, 65, 0.04)"
                        : "transparent",
                    borderLeft: isSelected ? `2px solid var(--muthr-green)` : "2px solid transparent",
                  }}
                >
                  {/* Span type indicator */}
                  <span
                    className="inline-block h-3 w-3 flex-shrink-0"
                    style={{
                      background: barColor,
                      boxShadow: isActive ? `0 0 8px ${barColor}` : "none",
                    }}
                  />

                  {/* Span label */}
                  <span
                    className="flex-1 truncate text-[11px] uppercase tracking-wider"
                    style={{
                      color: isActive
                        ? "var(--muthr-green)"
                        : isFailed
                          ? "var(--muthr-amber)"
                          : "var(--muthr-dim-green)",
                    }}
                  >
                    {span.name}
                  </span>

                  {/* Duration */}
                  <span
                    className="flex-shrink-0 text-[10px]"
                    style={{ color: "var(--muthr-dim-green)" }}
                  >
                    {formatDuration(span.startTime, span.endTime)}
                  </span>

                  {/* Diff badge */}
                  {span.hasDiff && (
                    <Badge
                      variant="outline"
                      className="text-[9px] px-1 py-0 border-[var(--muthr-green)] ml-1"
                      style={{ color: "var(--muthr-green)" }}
                    >
                      DIFF
                    </Badge>
                  )}

                  {/* Active indicator (growing bar) */}
                  {isActive && (
                    <span
                      className="absolute right-0 top-0 bottom-0 w-1"
                      style={{
                        background: "var(--muthr-green)",
                        boxShadow: `0 0 8px var(--muthr-green)`,
                        animation: "muthr-phosphor-pulse 1.5s ease-in-out infinite",
                      }}
                    />
                  )}
                </button>

                {/* Tooltip on hover */}
                {isHovered && (
                  <div
                    className="absolute z-50 px-3 py-2 pointer-events-none"
                    style={{
                      top: SPAN_HEIGHT,
                      left: "50%",
                      transform: "translateX(-50%)",
                      background: "var(--muthr-bg)",
                      border: "1px solid var(--muthr-dim-green)",
                      fontFamily: "var(--muthr-font)",
                      minWidth: 200,
                    }}
                  >
                    <div className="text-[10px] uppercase tracking-wider" style={{ color: "var(--muthr-green)" }}>
                      {span.name}
                    </div>
                    <div className="text-[10px]" style={{ color: "var(--muthr-dim-green)" }}>
                      Start: {formatTime(span.startTime)}
                    </div>
                    <div className="text-[10px]" style={{ color: "var(--muthr-dim-green)" }}>
                      Duration: {formatDuration(span.startTime, span.endTime)}
                    </div>
                    {span.hasDiff && (
                      <div className="text-[10px] mt-1" style={{ color: "var(--muthr-green)" }}>
                        Click to view diff
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
