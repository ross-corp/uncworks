import type { TraceSpan } from "../types/agent-run";
import { Badge } from "./ui/badge";

const TYPE_COLORS: Record<string, string> = {
  llm: "bg-primary",
  tool: "bg-secondary",
  thought: "bg-secondary/60",
  input: "bg-primary/60",
};

const TYPE_LABELS: Record<string, string> = {
  llm: "LLM",
  tool: "Tool",
  thought: "Thought",
  input: "Input",
};

function formatDuration(startTime: string, endTime: string): string {
  const start = new Date(startTime).getTime();
  const end = new Date(endTime).getTime();
  const ms = end - start;
  if (ms < 1000) return `${ms}ms`;
  const secs = Math.floor(ms / 1000);
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

export default function TraceTimeline({
  spans,
  selectedSpanId,
  onSelectSpan,
}: {
  spans: TraceSpan[];
  selectedSpanId?: string;
  onSelectSpan: (span: TraceSpan) => void;
}) {
  if (spans.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-sm text-muted-foreground/60">
        No trace spans recorded
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 overflow-x-auto p-2 fx-scanlines">
      {/* Legend */}
      <div className="mb-2 flex items-center gap-3">
        {Object.entries(TYPE_COLORS).map(([type, color]) => (
          <div key={type} className="flex items-center gap-1">
            <span className={`inline-block h-2.5 w-2.5 ${color}`} />
            <span className="text-xs text-muted-foreground/60">{TYPE_LABELS[type]}</span>
          </div>
        ))}
      </div>

      {/* Timeline bars */}
      <div className="flex flex-col gap-1">
        {spans.map((span) => {
          const color = TYPE_COLORS[span.type] ?? "bg-muted";
          const isSelected = span.id === selectedSpanId;
          const duration = formatDuration(span.startTime, span.endTime);

          return (
            <button
              key={span.id}
              onClick={() => onSelectSpan(span)}
              className={`flex items-center gap-2 px-3 py-1.5 text-left transition-colors ${
                isSelected
                  ? "bg-muted ring-1 ring-primary fx-glow"
                  : "hover:bg-muted/50"
              }`}
            >
              <span className={`inline-block h-3 w-3 flex-shrink-0 ${color}`} />
              <span className="flex-1 truncate text-xs font-medium text-muted-foreground">
                {span.name}
              </span>
              <span className="flex-shrink-0 text-xs text-muted-foreground/60">
                {duration}
              </span>
              {span.hasDiff && (
                <Badge variant="outline" className="border-secondary/30 text-secondary text-[10px] px-1.5 py-0.5">
                  diff
                </Badge>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
