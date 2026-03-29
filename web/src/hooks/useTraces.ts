import { useState, useEffect } from "react";
import type { TraceSpan, AgentRunPhase } from "../types/agent-run";
import { apiFetch } from "./apiFetch";
import { usePoll } from "./usePoll";

const TERMINAL_PHASES = new Set<AgentRunPhase>(["succeeded", "failed", "cancelled"]);

export function useTraces(runId: string, phase?: AgentRunPhase) {
  const [spans, setSpans] = useState<TraceSpan[]>([]);
  const [loading, setLoading] = useState(false);
  const isTerminal = phase ? TERMINAL_PHASES.has(phase) : false;

  useEffect(() => {
    if (!runId) setSpans([]);
  }, [runId]);

  // Use a longer interval for terminal runs to avoid unnecessary polling.
  // usePoll fires immediately on mount so we always get an initial fetch.
  usePoll(async () => {
    if (!runId) return;
    setLoading(true);
    try {
      const r = await apiFetch(`/api/v1/runs/${runId}/traces`);
      if (r.ok) {
        const data: TraceSpan[] = await r.json();
        setSpans(data ?? []);
      }
    } catch (err) {
      // traces may not exist yet during hydration — log for debugging
      console.error("[useTraces]", err);
    } finally {
      setLoading(false);
    }
  }, isTerminal ? 60_000 : 5_000, [runId, isTerminal]);

  return { spans, loading };
}
