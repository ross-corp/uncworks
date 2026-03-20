import { Badge } from "./ui/badge";

type Phase = string;

type BadgeVariant = "default" | "secondary" | "destructive" | "outline" | "glow" | "warning" | "danger";

const STATUS_CONFIG: Record<string, { icon: string; variant: BadgeVariant; className?: string }> = {
  running: { icon: "\u25CF", variant: "glow", className: "animate-pulse" },
  succeeded: { icon: "\u2713", variant: "outline", className: "border-green-500/40 text-green-500" },
  failed: { icon: "\u2717", variant: "destructive" },
  pending: { icon: "\u25CE", variant: "secondary" },
  waiting_for_input: { icon: "\u23F8", variant: "warning" },
  cancelled: { icon: "\u2298", variant: "outline" },
};

export default function RunStatusBadge({ phase }: { phase: Phase }) {
  const config = STATUS_CONFIG[phase] ?? STATUS_CONFIG.pending;
  const label = phase === "waiting_for_input" ? "waiting" : phase;
  return (
    <Badge variant={config.variant} className={config.className}>
      {config.icon} {label}
    </Badge>
  );
}
