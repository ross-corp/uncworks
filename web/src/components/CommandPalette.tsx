import { useState, useEffect, useRef, useCallback } from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import type { AgentRun } from "../types/agent-run";

type ResultType = "run" | "command" | "filter";

interface PaletteResult {
  id: string;
  type: ResultType;
  label: string;
  detail?: string;
  value: string;
}

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  runs: AgentRun[];
  onSelectRun: (run: AgentRun) => void;
  onCommand: (cmd: string) => void;
}

const COMMANDS: PaletteResult[] = [
  { id: "cmd-new-run", type: "command", label: "New Run", detail: "Create a new agent run", value: "new-run" },
  { id: "cmd-toggle-theme", type: "command", label: "Toggle Theme", detail: "Switch light/dark", value: "toggle-theme" },
  { id: "cmd-show-all", type: "command", label: "Show All Runs", detail: "Clear filters", value: "filter-all" },
  { id: "cmd-show-active", type: "command", label: "Show Active", detail: "Filter to active runs", value: "filter-active" },
  { id: "cmd-show-succeeded", type: "command", label: "Show Succeeded", detail: "Filter to succeeded runs", value: "filter-succeeded" },
  { id: "cmd-show-failed", type: "command", label: "Show Failed", detail: "Filter to failed runs", value: "filter-failed" },
];

const MRU_KEY = "clean-ui-mru";

function loadMru(): string[] {
  try {
    const raw = localStorage.getItem(MRU_KEY);
    if (raw) return JSON.parse(raw);
  } catch { /* ignore */ }
  return [];
}

function saveMru(ids: string[]) {
  localStorage.setItem(MRU_KEY, JSON.stringify(ids.slice(0, 5)));
}

export default function CommandPalette({
  open,
  onClose,
  runs,
  onSelectRun,
  onCommand,
}: CommandPaletteProps) {
  const [query, setQuery] = useState("");
  const [highlightIndex, setHighlightIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Build results
  const results = buildResults(query, runs);

  // Reset on open
  useEffect(() => {
    if (open) {
      setQuery("");
      setHighlightIndex(0);
      // Auto-focus after render
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  // Clamp highlight
  useEffect(() => {
    if (highlightIndex >= results.length) {
      setHighlightIndex(Math.max(0, results.length - 1));
    }
  }, [results.length, highlightIndex]);

  // Scroll highlighted item into view
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.querySelector(`[data-index="${highlightIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [highlightIndex]);

  const execute = useCallback(
    (result: PaletteResult) => {
      // Update MRU
      const mru = loadMru().filter((id) => id !== result.id);
      mru.unshift(result.id);
      saveMru(mru);

      if (result.type === "run") {
        const run = runs.find((r) => r.id === result.value);
        if (run) onSelectRun(run);
      } else {
        onCommand(result.value);
      }
      onClose();
    },
    [runs, onSelectRun, onCommand, onClose],
  );

  function handleKeyDown(e: React.KeyboardEvent) {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setHighlightIndex((i) => Math.min(i + 1, results.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setHighlightIndex((i) => Math.max(i - 1, 0));
        break;
      case "Enter":
        e.preventDefault();
        if (results[highlightIndex]) {
          execute(results[highlightIndex]);
        }
        break;
      case "Escape":
        e.preventDefault();
        onClose();
        break;
    }
  }

  return (
    <DialogPrimitive.Root open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/50" />
        <DialogPrimitive.Content
          data-testid="command-palette"
          className="fixed inset-0 z-50 flex justify-center pointer-events-none"
          onKeyDown={handleKeyDown}
        >
          <DialogPrimitive.Title className="sr-only">Command Palette</DialogPrimitive.Title>
          <DialogPrimitive.Description className="sr-only">
            Search runs and execute commands
          </DialogPrimitive.Description>
          <div
            className="pointer-events-auto mt-[20vh] h-fit max-w-lg w-full mx-auto border shadow-lg"
            style={{
              backgroundColor: "var(--color-bg)",
              color: "var(--color-fg)",
              borderColor: "var(--color-border)",
            }}
          >
            {/* Search input */}
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setHighlightIndex(0);
              }}
              placeholder="Search runs, commands..."
              className="w-full px-4 py-3 text-sm outline-none border-b"
              style={{
                backgroundColor: "var(--color-bg)",
                color: "var(--color-fg)",
                borderColor: "var(--color-border)",
              }}
            />

            {/* Results list */}
            <div
              ref={listRef}
              className="overflow-y-auto"
              style={{ maxHeight: "400px" }}
            >
              {results.length === 0 && (
                <div
                  className="px-4 py-6 text-center text-sm"
                  style={{ color: "var(--color-muted)" }}
                >
                  No results
                </div>
              )}
              {renderGrouped(results, highlightIndex, execute)}
            </div>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}

function buildResults(query: string, runs: AgentRun[]): PaletteResult[] {
  const q = query.toLowerCase().trim();

  if (!q) {
    // Show MRU commands + recent runs
    const mru = loadMru();
    const mruResults: PaletteResult[] = [];
    for (const id of mru) {
      const cmd = COMMANDS.find((c) => c.id === id);
      if (cmd) { mruResults.push(cmd); continue; }
      const run = runs.find((r) => r.id === id || `run-${r.id}` === id);
      if (run) {
        mruResults.push({
          id: `run-${run.id}`,
          type: "run",
          label: run.name || run.id,
          detail: run.spec.prompt.slice(0, 60),
          value: run.id,
        });
      }
    }

    // Add remaining commands and recent runs
    const recentRuns: PaletteResult[] = runs.slice(0, 5).map((r) => ({
      id: `run-${r.id}`,
      type: "run" as ResultType,
      label: r.name || r.id,
      detail: r.spec.prompt.slice(0, 60),
      value: r.id,
    }));

    const all = [...mruResults];
    for (const cmd of COMMANDS) {
      if (!all.find((r) => r.id === cmd.id)) all.push(cmd);
    }
    for (const rr of recentRuns) {
      if (!all.find((r) => r.id === rr.id)) all.push(rr);
    }
    return all;
  }

  const results: PaletteResult[] = [];

  // Search runs
  for (const run of runs) {
    const haystack = [run.name, run.id, run.spec.prompt, ...run.spec.repos.map((r) => r.url)]
      .join(" ")
      .toLowerCase();
    if (haystack.includes(q)) {
      results.push({
        id: `run-${run.id}`,
        type: "run",
        label: run.name || run.id,
        detail: run.spec.prompt.slice(0, 60),
        value: run.id,
      });
    }
  }

  // Search commands
  for (const cmd of COMMANDS) {
    if (cmd.label.toLowerCase().includes(q) || (cmd.detail && cmd.detail.toLowerCase().includes(q))) {
      results.push(cmd);
    }
  }

  return results;
}

function renderGrouped(
  results: PaletteResult[],
  highlightIndex: number,
  onSelect: (r: PaletteResult) => void,
) {
  const groups: { type: ResultType; label: string; items: { result: PaletteResult; globalIndex: number }[] }[] = [];
  const typeLabels: Record<ResultType, string> = { run: "Runs", command: "Commands", filter: "Filters" };

  let idx = 0;
  for (const r of results) {
    let group = groups.find((g) => g.type === r.type);
    if (!group) {
      group = { type: r.type, label: typeLabels[r.type], items: [] };
      groups.push(group);
    }
    group.items.push({ result: r, globalIndex: idx });
    idx++;
  }

  return groups.map((group) => (
    <div key={group.type}>
      <div
        className="px-4 py-1 text-xs font-medium"
        style={{ color: "var(--color-muted)" }}
      >
        {group.label}
      </div>
      {group.items.map(({ result, globalIndex }) => {
        const isHighlighted = globalIndex === highlightIndex;
        return (
          <button
            key={result.id}
            data-index={globalIndex}
            onClick={() => onSelect(result)}
            className="w-full text-left px-4 py-2 text-sm flex items-center justify-between cursor-pointer"
            style={{
              backgroundColor: isHighlighted ? "var(--color-border)" : "transparent",
              color: "var(--color-fg)",
            }}
          >
            <span>{result.label}</span>
            {result.detail && (
              <span
                className="text-xs truncate ml-4"
                style={{ color: "var(--color-muted)" }}
              >
                {result.detail}
              </span>
            )}
          </button>
        );
      })}
    </div>
  ));
}
