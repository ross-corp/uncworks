import { Progress } from "./ui/progress";
import { Badge } from "./ui/badge";

const STAGES = ["plan", "execute", "verify"] as const;

type StageStatus = "done" | "active" | "pending" | "failed";

function getStageStatus(stage: string | undefined, phase: string | undefined, s: string): StageStatus {
  const idx = STAGES.indexOf(s as typeof STAGES[number]);

  // Map stage names: "planning" -> plan, "executing" -> execute, "verifying" -> verify
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

type BadgeVariant = "default" | "secondary" | "destructive" | "outline" | "glow" | "warning" | "danger";

const STATUS_BADGE: Record<StageStatus, { variant: BadgeVariant; className?: string }> = {
  done: { variant: "outline", className: "border-green-500/40 text-green-500" },
  active: { variant: "glow" },
  pending: { variant: "secondary" },
  failed: { variant: "danger" },
};

function computeProgress(stage: string | undefined, phase: string | undefined): number {
  if (phase === "succeeded") return 100;

  let normalizedCurrent = -1;
  if (stage === "planning") normalizedCurrent = 0;
  else if (stage === "executing") normalizedCurrent = 1;
  else if (stage === "verifying") normalizedCurrent = 2;

  // Each stage is ~33%, active stage counts as half-done
  const doneStages = Math.max(0, normalizedCurrent);
  const partial = normalizedCurrent >= 0 ? 0.5 : 0;
  return Math.round(((doneStages + partial) / STAGES.length) * 100);
}

export default function StageProgress({ stage, phase }: { stage?: string; phase?: string }) {
  const progress = computeProgress(stage, phase);

  return (
    <div className="flex items-center gap-3">
      <div className="flex items-center gap-1.5">
        {STAGES.map((s) => {
          const status = getStageStatus(stage, phase, s);
          const config = STATUS_BADGE[status];
          return (
            <Badge key={s} variant={config.variant} className={config.className}>
              {s}
            </Badge>
          );
        })}
      </div>
      <Progress value={progress} className="h-2 flex-1" />
      <span className="text-[10px] font-mono text-muted-foreground w-8 text-right">{progress}%</span>
    </div>
  );
}
