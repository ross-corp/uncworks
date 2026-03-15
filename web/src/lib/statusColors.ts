import type { AgentRunPhase } from "../types/agent-run";

export interface StatusColorTokens {
  /** CSS custom property for the foreground/dot color, e.g. "var(--unc-success)" */
  color: string;
  /** CSS custom property for the muted/background variant */
  mutedColor: string;
  /** Human-readable semantic name */
  semantic: "success" | "active" | "warning" | "error" | "neutral";
}

const STATUS_COLOR_MAP: Record<AgentRunPhase, StatusColorTokens> = {
  succeeded: {
    color: "var(--unc-success)",
    mutedColor: "var(--unc-success-muted)",
    semantic: "success",
  },
  running: {
    color: "var(--unc-active)",
    mutedColor: "var(--unc-active-muted)",
    semantic: "active",
  },
  waiting_for_input: {
    color: "var(--unc-active)",
    mutedColor: "var(--unc-active-muted)",
    semantic: "active",
  },
  pending: {
    color: "var(--unc-warning)",
    mutedColor: "var(--unc-warning-muted)",
    semantic: "warning",
  },
  failed: {
    color: "var(--unc-error)",
    mutedColor: "var(--unc-error-muted)",
    semantic: "error",
  },
  cancelled: {
    color: "var(--unc-neutral)",
    mutedColor: "var(--unc-neutral-muted)",
    semantic: "neutral",
  },
};

export function getStatusColors(phase: AgentRunPhase): StatusColorTokens {
  return STATUS_COLOR_MAP[phase];
}
