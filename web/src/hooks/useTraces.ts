import { useState, useEffect } from "react";
import type { TraceSpan } from "../types/agent-run";
import { apiFetch } from "./apiFetch";
import { usePoll } from "./usePoll";

export function useTraces(runId: string) {
  const [spans, setSpans] = useState<TraceSpan[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!runId) setSpans([]);
  }, [runId]);

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
  }, 5000, [runId]);

  return { spans, loading };
}
