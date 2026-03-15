import { useState, useEffect, useRef } from "react";
import { useClient, mapEvent } from "./useClient";

export function useWatchRun(
  runId: string | null,
  onPhaseChange?: (newPhase: string) => void,
) {
  const client = useClient();
  const [logLines, setLogLines] = useState<string[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const onPhaseChangeRef = useRef(onPhaseChange);
  onPhaseChangeRef.current = onPhaseChange;

  useEffect(() => {
    if (!runId) {
      setLogLines([]);
      setIsStreaming(false);
      return;
    }

    setLogLines([]);
    setIsStreaming(true);

    const abort = client.watchAgentRun(
      runId,
      (rawEvent) => {
        const event = mapEvent(rawEvent);
        if (event.type === "log") {
          setLogLines((prev) => [...prev, event.payload]);
        } else if (event.type === "phase_changed") {
          onPhaseChangeRef.current?.(event.payload);
        }
      },
      (_err) => {
        setIsStreaming(false);
      }
    );

    abortRef.current = abort;

    return () => {
      abort.abort();
      abortRef.current = null;
      setIsStreaming(false);
    };
  }, [runId, client]);

  return { logLines, isStreaming };
}
