import type { AgentRunPhase, Backend, ModelTier } from "../types/agent-run";
import { Badge } from "./ui/badge";
import { getStatusColors } from "../lib/statusColors";

const PHASE_LABELS: Record<AgentRunPhase, string> = {
  pending: "Pending",
  running: "Running",
  waiting_for_input: "Waiting for Input",
  succeeded: "Succeeded",
  failed: "Failed",
  cancelled: "Cancelled",
};

export function PhaseBadge({ phase }: { phase: AgentRunPhase }) {
  const { color, mutedColor } = getStatusColors(phase);
  const isRunning = phase === "running";

  return (
    <Badge
      variant="outline"
      className="font-mono text-xs border-transparent"
      style={{
        backgroundColor: mutedColor,
        color: color,
        borderColor: "transparent",
        animation: isRunning ? "status-pulse 2s ease-in-out infinite" : undefined,
      }}
    >
      {PHASE_LABELS[phase]}
    </Badge>
  );
}

const BACKEND_VARIANT: Record<Backend, "default" | "secondary" | "outline"> = {
  pod: "secondary",
  kubevirt: "outline",
  external: "outline",
};

const BACKEND_EXTRA: Record<Backend, string> = {
  pod: "",
  kubevirt: "border-secondary/30 text-secondary",
  external: "border-primary/30 text-primary",
};

const BACKEND_LABELS: Record<Backend, string> = {
  pod: "Pod",
  kubevirt: "KubeVirt",
  external: "External",
};

export function BackendBadge({ backend }: { backend: Backend }) {
  return (
    <Badge variant={BACKEND_VARIANT[backend]} className={`font-mono text-xs ${BACKEND_EXTRA[backend]}`}>
      {BACKEND_LABELS[backend]}
    </Badge>
  );
}

const TIER_VARIANT: Record<ModelTier, "default" | "secondary" | "outline"> = {
  default: "secondary",
  "default-cloud": "outline",
  premium: "outline",
};

const TIER_EXTRA: Record<ModelTier, string> = {
  default: "opacity-70",
  "default-cloud": "border-secondary/30 text-secondary",
  premium: "border-primary/30 text-primary",
};

const TIER_LABELS: Record<ModelTier, string> = {
  default: "Local",
  "default-cloud": "Cloud",
  premium: "Premium",
};

export function ModelTierBadge({ tier }: { tier: ModelTier }) {
  return (
    <Badge variant={TIER_VARIANT[tier]} className={`font-mono text-xs ${TIER_EXTRA[tier]}`}>
      {TIER_LABELS[tier]}
    </Badge>
  );
}
