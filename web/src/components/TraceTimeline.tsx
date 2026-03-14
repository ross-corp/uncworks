import type { TraceSpan } from "../types/agent-run";

const TYPE_COLORS: Record<string, string> = {
  llm: "bg-blue-500",
  tool: "bg-green-500",
  thought: "bg-purple-500",
  input: "bg-orange-500",
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
      <div className="flex items-center justify-center py-8 text-sm text-txt-tertiary">
        No trace spans recorded
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 overflow-x-auto p-2">
      {/* Legend */}
      <div className="mb-2 flex items-center gap-3">
        {Object.entries(TYPE_COLORS).map(([type, color]) => (
          <div key={type} className="flex items-center gap-1">
            <span className={`inline-block h-2.5 w-2.5 rounded-sm ${color}`} />
            <span className="text-xs text-txt-tertiary">{TYPE_LABELS[type]}</span>
          </div>
        ))}
      </div>

      {/* Timeline bars */}
      <div className="flex flex-col gap-1">
        {spans.map((span) => {
          const color = TYPE_COLORS[span.type] ?? "bg-gray-500";
          const isSelected = span.id === selectedSpanId;
          const duration = formatDuration(span.startTime, span.endTime);

          return (
            <button
              key={span.id}
              onClick={() => onSelectSpan(span)}
              className={`flex items-center gap-2 rounded px-3 py-1.5 text-left transition-colors ${
                isSelected
                  ? "bg-surface-2 ring-1 ring-accent"
                  : "hover:bg-surface-2/50"
              }`}
            >
              <span className={`inline-block h-3 w-3 flex-shrink-0 rounded-sm ${color}`} />
              <span className="flex-1 truncate text-xs font-medium text-txt-secondary">
                {span.name}
              </span>
              <span className="flex-shrink-0 text-xs text-txt-tertiary">
                {duration}
              </span>
              {span.hasDiff && (
                <span className="flex-shrink-0 rounded bg-green-500/20 px-1.5 py-0.5 text-[10px] font-medium text-green-400">
                  diff
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
