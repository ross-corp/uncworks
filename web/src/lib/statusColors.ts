import type { AgentRunPhase } from "../types/agent-run";

export interface StatusColorTokens {
  /** CSS custom property for the foreground/dot color, e.g. "var(--color-success)" */
  color: string;
  /** CSS custom property for the muted/background variant */
  mutedColor: string;
  /** Human-readable semantic name */
  semantic: "success" | "active" | "warning" | "error" | "neutral";
}

const STATUS_COLOR_MAP: Record<AgentRunPhase, StatusColorTokens> = {
  succeeded: {
    color: "var(--color-success)",
    mutedColor: "var(--color-success-muted)",
    semantic: "success",
  },
  running: {
    color: "var(--color-active)",
    mutedColor: "var(--color-active-muted)",
    semantic: "active",
  },
  waiting_for_input: {
    color: "var(--color-active)",
    mutedColor: "var(--color-active-muted)",
    semantic: "active",
  },
  pending: {
    color: "var(--color-warning)",
    mutedColor: "var(--color-warning-muted)",
    semantic: "warning",
  },
  failed: {
    color: "var(--color-error)",
    mutedColor: "var(--color-error-muted)",
    semantic: "error",
  },
  cancelled: {
    color: "var(--color-neutral)",
    mutedColor: "var(--color-neutral-muted)",
    semantic: "neutral",
  },
};

export function getStatusColors(phase: AgentRunPhase): StatusColorTokens {
  return STATUS_COLOR_MAP[phase];
}
