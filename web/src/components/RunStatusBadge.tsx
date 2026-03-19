type Phase = string;

const STATUS_CONFIG: Record<string, { icon: string; color: string }> = {
  running: { icon: "●", color: "text-green-500" },
  succeeded: { icon: "✓", color: "text-green-500" },
  failed: { icon: "✗", color: "text-red-500" },
  pending: { icon: "◎", color: "text-muted-foreground" },
  waiting_for_input: { icon: "⏸", color: "text-yellow-500" },
  cancelled: { icon: "⊘", color: "text-muted-foreground" },
};

export default function RunStatusBadge({ phase }: { phase: Phase }) {
  const config = STATUS_CONFIG[phase] ?? STATUS_CONFIG.pending;
  return (
    <span className={`flex items-center gap-1 text-xs ${config.color}`}>
      <span className={phase === "running" ? "animate-pulse" : ""}>{config.icon}</span>
      <span>{phase === "waiting_for_input" ? "waiting" : phase}</span>
    </span>
  );
}
