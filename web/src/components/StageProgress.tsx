const STAGES = ["plan", "execute", "verify"] as const;

type StageStatus = "done" | "active" | "pending" | "failed";

function getStageStatus(stage: string | undefined, phase: string | undefined, s: string): StageStatus {
  const idx = STAGES.indexOf(s as typeof STAGES[number]);

  // Map stage names: "planning" → plan, "executing" → execute, "verifying" → verify
  let normalizedCurrent = -1;
  if (stage === "planning") normalizedCurrent = 0;
  else if (stage === "executing") normalizedCurrent = 1;
  else if (stage === "verifying") normalizedCurrent = 2;

  if (phase === "failed" || phase === "cancelled") {
    if (idx < normalizedCurrent) return "done";
    if (idx === normalizedCurrent) return "failed";
    return "pending";
  }
  if (phase === "succeeded") return "done";

  if (idx < normalizedCurrent) return "done";
  if (idx === normalizedCurrent) return "active";
  return "pending";
}

const STATUS_STYLES: Record<StageStatus, string> = {
  done: "text-green-500",
  active: "text-primary font-medium",
  pending: "text-muted-foreground/40",
  failed: "text-red-500",
};

const STATUS_ICONS: Record<StageStatus, string> = {
  done: "✓",
  active: "...",
  pending: "○",
  failed: "✗",
};

export default function StageProgress({ stage, phase }: { stage?: string; phase?: string }) {
  return (
    <div className="flex items-center gap-3 text-xs">
      {STAGES.map((s, i) => {
        const status = getStageStatus(stage, phase, s);
        return (
          <div key={s} className="flex items-center gap-1">
            {i > 0 && <span className="text-muted-foreground/30 mx-1">→</span>}
            <span className={STATUS_STYLES[status]}>
              {STATUS_ICONS[status]} {s}
            </span>
          </div>
        );
      })}
    </div>
  );
}
