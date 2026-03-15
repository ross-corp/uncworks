import { useState, useEffect, useRef } from "react";
import type { TraceSpan } from "../types/agent-run";

/**
 * Hook that fetches trace spans for a given run ID and subscribes
 * to real-time span updates via SSE.
 */
export function useTraceSpans(runId: string | null) {
  const [spans, setSpans] = useState<TraceSpan[]>([]);
  const [loading, setLoading] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!runId) {
      setSpans([]);
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);

    // Fetch initial spans
    fetch(`/api/v1/runs/${runId}/traces`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((data: TraceSpan[]) => {
        if (!cancelled) setSpans(data ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          console.error("Failed to fetch trace spans:", err);
          setSpans([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    // Subscribe to real-time span updates
    const eventSource = new EventSource(`/api/v1/runs/${runId}/traces/watch`);
    eventSourceRef.current = eventSource;

    eventSource.onmessage = (msg) => {
      try {
        const span = JSON.parse(msg.data) as TraceSpan;
        if (!cancelled) {
          setSpans((prev) => {
            const existing = prev.findIndex((s) => s.id === span.id);
            if (existing >= 0) {
              const updated = [...prev];
              updated[existing] = span;
              return updated;
            }
            return [...prev, span];
          });
        }
      } catch {
        // Ignore malformed events
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => {
      cancelled = true;
      eventSource.close();
      eventSourceRef.current = null;
    };
  }, [runId]);

  return { spans, loading };
}
