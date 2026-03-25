import { useEffect, useRef, type DependencyList } from "react";

/**
 * usePoll — runs `fn` immediately on mount and repeats every `intervalMs` ms.
 * A `cancelled` guard prevents stale state updates after unmount.
 * Pass `deps` to re-start the poll when dependencies change.
 */
export function usePoll(
  fn: () => Promise<void>,
  intervalMs: number,
  deps: DependencyList = [],
): void {
  // Keep a stable ref to fn so callers don't need to memoize it
  const fnRef = useRef(fn);
  fnRef.current = fn;

  useEffect(() => {
    let cancelled = false;

    const invoke = async () => {
      if (!cancelled) {
        await fnRef.current();
      }
    };

    invoke();
    const id = setInterval(invoke, intervalMs);

    return () => {
      cancelled = true;
      clearInterval(id);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, ...deps]);
}
