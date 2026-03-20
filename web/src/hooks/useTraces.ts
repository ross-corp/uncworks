import { useState, useEffect } from "react";
import type { TraceSpan } from "../types/agent-run";
import { apiFetch } from "./apiFetch";

export function useTraces(runId: string) {
  const [spans, setSpans] = useState<TraceSpan[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!runId) {
      setSpans([]);
      return;
    }

    let cancelled = false;
    setLoading(true);

    async function fetchTraces() {
      try {
        const r = await apiFetch(`/api/v1/runs/${runId}/traces`);
        if (r.ok) {
          const data: TraceSpan[] = await r.json();
          if (!cancelled) setSpans(data ?? []);
        }
      } catch {
        // silent — traces may not exist yet during hydration
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchTraces();
    const interval = setInterval(fetchTraces, 5000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [runId]);

  return { spans, loading };
}
