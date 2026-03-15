import { useEffect, useCallback, type RefObject } from "react";
import type { AgentRun } from "../types/agent-run";

interface UseKeyboardNavigationOptions {
  runs: AgentRun[];
  selectedRunId: string | null | undefined;
  setSelectedRunId: (run: AgentRun | null) => void;
  isDetailOpen: boolean;
  closeDetail: () => void;
  searchInputRef: RefObject<HTMLInputElement | null>;
}

/**
 * Global keyboard shortcuts for navigating the run list.
 *
 * - j / k   — move selection down / up through runs
 * - Enter   — open detail for the selected run
 * - Escape  — close the detail view
 * - /       — focus the search input
 *
 * All shortcuts are suppressed when the active element is an input,
 * textarea, select, or contenteditable element.
 */
export function useKeyboardNavigation({
  runs,
  selectedRunId,
  setSelectedRunId,
  isDetailOpen,
  closeDetail,
  searchInputRef,
}: UseKeyboardNavigationOptions) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Guard: skip when focused on an editable element
      const tag = (e.target as HTMLElement)?.tagName?.toLowerCase();
      if (
        tag === "input" ||
        tag === "textarea" ||
        tag === "select" ||
        (e.target as HTMLElement)?.isContentEditable
      ) {
        // Exception: Escape should still close detail even from inputs
        if (e.key === "Escape" && isDetailOpen) {
          closeDetail();
        }
        return;
      }

      const currentIndex = selectedRunId
        ? runs.findIndex((r) => r.id === selectedRunId)
        : -1;

      switch (e.key) {
        case "j": {
          // Select next run (increment index)
          if (runs.length === 0) return;
          const nextIndex = Math.min(currentIndex + 1, runs.length - 1);
          setSelectedRunId(runs[nextIndex]);
          // Scroll into view
          const nextEl = document.querySelector(
            `[data-testid="run-card-${runs[nextIndex].id}"]`
          );
          nextEl?.scrollIntoView({ block: "nearest" });
          break;
        }

        case "k": {
          // Select previous run (decrement index)
          if (runs.length === 0) return;
          const prevIndex = Math.max(currentIndex <= 0 ? 0 : currentIndex - 1, 0);
          setSelectedRunId(runs[prevIndex]);
          // Scroll into view
          const prevEl = document.querySelector(
            `[data-testid="run-card-${runs[prevIndex].id}"]`
          );
          prevEl?.scrollIntoView({ block: "nearest" });
          break;
        }

        case "Enter": {
          // Open detail for selected run
          if (selectedRunId && !isDetailOpen) {
            const run = runs.find((r) => r.id === selectedRunId);
            if (run) {
              setSelectedRunId(run);
            }
          }
          break;
        }

        case "Escape": {
          // Close detail view
          if (isDetailOpen) {
            closeDetail();
          }
          break;
        }

        case "/": {
          // Focus search input
          e.preventDefault();
          searchInputRef.current?.focus();
          break;
        }

        default:
          return; // Don't prevent default for unhandled keys
      }
    },
    [runs, selectedRunId, setSelectedRunId, isDetailOpen, closeDetail, searchInputRef]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
