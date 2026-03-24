import { Badge } from "./ui/badge";

type Phase = string;

type BadgeVariant = "default" | "secondary" | "destructive" | "outline" | "glow" | "warning" | "danger";

const STATUS_CONFIG: Record<string, { variant: BadgeVariant; className?: string; dot?: string }> = {
  running: { variant: "glow", className: "animate-pulse", dot: "bg-blue-500" },
  succeeded: { variant: "outline", className: "border-green-500/40 text-green-600 dark:text-green-400", dot: "bg-green-500" },
  failed: { variant: "destructive", dot: "bg-red-500" },
  pending: { variant: "secondary", dot: "bg-neutral-400" },
  waiting_for_input: { variant: "warning", dot: "bg-amber-500" },
  cancelled: { variant: "outline", className: "text-muted-foreground", dot: "bg-neutral-400" },
};

interface RunStatusBadgeProps {
  phase: Phase;
  stage?: string;
}

export default function RunStatusBadge({ phase, stage }: RunStatusBadgeProps) {
  const config = STATUS_CONFIG[phase] ?? STATUS_CONFIG.pending;

  let label = phase;
  if (phase === "waiting_for_input") label = "waiting";
  if (phase === "pending") label = "queued";

  let stageLabel = "";
  if (stage && (phase === "running" || phase === "pending")) {
    stageLabel = stage.toLowerCase();
  }

  return (
    <Badge variant={config.variant} className={config.className}>
      <span className={`inline-block w-1.5 h-1.5 rounded-full ${config.dot || "bg-current"} ${phase === "running" ? "animate-pulse" : ""}`} />
      {label}{stageLabel ? ` / ${stageLabel}` : ""}
    </Badge>
  );
}
