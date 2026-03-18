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

    apiFetch(`/api/v1/runs/${runId}/traces`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((data: TraceSpan[]) => {
        if (!cancelled) setSpans(data ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          console.error("Failed to fetch traces:", err);
          setSpans([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [runId]);

  return { spans, loading };
}
