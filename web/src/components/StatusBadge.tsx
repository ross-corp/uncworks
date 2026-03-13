import type { AgentRunPhase, Backend, ModelTier } from "../types/agent-run";

const PHASE_STYLES: Record<AgentRunPhase, string> = {
  pending: "bg-amber-500/15 text-amber-400",
  running: "bg-blue-500/15 text-blue-400 animate-pulse",
  waiting_for_input: "bg-purple-500/15 text-purple-400",
  succeeded: "bg-green-500/15 text-green-400",
  failed: "bg-red-500/15 text-red-400",
  cancelled: "bg-stone-700/50 text-stone-300",
};

const PHASE_LABELS: Record<AgentRunPhase, string> = {
  pending: "Pending",
  running: "Running",
  waiting_for_input: "Waiting for Input",
  succeeded: "Succeeded",
  failed: "Failed",
  cancelled: "Cancelled",
};

export function PhaseBadge({ phase }: { phase: AgentRunPhase }) {
  return (
    <span
      className={`inline-flex items-center rounded px-2 py-0.5 font-mono text-xs ${PHASE_STYLES[phase]}`}
    >
      {PHASE_LABELS[phase]}
    </span>
  );
}

const BACKEND_STYLES: Record<Backend, string> = {
  pod: "bg-blue-500/15 text-blue-400",
  kubevirt: "bg-violet-500/15 text-violet-400",
  external: "bg-teal-500/15 text-teal-400",
};

const BACKEND_LABELS: Record<Backend, string> = {
  pod: "Pod",
  kubevirt: "KubeVirt",
  external: "External",
};

export function BackendBadge({ backend }: { backend: Backend }) {
  return (
    <span
      className={`inline-flex items-center rounded px-2 py-0.5 font-mono text-xs ${BACKEND_STYLES[backend]}`}
    >
      {BACKEND_LABELS[backend]}
    </span>
  );
}

const TIER_STYLES: Record<ModelTier, string> = {
  default: "bg-stone-700/50 text-stone-300",
  "default-cloud": "bg-sky-500/15 text-sky-400",
  premium: "bg-amber-500/15 text-amber-400",
};

const TIER_LABELS: Record<ModelTier, string> = {
  default: "Local",
  "default-cloud": "Cloud",
  premium: "Premium",
};

export function ModelTierBadge({ tier }: { tier: ModelTier }) {
  return (
    <span
      className={`inline-flex items-center rounded px-2 py-0.5 font-mono text-xs ${TIER_STYLES[tier]}`}
    >
      {TIER_LABELS[tier]}
    </span>
  );
}
