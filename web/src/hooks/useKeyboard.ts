import { useEffect, useCallback } from "react";
import type { AgentRun } from "../types/agent-run";

interface UseKeyboardOptions {
  runs: AgentRun[];
  selectedId: string | null;
  setSelectedId: (id: string | null) => void;
  detailOpen: boolean;
  setDetailOpen: (open: boolean) => void;
  paletteOpen: boolean;
  setPaletteOpen: (open: boolean) => void;
  openNewRun: () => void;
  filter: string;
  setFilter: (f: string) => void;
  activeTab?: number;
  setActiveTab?: (idx: number) => void;
  tabCount?: number;
}

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName.toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return true;
  if ((el as HTMLElement).isContentEditable) return true;
  return false;
}

export function useKeyboard({
  runs,
  selectedId,
  setSelectedId,
  detailOpen,
  setDetailOpen,
  paletteOpen,
  setPaletteOpen,
  openNewRun,
  filter,
  setFilter,
  activeTab = 0,
  setActiveTab,
  tabCount = 5,
}: UseKeyboardOptions) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Cmd+K / Ctrl+K: always open palette
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setPaletteOpen(true);
        return;
      }

      // Escape: always — close palette first, then detail
      if (e.key === "Escape") {
        if (paletteOpen) {
          setPaletteOpen(false);
          return;
        }
        if (detailOpen) {
          setDetailOpen(false);
          return;
        }
        return;
      }

      // Everything below is suppressed when input is focused
      if (isInputFocused()) return;

      const currentIndex = selectedId
        ? runs.findIndex((r) => r.id === selectedId)
        : -1;

      switch (e.key) {
        case "j": {
          if (runs.length === 0) return;
          const nextIndex = Math.min(currentIndex + 1, runs.length - 1);
          setSelectedId(runs[nextIndex].id);
          const el = document.querySelector(`[data-run-id="${runs[nextIndex].id}"]`);
          el?.scrollIntoView({ block: "nearest" });
          break;
        }

        case "k": {
          if (runs.length === 0) return;
          const prevIndex = Math.max(currentIndex <= 0 ? 0 : currentIndex - 1, 0);
          setSelectedId(runs[prevIndex].id);
          const el = document.querySelector(`[data-run-id="${runs[prevIndex].id}"]`);
          el?.scrollIntoView({ block: "nearest" });
          break;
        }

        case "Enter": {
          if (selectedId && !detailOpen) {
            setDetailOpen(true);
          }
          break;
        }

        case "n": {
          openNewRun();
          break;
        }

        case "1": {
          setFilter("all");
          break;
        }

        case "2": {
          setFilter("active");
          break;
        }

        case "3": {
          setFilter("succeeded");
          break;
        }

        case "4": {
          setFilter("failed");
          break;
        }

        case "Tab": {
          if (detailOpen && setActiveTab) {
            e.preventDefault();
            setActiveTab((activeTab + 1) % tabCount);
          }
          break;
        }

        default:
          return;
      }
    },
    [
      runs,
      selectedId,
      setSelectedId,
      detailOpen,
      setDetailOpen,
      paletteOpen,
      setPaletteOpen,
      openNewRun,
      filter,
      setFilter,
      activeTab,
      setActiveTab,
      tabCount,
    ],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
